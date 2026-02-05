package auth

import "testing"

// ─── Password hashing (Argon2id — intentionally slow) ───────────────

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashPassword("correct-horse-battery-staple") //nolint:errcheck // benchmark
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		b.Fatalf("HashPassword: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword("correct-horse-battery-staple", hash) //nolint:errcheck // benchmark
	}
}

// ─── JWT tokens (per-request hot path) ──────────────────────────────

func BenchmarkGenerateAccessToken(b *testing.B) {
	user := &User{ID: "usr-bench", Role: RoleAdmin}
	secret := []byte("benchmark-secret-key-32-bytes-xx")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateAccessToken(user, secret, 15) //nolint:errcheck // benchmark
	}
}

func BenchmarkParseToken(b *testing.B) {
	user := &User{ID: "usr-bench", Role: RoleAdmin}
	secret := []byte("benchmark-secret-key-32-bytes-xx")

	token, err := GenerateAccessToken(user, secret, 15)
	if err != nil {
		b.Fatalf("GenerateAccessToken: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseToken(token, secret) //nolint:errcheck // benchmark
	}
}

func BenchmarkGenerateRefreshToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRefreshToken() //nolint:errcheck // benchmark
	}
}
