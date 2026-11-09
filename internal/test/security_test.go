package test

import (
	m1domain "araya-business-hub/be/internal/modules/m1/domain"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"araya-business-hub/be/internal/platform/security"
	"github.com/golang-jwt/jwt/v5"
)

func TestSecurity_MissingToken_Rejected(t *testing.T) {
	r, _ := SetupFullTestApp()
	for _, path := range []string{"/api/m1/v1/users", "/api/m2/v1/admin/business-units", "/api/m3/v1/partners"} {
		w := DoReq(r, http.MethodGet, path, "", nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for missing token on %s, got %d", path, w.Code)
		}
	}
}

func TestSecurity_InvalidSignature_Rejected(t *testing.T) {
	r, _ := SetupFullTestApp()
	fakeToken, _ := security.GenerateToken(security.PlatformClaims{
		UserID: 99, Name: "Attacker", Role: "admin",
	}, "wrong_secret_key_wrong_secret_key_12345", 8)
	w := DoReq(r, http.MethodGet, "/api/m1/v1/users", fakeToken, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for token with invalid signature, got %d", w.Code)
	}
}

func TestSecurity_ExpiredToken_Rejected(t *testing.T) {
	r, _ := SetupFullTestApp()
	now := time.Now().Add(-2 * time.Hour)
	claims := security.PlatformClaims{
		UserID: 1, Name: "Admin", Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredStr, _ := tok.SignedString([]byte(TestJWTSecret))
	w := DoReq(r, http.MethodGet, "/api/m1/v1/users", expiredStr, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestSecurity_InsufficientPermission_Forbidden(t *testing.T) {
	r, _ := SetupFullTestApp()
	staff := &m1domain.User{Email: "staff@araya.com", Name: "Staff", Role: "staff", IsActive: true}
	_ = testM1.UserRepo.Create(context.Background(), staff)
	staffToken := GenerateToken(staff.ID, "staff@araya.com", "staff")
	w := DoReq(r, http.MethodDelete, "/api/m1/v1/users/2", staffToken, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for staff deleting user, got %d", w.Code)
	}
}

func TestSecurity_PublicEndpoints_AccessibleWithoutAuth(t *testing.T) {
	r, _ := SetupFullTestApp()
	for _, path := range []string{"/health", "/api/v1/health", "/api/m5/v1/business-units", "/api/m5/v1/products"} {
		w := DoReq(r, http.MethodGet, path, "", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for public endpoint %s, got %d", path, w.Code)
		}
	}
}

func TestSecurity_PasswordRedacted_InUserResponses(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	w := DoReq(r, http.MethodGet, "/api/m1/v1/users", token, nil)
	bodyStr := strings.ToLower(w.Body.String())
	if strings.Contains(bodyStr, "password_hash") || strings.Contains(bodyStr, "admin1234!") {
		t.Fatalf("security violation: password hash or plaintext password leaked in response: %s", bodyStr)
	}
}

func TestSecurity_TokenExpiryDuration(t *testing.T) {
	claims := security.PlatformClaims{UserID: 1, Name: "Admin", Role: "admin"}
	tok, err := security.GenerateToken(claims, TestJWTSecret, 8)
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}
	parsed, err := security.ValidateToken(tok, TestJWTSecret)
	if err != nil || parsed.ExpiresAt == nil || parsed.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("token expiration validation failed")
	}
}
