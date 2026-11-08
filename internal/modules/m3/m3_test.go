package m3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/modules/m1"
	"araya-business-hub/be/internal/modules/m2"
	"araya-business-hub/be/internal/modules/m3"
	"araya-business-hub/be/internal/modules/m3/domain"
	"araya-business-hub/be/internal/modules/m3/dto"
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

func setupTestM3App() (*gin.Engine, *m3.Module) {
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
	m2.InitModule(router, testSecret, policy, auditSvc)
	mod := m3.InitModule(router, testSecret, policy, auditSvc)
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

func getCustToken() string {
	tok, _ := security.GenerateToken(security.PlatformClaims{
		UserID: 99, Name: "Customer Test", Role: "customer", Division: "publik",
	}, testSecret, 8)
	return tok
}

func TestM3_RegisterPartner_Success(t *testing.T) {
	router, _ := setupTestM3App()
	req := dto.RegisterPartnerRequest{
		CompanyName: "PT Sukses Berdikari", ContactPerson: "Agus Pratama",
		Email: "agus@suksesberdikari.com", Phone: "081234567890", Address: "Kawasan Industri Blok B",
	}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/register", "", req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for register partner, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["status"] != domain.StatusPending {
		t.Fatalf("expected initial status pending, got %v", m["status"])
	}
}

func TestM3_RegisterPartner_MissingFields(t *testing.T) {
	router, _ := setupTestM3App()
	req := dto.RegisterPartnerRequest{
		CompanyName: "", ContactPerson: "Agus", Email: "agus@test.com", Phone: "",
	}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/register", "", req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for incomplete form, got %d", w.Code)
	}
}

func TestM3_RegisterPartner_DuplicateEmail(t *testing.T) {
	router, _ := setupTestM3App()
	req := dto.RegisterPartnerRequest{
		CompanyName: "PT Sukses Duplikat", ContactPerson: "Budi",
		Email: "budi@mitragrafika.com", Phone: "081234567890",
	}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/register", "", req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict for duplicate email, got %d: %s", w.Code, w.Body.String())
	}
}

func TestM3_VerifyPartner_ApproveSuccess(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()

	req := dto.VerifyPartnerRequest{Action: "approve"}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/3/verify", token, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for partner approval, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["status"] != domain.StatusVerified || m["verified_by"] != "Admin Test" {
		t.Fatalf("unexpected approval payload: %+v", m)
	}
}

func TestM3_VerifyPartner_RejectSuccess(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()

	req := dto.VerifyPartnerRequest{Action: "reject", Reason: "Dokumen legalitas tidak lengkap"}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/3/verify", token, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for partner rejection, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["status"] != domain.StatusRejected || m["rejection_reason"] != req.Reason {
		t.Fatalf("unexpected rejection payload: %+v", m)
	}
}

func TestM3_VerifyPartner_RejectMissingReason(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	req := dto.VerifyPartnerRequest{Action: "reject", Reason: "  "}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/3/verify", token, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for reject without reason, got %d", w.Code)
	}
}

func TestM3_VerifyPartner_InvalidTransition(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	// Partner 1 is already verified
	req := dto.VerifyPartnerRequest{Action: "approve"}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/1/verify", token, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid transition on verified partner, got %d", w.Code)
	}
}

func TestM3_SuspendPartner_Success(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	req := dto.SuspendPartnerRequest{Reason: "Pelanggaran SLA pengiriman berkali-kali"}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/1/suspend", token, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for suspend partner, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["status"] != domain.StatusSuspended || m["suspension_reason"] != req.Reason {
		t.Fatalf("unexpected suspension payload: %+v", m)
	}
}

func TestM3_SuspendPartner_MissingReason(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	req := dto.SuspendPartnerRequest{Reason: "  "}
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/1/suspend", token, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for suspend without reason, got %d", w.Code)
	}
}

func TestM3_ReactivatePartner_Success(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	// 1. Suspend partner 1 first
	_ = doReq(router, http.MethodPost, "/api/m3/v1/partners/1/suspend", token, dto.SuspendPartnerRequest{Reason: "Hold"})
	// 2. Reactivate partner 1
	w := doReq(router, http.MethodPost, "/api/m3/v1/partners/1/reactivate", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for reactivate partner, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["status"] != domain.StatusVerified || (m["suspension_reason"] != nil && m["suspension_reason"] != "") {
		t.Fatalf("unexpected reactivate payload: %+v", m)
	}
}

func TestM3_ListVerifiedPartners_K06(t *testing.T) {
	router, _ := setupTestM3App()
	w := doReq(router, http.MethodGet, "/api/m3/v1/partners/verified", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for verified partners, got %d", w.Code)
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items := res.Data.([]any)
	if len(items) != 2 {
		t.Fatalf("expected exactly 2 verified partners, got %d", len(items))
	}
}

func TestM3_GetPartnerDetail(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	w := doReq(router, http.MethodGet, "/api/m3/v1/partners/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for partner detail, got %d", w.Code)
	}
	wNotFound := doReq(router, http.MethodGet, "/api/m3/v1/partners/999", token, nil)
	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent partner, got %d", wNotFound.Code)
	}
}

func TestM3_ListPartners_AdminPagination(t *testing.T) {
	router, _ := setupTestM3App()
	token := getAdminToken()
	w := doReq(router, http.MethodGet, "/api/m3/v1/partners?status=pending&page=1&per_page=10", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for partner list, got %d", w.Code)
	}
	var res response.ListSuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Meta.Total < 1 {
		t.Fatalf("expected at least 1 pending partner, got %d", res.Meta.Total)
	}
}

func TestM3_RBAC_UnauthorizedAndForbidden(t *testing.T) {
	router, _ := setupTestM3App()
	wUnauth := doReq(router, http.MethodGet, "/api/m3/v1/partners", "", nil)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated list, got %d", wUnauth.Code)
	}
	custToken := getCustToken()
	wForbidden := doReq(router, http.MethodPost, "/api/m3/v1/partners/1/suspend", custToken, dto.SuspendPartnerRequest{Reason: "Test"})
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for insufficient permission, got %d", wForbidden.Code)
	}
}
