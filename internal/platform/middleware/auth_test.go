package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/response"
	"araya-business-hub/be/internal/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const secret = "test_super_secret_key_32_characters_minimum"
const serviceKey = "test_service_secret_key_1234"

func init() {
	gin.SetMode(gin.TestMode)
}

func TestJWTAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Authenticate(secret))
	r.GET("/protected", func(c *gin.Context) {
		user := middleware.GetAuthUser(c)
		response.OK(c, gin.H{
			"user_id":  user.UserID,
			"name":     user.Name,
			"role":     user.Role,
			"division": user.Division,
		})
	})

	t.Run("Missing Authorization Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Malformed Authorization Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic some_base64_value")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Expired JWT Token", func(t *testing.T) {
		claims := security.PlatformClaims{
			UserID:   99,
			Name:     "Expired User",
			Role:     "staff",
			Division: "keuangan",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokStr, _ := tok.SignedString([]byte(secret))

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Valid JWT Token", func(t *testing.T) {
		tokStr, err := security.GenerateToken(security.PlatformClaims{
			UserID:   10,
			Name:     "Budi Santoso",
			Role:     "admin",
			Division: "operasional",
		}, secret, 8)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var res response.SuccessResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		data := res.Data.(map[string]any)
		if data["user_id"] != float64(10) || data["name"] != "Budi Santoso" {
			t.Fatalf("unexpected user in response: %+v", data)
		}
	})
}

func TestServiceAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ServiceAuthenticate(serviceKey))
	r.GET("/service-endpoint", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "service_verified"})
	})

	t.Run("Missing Service Key", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/service-endpoint", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Invalid Service Key", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/service-endpoint", nil)
		req.Header.Set(middleware.ServiceKeyHeader, "wrong_secret_key")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Valid Service Key", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/service-endpoint", nil)
		req.Header.Set(middleware.ServiceKeyHeader, serviceKey)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}
