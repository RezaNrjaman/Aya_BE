package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/platform/config"
	platformHttp "araya-business-hub/be/internal/platform/http"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	cfg := &config.Config{
		Port:           "8080",
		AppEnv:         "test",
		AllowedOrigins: []string{"*"},
	}
	deps := platformHttp.RouterDependencies{
		Config: cfg,
		DB:     nil,
	}
	return platformHttp.NewRouter(deps)
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", res["status"])
	}

	// Verify X-Request-ID header was automatically generated and set
	reqID := w.Header().Get(middleware.RequestIDHeader)
	if reqID == "" {
		t.Errorf("expected X-Request-ID header to be present")
	}
}

func TestAPIHealthEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res response.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected success true")
	}

	dataMap, ok := res.Data.(map[string]any)
	if !ok || dataMap["status"] != "running" {
		t.Errorf("expected data.status to be 'running', got %v", res.Data)
	}
}

func TestNotFoundHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/non-existent-endpoint", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var res response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if res.Success {
		t.Fatalf("expected success false")
	}
	if res.Error.Code != response.CodeNotFound {
		t.Errorf("expected code NOT_FOUND, got %s", res.Error.Code)
	}
}
