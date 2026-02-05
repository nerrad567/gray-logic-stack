package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims extends JWT standard claims with Gray Logic-specific fields.
type CustomClaims struct {
	jwt.RegisteredClaims
	Role      Role   `json:"role"`
	SessionID string `json:"sid"`
}

// GenerateAccessToken creates a signed JWT access token for a user.
// Access tokens are short-lived (configured TTL) and validated by signature only (no DB hit).
func GenerateAccessToken(user *User, secret string, ttlMinutes int) (string, error) {
	if ttlMinutes <= 0 {
		ttlMinutes = 15 //nolint:mnd // default 15-minute access token TTL
	}

	now := time.Now()
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttlMinutes) * time.Minute)),
			ID:        uuid.NewString(),
		},
		Role:      user.Role,
		SessionID: uuid.NewString(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken creates a cryptographically random refresh token (256-bit).
// The raw token is returned to the client; the hash is stored in the database.
func GenerateRefreshToken() (raw string, err error) {
	b := make([]byte, 32) //nolint:mnd // 256-bit token
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ParseToken validates and parses a JWT access token, returning the custom claims.
// It checks the signature, expiry, and required fields.
func ParseToken(tokenString, secret string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: missing subject", ErrTokenInvalid)
	}

	if claims.Role == "" {
		return nil, fmt.Errorf("%w: missing role", ErrTokenInvalid)
	}

	return claims, nil
}
