package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// PlatformClaims defines the standard JWT claims for Araya Business Hub Platform.
// Exactly conforming to PRD v1.0 and Panduan API Bersama v1.0.
type PlatformClaims struct {
	UserID      int    `json:"user_id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Division    string `json:"division"`
	UnitUsahaID *int   `json:"unit_usaha_id,omitempty"`
	PartnerID   *int   `json:"partner_id,omitempty"`
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired     = errors.New("token telah kedaluwarsa")
	ErrTokenInvalid     = errors.New("token tidak valid")
	ErrInvalidSignature = errors.New("signature token tidak sesuai")
	ErrMissingClaims    = errors.New("claim wajib pada token tidak lengkap")
)

// GenerateToken creates a signed JWT string using HMAC-SHA256.
func GenerateToken(claims PlatformClaims, secret string, expirationHours int) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret cannot be empty")
	}

	if expirationHours <= 0 {
		expirationHours = 8 // Default 8 hours per PRD v1.0
	}

	now := time.Now().UTC()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(expirationHours) * time.Hour))
	claims.Issuer = "araya-hub-m1"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses, validates signature, and extracts PlatformClaims.
func ValidateToken(tokenStr, secret string) (*PlatformClaims, error) {
	if secret == "" {
		return nil, errors.New("JWT secret cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &PlatformClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*PlatformClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	// Validate mandatory claims per PRD v1.0
	if claims.UserID <= 0 || claims.Name == "" || claims.Role == "" {
		return nil, ErrMissingClaims
	}

	return claims, nil
}
