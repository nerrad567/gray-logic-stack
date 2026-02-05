package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
)

// seedPasswordBytes is the number of random bytes for the seed owner password.
const seedPasswordBytes = 16

// SeedOwner creates the initial owner account on first boot if no users exist.
// The generated password is printed to stdout and logged — it must be changed immediately.
// Returns the generated password (empty string if seeding was skipped).
func SeedOwner(ctx context.Context, userRepo UserRepository, logger *slog.Logger) (string, error) {
	count, err := userRepo.Count(ctx)
	if err != nil {
		return "", fmt.Errorf("checking user count: %w", err)
	}

	if count > 0 {
		logger.Info("users exist, skipping owner seed")
		return "", nil
	}

	// Generate a random password
	passwordBytes := make([]byte, seedPasswordBytes)
	if _, err := rand.Read(passwordBytes); err != nil { //nolint:govet // shadow: err re-declared in nested scope
		return "", fmt.Errorf("generating seed password: %w", err)
	}
	password := hex.EncodeToString(passwordBytes)

	hash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hashing seed password: %w", err)
	}

	owner := &User{
		Username:     "owner",
		DisplayName:  "System Owner",
		PasswordHash: hash,
		Role:         RoleOwner,
		IsActive:     true,
	}

	if err := userRepo.Create(ctx, owner); err != nil {
		return "", fmt.Errorf("creating seed owner: %w", err)
	}

	logger.Warn("seed owner account created",
		"username", "owner",
		"password", password,
		"action_required", "change this password immediately",
	)

	return password, nil
}
