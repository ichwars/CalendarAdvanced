package crypto

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordHashingUsesArgon2idAndVerifies(t *testing.T) {
	hash, err := HashPassword("Sicheres-Passwort-123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !strings.Contains(hash, "$argon2id$") {
		t.Fatalf("expected argon2id hash, got %q", hash)
	}
	if !VerifyPassword(hash, "Sicheres-Passwort-123!") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("wrong password verified")
	}
}

func TestTOTPVerification(t *testing.T) {
	secret, err := NewTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0)
	code, err := TOTPCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyTOTP(secret, code, now) {
		t.Fatal("expected totp code to verify")
	}
	if VerifyTOTP(secret, "000000", now) && code != "000000" {
		t.Fatal("unexpected code accepted")
	}
}

func TestTokenCipherRoundTrip(t *testing.T) {
	cipher, err := NewTokenCipher("01234567890123456789012345678901")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := cipher.Encrypt("secret-access-token")
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "secret-access-token" {
		t.Fatal("token was not encrypted")
	}
	plain, err := cipher.Decrypt(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "secret-access-token" {
		t.Fatalf("unexpected decrypted token %q", plain)
	}
}
