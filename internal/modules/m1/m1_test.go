package m1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/modules/m1"
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/config"
	platformHttp "araya-business-hub/be/internal/platform/http"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

const testSecret = "test_jwt_secret_key_minimum_32_characters"

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestApp() (*gin.Engine, *m1.Module) {
	reg := rbac.NewRegistry()
	_ = rbac.SeedPlatformPermissions(reg)
	repo := rbac.NewInMemoryRepository()
	_ = rbac.SeedInitialRoles(context.Background(), repo)
	policy := rbac.NewPolicyService(reg, repo)
	auditSvc := audit.NewService(audit.NewInMemoryRepository())

	cfg := &config.Config{Port: "8080", AppEnv: "test", JWTSecret: testSecret, JWTExpirationHours: 8}
	router := platformHttp.NewRouter(platformHttp.RouterDependencies{
		Config: cfg, RBACRegistry: reg, RBACRepo: repo, PolicyService: policy, AuditService: auditSvc,
	})
	mod := m1.InitModule(router, testSecret, 8, repo, policy, auditSvc)
	return router, mod
}

func doReq(r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func loginAdmin(r *gin.Engine) string {
	w := doReq(r, http.MethodPost, "/api/m1/v1/login", "", dto.LoginRequest{
		Email: "admin@araya.com", Password: "Admin123!",
	})
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	data := res.Data.(map[string]any)
	return data["token"].(string)
}

func TestM1_LoginSuccess(t *testing.T) {
	router, _ := setupTestApp()
	token := loginAdmin(router)
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
}

func TestM1_LoginInvalidCredentials(t *testing.T) {
	router, _ := setupTestApp()
	w := doReq(router, http.MethodPost, "/api/m1/v1/login", "", dto.LoginRequest{
		Email: "admin@araya.com", Password: "WrongPassword!",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", w.Code)
	}
}

func TestM1_GetProfileContractK01(t *testing.T) {
	router, _ := setupTestApp()
	token := loginAdmin(router)
	w := doReq(router, http.MethodGet, "/api/m1/v1/me", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for K-01 GET /me, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	data := res.Data.(map[string]any)
	if data["email"] != "admin@araya.com" || data["role"] != "admin" {
		t.Fatalf("unexpected profile data: %+v", data)
	}
}

func TestM1_UserLifecycleAndDeactivation(t *testing.T) {
	router, _ := setupTestApp()
	token := loginAdmin(router)

	// 1. Create user
	w := doReq(router, http.MethodPost, "/api/m1/v1/users", token, dto.CreateUserRequest{
		Email: "staff@araya.com", Password: "StaffPassword123!", Name: "Staff Kasir", Role: "kasir",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Deactivate user
	wDeact := doReq(router, http.MethodDelete, "/api/m1/v1/users/2", token, nil)
	if wDeact.Code != http.StatusOK {
		t.Fatalf("expected 200 deactivated, got %d", wDeact.Code)
	}

	// 3. Deactivated user login rejected
	wLog := doReq(router, http.MethodPost, "/api/m1/v1/login", "", dto.LoginRequest{
		Email: "staff@araya.com", Password: "StaffPassword123!",
	})
	if wLog.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for deactivated user login, got %d", wLog.Code)
	}
}

func TestM1_ChangePasswordFlow(t *testing.T) {
	router, _ := setupTestApp()
	token := loginAdmin(router)

	wBad := doReq(router, http.MethodPatch, "/api/m1/v1/password", token, dto.ChangePasswordRequest{
		OldPassword: "WrongOldPassword", NewPassword: "NewSecretPassword123!",
	})
	if wBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for wrong old password, got %d", wBad.Code)
	}

	wOk := doReq(router, http.MethodPatch, "/api/m1/v1/password", token, dto.ChangePasswordRequest{
		OldPassword: "Admin123!", NewPassword: "NewSecretPassword123!",
	})
	if wOk.Code != http.StatusOK {
		t.Fatalf("expected 200 for successful password change, got %d", wOk.Code)
	}

	wLogin := doReq(router, http.MethodPost, "/api/m1/v1/login", "", dto.LoginRequest{
		Email: "admin@araya.com", Password: "NewSecretPassword123!",
	})
	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 for login with new password, got %d", wLogin.Code)
	}
}

func TestM1_AuditLogViewer(t *testing.T) {
	router, _ := setupTestApp()
	token := loginAdmin(router)

	w := doReq(router, http.MethodGet, "/api/m1/v1/audit-logs", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for audit log viewer, got %d", w.Code)
	}
	var res response.ListSuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if !res.Success || res.Meta.Total < 1 {
		t.Fatalf("expected audit logs with total >= 1, got %d", res.Meta.Total)
	}
}
