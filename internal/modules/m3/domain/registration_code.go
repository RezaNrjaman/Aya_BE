package domain

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRegistrationCode returns an unguessable partner tracking code for
// the public registration status page (ARY-002: no sequential-ID lookups).
func GenerateRegistrationCode() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
