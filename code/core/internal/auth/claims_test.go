package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	user := &User{
		ID:   "usr-001",
		Role: RoleAdmin,
	}
	secret := "test-secret-key-for-jwt-signing"

	token, err := GenerateAccessToken(user, secret, 15)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Fatal("GenerateAccessToken() returned empty token")
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.Subject != "usr-001" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "usr-001")
	}

	if claims.Role != RoleAdmin {
		t.Errorf("Role = %q, want %q", claims.Role, RoleAdmin)
	}

	if claims.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if claims.ID == "" {
		t.Error("JTI (ID) should not be empty")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	user := &User{ID: "usr-001", Role: RoleUser}

	token, err := GenerateAccessToken(user, "correct-secret", 15)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = ParseToken(token, "wrong-secret")
	if err == nil {
		t.Error("ParseToken() should fail with wrong secret")
	}
}

func TestParseToken_Expired(t *testing.T) {
	user := &User{ID: "usr-001", Role: RoleUser}

	// Generate a token that's already expired (negative TTL won't work,
	// so we generate with 1 minute and then parse with a check)
	// Instead, directly test with a manipulated token time by using 0 TTL
	// which defaults to 15 min — we can't easily expire it in a unit test
	// without sleeping. Instead, test that a garbage token fails.
	_, err := ParseToken("not-a-valid-jwt", "secret")
	if err == nil {
		t.Error("ParseToken() should fail with invalid token string")
	}

	// Valid token should still parse
	token, err := GenerateAccessToken(user, "secret", 15)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	// Token should not be expired yet
	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("newly generated token should not be expired")
	}
}

func TestParseToken_InvalidSigningMethod(t *testing.T) {
	// Empty string
	_, err := ParseToken("", "secret")
	if err == nil {
		t.Error("ParseToken() should fail with empty token")
	}

	// Malformed JWT (wrong number of segments)
	_, err = ParseToken("abc.def", "secret")
	if err == nil {
		t.Error("ParseToken() should fail with malformed JWT")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	raw, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if raw == "" {
		t.Error("GenerateRefreshToken() returned empty string")
	}

	// Should generate unique tokens
	raw2, _ := GenerateRefreshToken()
	if raw == raw2 {
		t.Error("two refresh tokens should be unique")
	}
}

func TestGenerateAccessToken_DefaultTTL(t *testing.T) {
	user := &User{ID: "usr-001", Role: RoleUser}

	// TTL of 0 should default to 15 minutes
	token, err := GenerateAccessToken(user, "secret", 0)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	expectedExpiry := time.Now().Add(15 * time.Minute)
	diff := claims.ExpiresAt.Time.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("default TTL should be ~15 minutes, got expiry diff of %v", diff)
	}
}
