package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	recorder := performCORS([]string{"https://app.example.com"}, "https://app.example.com")
	if recorder.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatal("configured origin should be allowed")
	}
}

func TestCORSRejectsWildcardAndUnlistedOrigins(t *testing.T) {
	cases := []struct {
		allowed []string
		origin  string
	}{
		{[]string{"*"}, "https://attacker.example"},
		{[]string{"https://app.example.com"}, "https://attacker.example"},
	}
	for _, test := range cases {
		recorder := performCORS(test.allowed, test.origin)
		if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("unexpected CORS access for %s", test.origin)
		}
	}
}

func TestCORSDoesNotExposeServiceKeyHeader(t *testing.T) {
	recorder := performCORS([]string{"https://app.example.com"}, "https://app.example.com")
	if strings.Contains(recorder.Header().Get("Access-Control-Allow-Headers"), "X-Service-Key") {
		t.Fatal("service key must not be available to browser clients")
	}
}

func performCORS(allowed []string, requestOrigin string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS(allowed))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", requestOrigin)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
