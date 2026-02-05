package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters — OWASP 2025 recommendation.
const (
	argonTime    = 3         // iterations
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 1         // parallelism
	argonKeyLen  = 32        // output hash length
	argonSaltLen = 16        // salt length
)

// HashPassword hashes a plaintext password using Argon2id and returns it
// in PHC string format: $argon2id$v=19$m=65536,t=3,p=1$<salt>$<hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword checks a plaintext password against an Argon2id PHC hash string.
// Returns true if the password matches.
func VerifyPassword(password, encodedHash string) (bool, error) {
	salt, hash, params, err := decodePHC(encodedHash)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(hash))) //nolint:gosec // G115: hash length always fits uint32

	return subtle.ConstantTimeCompare(hash, candidate) == 1, nil
}

type argonParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

// decodePHC parses an Argon2id PHC string format into its components.
func decodePHC(encoded string) (salt, hash []byte, params argonParams, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 { //nolint:mnd // PHC format has exactly 6 $-delimited parts
		return nil, nil, params, fmt.Errorf("invalid PHC hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, params, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil { //nolint:govet // shadow: err re-declared in nested scope
		return nil, nil, params, fmt.Errorf("parsing version: %w", err)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads); err != nil { //nolint:govet // shadow: err re-declared in nested scope
		return nil, nil, params, fmt.Errorf("parsing parameters: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, params, fmt.Errorf("decoding salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, params, fmt.Errorf("decoding hash: %w", err)
	}

	return salt, hash, params, nil
}
