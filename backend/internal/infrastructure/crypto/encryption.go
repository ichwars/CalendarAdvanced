package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

type TokenCipher struct {
	aead cipher.AEAD
}

func NewTokenCipher(secret string) (*TokenCipher, error) {
	if secret == "" {
		return nil, errors.New("token encryption key is required")
	}
	var key []byte
	if decoded, err := base64.StdEncoding.DecodeString(secret); err == nil && len(decoded) == 32 {
		key = decoded
	} else if decoded, err := hex.DecodeString(secret); err == nil && len(decoded) == 32 {
		key = decoded
	} else {
		sum := sha256.Sum256([]byte(secret))
		key = sum[:]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &TokenCipher{aead: aead}, nil
}

func (c *TokenCipher) Encrypt(plain string) (string, error) {
	if c == nil || c.aead == nil {
		return "", errors.New("token cipher is not configured")
	}
	nonce, err := RandomBytes(c.aead.NonceSize())
	if err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *TokenCipher) Decrypt(encoded string) (string, error) {
	if c == nil || c.aead == nil {
		return "", errors.New("token cipher is not configured")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(payload) < c.aead.NonceSize() {
		return "", errors.New("encrypted payload is invalid")
	}
	nonce := payload[:c.aead.NonceSize()]
	ciphertext := payload[c.aead.NonceSize():]
	plain, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
