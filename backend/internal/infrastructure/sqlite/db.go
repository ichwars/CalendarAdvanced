package sqlite

/*
#cgo LDFLAGS: -lsqlite3
#include <sqlite3.h>
#include <stdlib.h>

static int ck_bind_text(sqlite3_stmt* stmt, int idx, const char* text) {
    return sqlite3_bind_text(stmt, idx, text, -1, SQLITE_TRANSIENT);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type DB struct {
	mu  sync.Mutex
	raw *C.sqlite3
}

type Row map[string]string

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var raw *C.sqlite3
	flags := C.SQLITE_OPEN_READWRITE | C.SQLITE_OPEN_CREATE | C.SQLITE_OPEN_FULLMUTEX
	if rc := C.sqlite3_open_v2(cpath, &raw, C.int(flags), nil); rc != C.SQLITE_OK {
		msg := "sqlite open failed"
		if raw != nil {
			msg = C.GoString(C.sqlite3_errmsg(raw))
			C.sqlite3_close(raw)
		}
		return nil, errors.New(msg)
	}
	db := &DB{raw: raw}
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	} {
		if err := db.ExecScript(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.raw == nil {
		return nil
	}
	if rc := C.sqlite3_close(db.raw); rc != C.SQLITE_OK {
		return fmt.Errorf("sqlite close failed: %d", int(rc))
	}
	db.raw = nil
	return nil
}

func (db *DB) ExecScript(script string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	cscript := C.CString(script)
	defer C.free(unsafe.Pointer(cscript))
	var errmsg *C.char
	if rc := C.sqlite3_exec(db.raw, cscript, nil, nil, &errmsg); rc != C.SQLITE_OK {
		defer C.sqlite3_free(unsafe.Pointer(errmsg))
		return errors.New(C.GoString(errmsg))
	}
	return nil
}

func (db *DB) Exec(query string, args ...any) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	stmt, err := db.prepareLocked(query)
	if err != nil {
		return err
	}
	defer C.sqlite3_finalize(stmt)
	if err := db.bindLocked(stmt, args...); err != nil {
		return err
	}
	rc := C.sqlite3_step(stmt)
	if rc != C.SQLITE_DONE && rc != C.SQLITE_ROW {
		return db.lastErrorLocked()
	}
	return nil
}

func (db *DB) Query(query string, args ...any) ([]Row, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	stmt, err := db.prepareLocked(query)
	if err != nil {
		return nil, err
	}
	defer C.sqlite3_finalize(stmt)
	if err := db.bindLocked(stmt, args...); err != nil {
		return nil, err
	}
	rows := make([]Row, 0)
	for {
		rc := C.sqlite3_step(stmt)
		switch rc {
		case C.SQLITE_ROW:
			count := int(C.sqlite3_column_count(stmt))
			row := Row{}
			for i := 0; i < count; i++ {
				name := C.GoString(C.sqlite3_column_name(stmt, C.int(i)))
				if C.sqlite3_column_type(stmt, C.int(i)) == C.SQLITE_NULL {
					row[name] = ""
					continue
				}
				row[name] = C.GoString((*C.char)(unsafe.Pointer(C.sqlite3_column_text(stmt, C.int(i)))))
			}
			rows = append(rows, row)
		case C.SQLITE_DONE:
			return rows, nil
		default:
			return nil, db.lastErrorLocked()
		}
	}
}

func (db *DB) LastInsertID() int64 {
	db.mu.Lock()
	defer db.mu.Unlock()
	return int64(C.sqlite3_last_insert_rowid(db.raw))
}

func (db *DB) prepareLocked(query string) (*C.sqlite3_stmt, error) {
	cquery := C.CString(query)
	defer C.free(unsafe.Pointer(cquery))
	var stmt *C.sqlite3_stmt
	if rc := C.sqlite3_prepare_v2(db.raw, cquery, -1, &stmt, nil); rc != C.SQLITE_OK {
		return nil, db.lastErrorLocked()
	}
	return stmt, nil
}

func (db *DB) bindLocked(stmt *C.sqlite3_stmt, args ...any) error {
	for i, arg := range args {
		idx := C.int(i + 1)
		var rc C.int
		switch v := arg.(type) {
		case nil:
			rc = C.sqlite3_bind_null(stmt, idx)
		case string:
			cs := C.CString(v)
			rc = C.ck_bind_text(stmt, idx, cs)
			C.free(unsafe.Pointer(cs))
		case []byte:
			cs := C.CString(string(v))
			rc = C.ck_bind_text(stmt, idx, cs)
			C.free(unsafe.Pointer(cs))
		case int:
			rc = C.sqlite3_bind_int64(stmt, idx, C.sqlite3_int64(v))
		case int64:
			rc = C.sqlite3_bind_int64(stmt, idx, C.sqlite3_int64(v))
		case bool:
			if v {
				rc = C.sqlite3_bind_int64(stmt, idx, 1)
			} else {
				rc = C.sqlite3_bind_int64(stmt, idx, 0)
			}
		case time.Time:
			var s string
			if !v.IsZero() {
				s = FormatTime(v)
			}
			cs := C.CString(s)
			rc = C.ck_bind_text(stmt, idx, cs)
			C.free(unsafe.Pointer(cs))
		default:
			cs := C.CString(fmt.Sprint(v))
			rc = C.ck_bind_text(stmt, idx, cs)
			C.free(unsafe.Pointer(cs))
		}
		if rc != C.SQLITE_OK {
			return db.lastErrorLocked()
		}
	}
	return nil
}

func (db *DB) lastErrorLocked() error {
	return errors.New(C.GoString(C.sqlite3_errmsg(db.raw)))
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func Int64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func Int(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func Bool(s string) bool {
	return s == "1" || s == "true"
}
