package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"araya-business-hub/be/internal/modules/m1"
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m2"
	"araya-business-hub/be/internal/modules/m3"
	"araya-business-hub/be/internal/modules/m4"
	"araya-business-hub/be/internal/modules/m5"
	m5Client "araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/config"
	platformHttp "araya-business-hub/be/internal/platform/http"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/security"
	"github.com/gin-gonic/gin"
)

const TestJWTSecret = "test_jwt_secret_key_minimum_32_characters"

// testM1 exposes the M1 module of the latest SetupFullTestApp so tests can
// create real fixture accounts (authorization resolves users live).
var testM1 *m1.Module

func init() {
	gin.SetMode(gin.TestMode)
}

func initTestRBAC() (*rbac.Registry, rbac.Repository, *rbac.PolicyService) {
	reg := rbac.NewRegistry()
	_ = rbac.SeedPlatformPermissions(reg)
	repo := rbac.NewInMemoryRepository()
	_ = rbac.SeedInitialRoles(context.Background(), repo)
	return reg, repo, rbac.NewPolicyService(reg, repo)
}

func SetupFullTestApp() (*gin.Engine, *rbac.PolicyService) {
	reg, repo, policy := initTestRBAC()
	auditSvc := audit.NewService(audit.NewInMemoryRepository())
	cfg := &config.Config{Port: "8080", AppEnv: "test", JWTSecret: TestJWTSecret, JWTExpirationHours: 8}
	router := platformHttp.NewRouter(platformHttp.RouterDependencies{
		Config: cfg, RBACRegistry: reg, RBACRepo: repo, PolicyService: policy, AuditService: auditSvc,
	})
	m1Mod := m1.InitModule(router, TestJWTSecret, 8, repo, policy, auditSvc)
	testM1 = m1Mod
	m2Mod := m2.InitModule(router, TestJWTSecret, policy, auditSvc)
	m3Mod := m3.InitModule(router, TestJWTSecret, policy, auditSvc, m1Mod.Provisioner)
	// ARY-004: m3's partner status gates the public storefront in m4.
	m4Mod := m4.InitModule(router, TestJWTSecret, policy, auditSvc, m3Mod.Svc)
	m2Cli := m5Client.NewInternalM2Adapter(m2Mod.UnitSvc)
	m4Cli := m5Client.NewInternalM4Adapter(m4Mod.ProductSvc, m4Mod.OrderSvc)
	m5.InitModule(router, TestJWTSecret, policy, auditSvc, m2Cli, m4Cli)
	return router, policy
}

func DoReq(r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	return DoReqWithHeaders(r, method, path, token, body, nil)
}

func DoReqWithHeaders(r *gin.Engine, method, path, token string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func LoginAdmin(r *gin.Engine) string {
	w := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", dto.LoginRequest{
		Email: "admin@araya.com", Password: "Admin123!",
	})
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data.Token
}

// OnboardVerifiedPartner runs the full ARY-002 onboarding flow: public
// registration, admin approval, single-use activation and owner login.
// It returns the partner ID and the owner's session token.
func OnboardVerifiedPartner(t interface {
	Helper()
	Fatalf(string, ...any)
}, r *gin.Engine, adminToken, email, companyName, password string) (int, string) {
	t.Helper()
	regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
		"company_name": companyName, "contact_person": "PIC", "email": email, "phone": "0812000000",
	})
	if regW.Code != http.StatusCreated {
		t.Fatalf("register %s failed: %s", email, regW.Body.String())
	}
	var reg struct {
		Data struct {
			ID               int    `json:"id"`
			RegistrationCode string `json:"registration_code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &reg)
	if vW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/verify", reg.Data.ID), adminToken, map[string]string{
		"action": "approve", "reason": "ok",
	}); vW.Code != http.StatusOK {
		t.Fatalf("verify %s failed: %s", email, vW.Body.String())
	}
	var st struct {
		Data struct {
			ActivationToken string `json:"activation_token"`
		} `json:"data"`
	}
	stW := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code="+reg.Data.RegistrationCode, "", nil)
	_ = json.Unmarshal(stW.Body.Bytes(), &st)
	if st.Data.ActivationToken == "" {
		t.Fatalf("no activation token for %s", email)
	}
	if aW := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": st.Data.ActivationToken, "password": password,
	}); aW.Code != http.StatusOK {
		t.Fatalf("activate %s failed: %s", email, aW.Body.String())
	}
	lW := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": email, "password": password,
	})
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(lW.Body.Bytes(), &login)
	if login.Data.Token == "" {
		t.Fatalf("login %s failed: %s", email, lW.Body.String())
	}
	return reg.Data.ID, login.Data.Token
}

func GenerateToken(userID int, email, role string) string {
	claims := security.PlatformClaims{
		UserID: userID,
		Name:   email,
		Role:   role,
	}
	tok, _ := security.GenerateToken(claims, TestJWTSecret, 8)
	return tok
}
