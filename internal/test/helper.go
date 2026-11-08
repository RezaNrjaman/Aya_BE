package test

import (
	"bytes"
	"context"
	"encoding/json"
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
	m1.InitModule(router, TestJWTSecret, 8, repo, policy, auditSvc)
	m2Mod := m2.InitModule(router, TestJWTSecret, policy, auditSvc)
	m3.InitModule(router, TestJWTSecret, policy, auditSvc)
	m4Mod := m4.InitModule(router, TestJWTSecret, policy, auditSvc)
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
		Email: "admin@araya.com", Password: "Admin1234!",
	})
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data.Token
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
