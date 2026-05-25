package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (db *DB) Migrate(dir string) error {
	if err := db.ExecScript(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL);`); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		rows, err := db.Query(`SELECT version FROM schema_migrations WHERE version = ?`, name)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		script := "BEGIN IMMEDIATE;\n" + string(body) + fmt.Sprintf("\nINSERT INTO schema_migrations(version, applied_at) VALUES('%s', strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ','now'));\nCOMMIT;", strings.ReplaceAll(name, "'", "''"))
		if err := db.ExecScript(script); err != nil {
			_ = db.ExecScript("ROLLBACK;")
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
