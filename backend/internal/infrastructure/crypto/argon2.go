package crypto

/*
#cgo LDFLAGS: -l:libargon2.so.1
#include <stdlib.h>
#include <stdint.h>
#include <stddef.h>

int argon2id_hash_encoded(uint32_t t_cost, uint32_t m_cost, uint32_t parallelism,
    const void *pwd, size_t pwdlen, const void *salt, size_t saltlen,
    size_t hashlen, char *encoded, size_t encodedlen);
int argon2id_verify(const char *encoded, const void *pwd, size_t pwdlen);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const (
	argonTime        = 3
	argonMemoryKiB   = 64 * 1024
	argonParallelism = 1
	argonSaltLength  = 16
	argonHashLength  = 32
	argonEncodedSize = 256
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is required")
	}
	salt, err := RandomBytes(argonSaltLength)
	if err != nil {
		return "", err
	}
	pwd := C.CBytes([]byte(password))
	defer C.free(pwd)
	saltPtr := C.CBytes(salt)
	defer C.free(saltPtr)
	encoded := (*C.char)(C.malloc(C.size_t(argonEncodedSize)))
	if encoded == nil {
		return "", errors.New("argon2 allocation failed")
	}
	defer C.free(unsafe.Pointer(encoded))
	result := C.argon2id_hash_encoded(
		C.uint32_t(argonTime),
		C.uint32_t(argonMemoryKiB),
		C.uint32_t(argonParallelism),
		pwd,
		C.size_t(len(password)),
		saltPtr,
		C.size_t(len(salt)),
		C.size_t(argonHashLength),
		encoded,
		C.size_t(argonEncodedSize),
	)
	if result != 0 {
		return "", errors.New("argon2id hashing failed")
	}
	return C.GoString(encoded), nil
}

func VerifyPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	encoded := C.CString(hash)
	defer C.free(unsafe.Pointer(encoded))
	pwd := C.CBytes([]byte(password))
	defer C.free(pwd)
	return C.argon2id_verify(encoded, pwd, C.size_t(len(password))) == 0
}
