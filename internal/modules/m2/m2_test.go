package m2_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/modules/m1"
	"araya-business-hub/be/internal/modules/m2"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/config"
	platformHttp "araya-business-hub/be/internal/platform/http"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/response"
	"araya-business-hub/be/internal/platform/security"
	"github.com/gin-gonic/gin"
)

const testSecret = "test_jwt_secret_key_minimum_32_characters"

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestM2App() (*gin.Engine, *m2.Module) {
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
	m1.InitModule(router, testSecret, 8, repo, policy, auditSvc)
	mod := m2.InitModule(router, testSecret, policy, auditSvc)
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

func getAdminToken() string {
	tok, _ := security.GenerateToken(security.PlatformClaims{
		UserID: 1, Name: "Admin Test", Role: "admin", Division: "operasional",
	}, testSecret, 8)
	return tok
}

func getScopedToken(unitID int) string {
	tok, _ := security.GenerateToken(security.PlatformClaims{
		UserID: 2, Name: "Manager Test", Role: "admin", Division: "operasional", UnitUsahaID: &unitID,
	}, testSecret, 8)
	return tok
}

func getNoPermToken() string {
	tok, _ := security.GenerateToken(security.PlatformClaims{
		UserID: 3, Name: "Customer Test", Role: "customer", Division: "publik",
	}, testSecret, 8)
	return tok
}

func TestM2_ActiveUnitsContractK02_Public(t *testing.T) {
	router, _ := setupTestM2App()
	w := doReq(router, http.MethodGet, "/api/m2/v1/business-units", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for Kontrak K-02, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items, ok := res.Data.([]any)
	if !ok || len(items) < 3 {
		t.Fatalf("expected at least 3 seeded active business units, got %v", res.Data)
	}
}

func TestM2_CreateUnit_Validation(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	w := doReq(router, http.MethodPost, "/api/m2/v1/business-units", token, dto.CreateUnitRequest{
		Name: "   ",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for empty name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestM2_CreateUnit_Success(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	w := doReq(router, http.MethodPost, "/api/m2/v1/business-units", token, dto.CreateUnitRequest{
		Name: "Unit Agribisnis & Pertanian", CategoryName: "Pertanian", Description: "Produk hasil tani BPU",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w.Code, w.Body.String())
	}
}

func TestM2_UpdateUnit_Success(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	newName := "Unit Percetakan Digital"
	w := doReq(router, http.MethodPatch, "/api/m2/v1/business-units/1", token, dto.UpdateUnitRequest{
		Name: &newName,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 updated, got %d: %s", w.Code, w.Body.String())
	}
}

func TestM2_GetUnitByID(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	wFound := doReq(router, http.MethodGet, "/api/m2/v1/business-units/1", token, nil)
	if wFound.Code != http.StatusOK {
		t.Fatalf("expected 200 for unit 1, got %d", wFound.Code)
	}
	wNotFound := doReq(router, http.MethodGet, "/api/m2/v1/business-units/999", token, nil)
	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent unit, got %d", wNotFound.Code)
	}
}

func TestM2_ListAdminUnits(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	w := doReq(router, http.MethodGet, "/api/m2/v1/admin/business-units?page=1&per_page=10", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin unit list, got %d", w.Code)
	}
	var res response.ListSuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Meta.Total < 3 {
		t.Fatalf("expected total at least 3, got %v", res.Meta)
	}
}

func assertUnitInK02(t *testing.T, router *gin.Engine, unitID int, shouldExist bool) {
	w := doReq(router, http.MethodGet, "/api/m2/v1/business-units", "", nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items := res.Data.([]any)
	found := false
	for _, it := range items {
		if it.(map[string]any)["id"] == float64(unitID) {
			found = true
			break
		}
	}
	if found != shouldExist {
		t.Fatalf("expected unit %d presence in K-02 to be %v, got %v", unitID, shouldExist, found)
	}
}

func TestM2_DeactivateAndReactivateLifecycle(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()

	wDeact := doReq(router, http.MethodPost, "/api/m2/v1/business-units/1/deactivate", token, nil)
	if wDeact.Code != http.StatusOK {
		t.Fatalf("expected 200 deactivated, got %d", wDeact.Code)
	}
	assertUnitInK02(t, router, 1, false)

	wAct := doReq(router, http.MethodPost, "/api/m2/v1/business-units/1/activate", token, nil)
	if wAct.Code != http.StatusOK {
		t.Fatalf("expected 200 reactivated, got %d", wAct.Code)
	}
	assertUnitInK02(t, router, 1, true)
}

func TestM2_DashboardSummaryContractK03(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	w := doReq(router, http.MethodGet, "/api/m2/v1/dashboard-summary?from=2026-10-01&to=2026-10-31&unit_usaha_id=1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for Kontrak K-03, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	data := res.Data.(map[string]any)
	if data["period_from"] != "2026-10-01" || data["unit_usaha_id"] != float64(1) {
		t.Fatalf("unexpected dashboard summary filter mapping: %+v", data)
	}
}

func TestM2_DashboardSummary_InvalidDateRange(t *testing.T) {
	router, _ := setupTestM2App()
	token := getAdminToken()
	w := doReq(router, http.MethodGet, "/api/m2/v1/dashboard-summary?from=2026-11-10&to=2026-11-01", token, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid date range, got %d", w.Code)
	}
}

func TestM2_DashboardSummary_ScopeEnforcement(t *testing.T) {
	router, _ := setupTestM2App()
	mgrToken := getScopedToken(1)

	// Authorized scoped access
	wAllowed := doReq(router, http.MethodGet, "/api/m2/v1/dashboard-summary?unit_usaha_id=1", mgrToken, nil)
	if wAllowed.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed scope, got %d", wAllowed.Code)
	}

	// Forbidden out-of-scope access
	wForbidden := doReq(router, http.MethodGet, "/api/m2/v1/dashboard-summary?unit_usaha_id=2", mgrToken, nil)
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for forbidden unit scope, got %d", wForbidden.Code)
	}
}

func TestM2_RBAC_UnauthorizedAndForbidden(t *testing.T) {
	router, _ := setupTestM2App()
	wUnauth := doReq(router, http.MethodGet, "/api/m2/v1/dashboard-summary", "", nil)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", wUnauth.Code)
	}
	custToken := getNoPermToken()
	wForbidden := doReq(router, http.MethodGet, "/api/m2/v1/dashboard-summary", custToken, nil)
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for insufficient permission, got %d", wForbidden.Code)
	}
}
