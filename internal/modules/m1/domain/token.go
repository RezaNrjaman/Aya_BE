package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

var (
	// ErrActivationTokenInvalid is returned for unknown or already-used tokens.
	ErrActivationTokenInvalid = errors.New("token aktivasi tidak valid atau sudah digunakan")
	// ErrActivationTokenExpired is returned when the invitation token is past
	// its lifetime.
	ErrActivationTokenExpired = errors.New("token aktivasi sudah kedaluwarsa, minta tautan baru")
)

// ActivationTokenTTL is how long an activation / password-reset invitation
// stays valid after being issued.
const ActivationTokenTTL = 72 * time.Hour

// GenerateActivationToken returns a fresh single-use invitation token. The
// plaintext token is meant to be delivered out-of-band (email); only the
// returned hash may be persisted.
func GenerateActivationToken() (token string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	token = "act_" + hex.EncodeToString(raw)
	return token, HashToken(token), nil
}

// HashToken derives the stored SHA-256 hash of an invitation token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateRegistrationCode returns an unguessable partner tracking code used
// by the public registration status page (ARY-002: no sequential-ID lookups).
func GenerateRegistrationCode() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
