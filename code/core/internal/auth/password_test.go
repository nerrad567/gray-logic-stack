package auth

import (
	"strings"
	"testing"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Verify the hash is in PHC format
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("hash should start with $argon2id$, got %q", hash)
	}

	// Correct password should verify
	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !ok {
		t.Error("VerifyPassword() should return true for correct password")
	}
}

func TestHashPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if ok {
		t.Error("VerifyPassword() should return false for wrong password")
	}
}

func TestHashPassword_UniqueSalts(t *testing.T) {
	password := "same-password"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("two hashes of the same password should have different salts")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"not PHC", "plaintext"},
		{"wrong algorithm", "$bcrypt$v=19$m=65536,t=3,p=1$salt$hash"},
		{"too few parts", "$argon2id$v=19$m=65536,t=3,p=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword("password", tt.hash)
			if err == nil {
				t.Error("VerifyPassword() should return error for invalid hash format")
			}
		})
	}
}

func TestHashPassword_PHCFormat(t *testing.T) {
	hash, err := HashPassword("test")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("PHC format should have 6 $-delimited parts, got %d: %q", len(parts), hash)
	}

	if parts[1] != "argon2id" {
		t.Errorf("algorithm should be argon2id, got %q", parts[1])
	}

	if parts[2] != "v=19" {
		t.Errorf("version should be v=19, got %q", parts[2])
	}

	if parts[3] != "m=65536,t=3,p=1" {
		t.Errorf("params should be m=65536,t=3,p=1, got %q", parts[3])
	}
}
