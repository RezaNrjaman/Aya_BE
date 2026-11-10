package m5_test

import (
	m1domain "araya-business-hub/be/internal/modules/m1/domain"
	m1repo "araya-business-hub/be/internal/modules/m1/repository"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/modules/m1"
	"araya-business-hub/be/internal/modules/m2"
	"araya-business-hub/be/internal/modules/m3"
	"araya-business-hub/be/internal/modules/m4"
	"araya-business-hub/be/internal/modules/m5"
	m5Client "araya-business-hub/be/internal/modules/m5/client"
	"araya-business-hub/be/internal/modules/m5/dto"
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

// Fixture accounts must exist in M1 because authorization resolves the user's
// current role and status on every request.
var (
	managerUserID  int
	customerUserID int
)

func seedTestUsers(repo m1repo.UserRepository) {
	mgr := &m1domain.User{Email: "manager@test.local", Name: "Manager Test", Role: "admin", IsActive: true}
	_ = repo.Create(context.Background(), mgr)
	cust := &m1domain.User{Email: "customer@test.local", Name: "Customer Test", Role: "customer", IsActive: true}
	_ = repo.Create(context.Background(), cust)
	managerUserID, customerUserID = mgr.ID, cust.ID
}

func setupTestM5App() (*gin.Engine, *m5.Module) {
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
	m1Mod := m1.InitModule(router, testSecret, 8, repo, policy, auditSvc)
	seedTestUsers(m1Mod.UserRepo)
	m2Mod := m2.InitModule(router, testSecret, policy, auditSvc)
	m3.InitModule(router, testSecret, policy, auditSvc)
	m4Mod := m4.InitModule(router, testSecret, policy, auditSvc, nil)

	m2Cli := m5Client.NewInternalM2Adapter(m2Mod.UnitSvc)
	m4Cli := m5Client.NewInternalM4Adapter(m4Mod.ProductSvc, m4Mod.OrderSvc)
	mod := m5.InitModule(router, testSecret, policy, auditSvc, m2Cli, m4Cli)
	return router, mod
}

func doReq(r *gin.Engine, method, path, token, sessionID string, body any) *httptest.ResponseRecorder {
	var buf []byte
	if body != nil {
		buf, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if sessionID != "" {
		req.Header.Set("X-Session-ID", sessionID)
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
		UserID: customerUserID, Name: "Customer Test", Role: "customer", Division: "publik",
	}, testSecret, 8)
	return tok
}

func TestM5_Catalog_BusinessUnits_ActiveOnly(t *testing.T) {
	router, _ := setupTestM5App()
	w := doReq(router, http.MethodGet, "/api/m5/v1/business-units", "", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for active business units, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items := res.Data.([]any)
	if len(items) == 0 {
		t.Fatal("expected at least 1 active business unit")
	}
}

func TestM5_Catalog_Products_ActiveOnly(t *testing.T) {
	router, _ := setupTestM5App()
	w := doReq(router, http.MethodGet, "/api/m5/v1/products?unit_usaha_id=1", "", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for active products, got %d", w.Code)
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items := res.Data.([]any)
	if len(items) == 0 {
		t.Fatal("expected at least 1 active product for unit 1")
	}
}

func TestM5_Catalog_Overview(t *testing.T) {
	router, _ := setupTestM5App()
	w := doReq(router, http.MethodGet, "/api/m5/v1/catalog", "", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog overview, got %d", w.Code)
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if _, ok := m["business_units"]; !ok {
		t.Fatal("missing business_units in catalog summary")
	}
	if _, ok := m["products"]; !ok {
		t.Fatal("missing products in catalog summary")
	}
}

func TestM5_Catalog_SearchAndFilter(t *testing.T) {
	router, _ := setupTestM5App()
	w := doReq(router, http.MethodGet, "/api/m5/v1/products?search=Banner", "", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for search products, got %d", w.Code)
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items := res.Data.([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 product for 'Banner', got %d", len(items))
	}
}

func TestM5_Catalog_NoResults_ReturnsEmptyListNotError(t *testing.T) {
	router, _ := setupTestM5App()
	w := doReq(router, http.MethodGet, "/api/m5/v1/products?search=NonExistentKeywordXYZ", "", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty search result, got %d", w.Code)
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	items := res.Data.([]any)
	if len(items) != 0 {
		t.Fatalf("expected empty list for non-existent keyword, got %d", len(items))
	}
}

func TestM5_ServiceRequest_SubmitSuccess(t *testing.T) {
	router, _ := setupTestM5App()
	reqBody := dto.CreateServiceRequest{
		UnitUsahaID:   1,
		CustomerName:  "Dewi Sartika",
		CustomerEmail: "dewi@example.com",
		CustomerPhone: "08123456789",
		ServiceType:   "Konveksi",
		Description:   "Pemesanan jaket almamater 50 pcs",
	}
	w := doReq(router, http.MethodPost, "/api/m5/v1/service-requests", "", "", reqBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid service request, got %d: %s", w.Code, w.Body.String())
	}
}

func TestM5_ServiceRequest_ValidationFails(t *testing.T) {
	router, _ := setupTestM5App()
	reqInvalid := dto.CreateServiceRequest{
		UnitUsahaID:   0,
		CustomerName:  "",
		CustomerEmail: "invalid-email-format",
		Description:   "",
	}
	w := doReq(router, http.MethodPost, "/api/m5/v1/service-requests", "", "", reqInvalid)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid service request, got %d", w.Code)
	}
}

func TestM5_ServiceRequest_StatusLifecycle(t *testing.T) {
	router, _ := setupTestM5App()
	adminToken := getAdminToken()

	reqBody := dto.CreateServiceRequest{
		UnitUsahaID: 1, CustomerName: "Ahmad", CustomerEmail: "ahmad@example.com", Description: "Permintaan cetak",
	}
	_ = doReq(router, http.MethodPost, "/api/m5/v1/service-requests", "", "", reqBody)

	// Status pending -> diproses
	wProc := doReq(router, http.MethodPatch, "/api/m5/v1/admin/service-requests/2/status", adminToken, "", dto.UpdateServiceRequestStatus{Status: "diproses"})
	if wProc.Code != http.StatusOK {
		t.Fatalf("expected 200 for status diproses, got %d: %s", wProc.Code, wProc.Body.String())
	}
	// Status diproses -> selesai
	wComp := doReq(router, http.MethodPatch, "/api/m5/v1/admin/service-requests/2/status", adminToken, "", dto.UpdateServiceRequestStatus{Status: "selesai"})
	if wComp.Code != http.StatusOK {
		t.Fatalf("expected 200 for status selesai, got %d: %s", wComp.Code, wComp.Body.String())
	}
}

func assertCartItemCountAndQty(t *testing.T, body []byte, expectedItems, expectedQty int) {
	var res response.SuccessResponse
	_ = json.Unmarshal(body, &res)
	cart := res.Data.(map[string]any)
	items := cart["items"].([]any)
	if len(items) != expectedItems {
		t.Fatalf("expected %d unique items, got %d", expectedItems, len(items))
	}
	first := items[0].(map[string]any)
	if int(first["quantity"].(float64)) != expectedQty {
		t.Fatalf("expected quantity %d, got %v", expectedQty, first["quantity"])
	}
}

func TestM5_Cart_AddItem_NewAndExistingQuantityIncrement(t *testing.T) {
	router, _ := setupTestM5App()
	sessionID := "test-session-cart-1"

	w1 := doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{
		ProductID: 1, Quantity: 2,
	})
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 for add to cart, got %d", w1.Code)
	}
	w2 := doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{
		ProductID: 1, Quantity: 3,
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for adding same item, got %d", w2.Code)
	}
	assertCartItemCountAndQty(t, w2.Body.Bytes(), 1, 5)
}

func TestM5_Cart_UpdateQuantity(t *testing.T) {
	router, _ := setupTestM5App()
	sessionID := "test-session-cart-2"

	_ = doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{ProductID: 1, Quantity: 1})
	wUp := doReq(router, http.MethodPatch, "/api/m5/v1/cart/items/1", "", sessionID, dto.UpdateCartItemRequest{Quantity: 4})
	if wUp.Code != http.StatusOK {
		t.Fatalf("expected 200 for update quantity, got %d", wUp.Code)
	}
	// Invalid quantity <= 0
	wInv := doReq(router, http.MethodPatch, "/api/m5/v1/cart/items/1", "", sessionID, dto.UpdateCartItemRequest{Quantity: 0})
	if wInv.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid quantity, got %d", wInv.Code)
	}
}

func TestM5_Cart_RemoveItemAndClear(t *testing.T) {
	router, _ := setupTestM5App()
	sessionID := "test-session-cart-3"

	_ = doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{ProductID: 1, Quantity: 1})
	_ = doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{ProductID: 2, Quantity: 1})

	wDel := doReq(router, http.MethodDelete, "/api/m5/v1/cart/items/1", "", sessionID, nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 for remove item, got %d", wDel.Code)
	}
	wClear := doReq(router, http.MethodDelete, "/api/m5/v1/cart", "", sessionID, nil)
	if wClear.Code != http.StatusOK {
		t.Fatalf("expected 200 for clear cart, got %d", wClear.Code)
	}
}

func TestM5_Checkout_EmptyCartFails(t *testing.T) {
	router, _ := setupTestM5App()
	sessionID := "test-session-empty"

	w := doReq(router, http.MethodPost, "/api/m5/v1/cart/checkout", "", sessionID, dto.CheckoutRequest{
		CustomerID: 1,
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for checkout empty cart, got %d", w.Code)
	}
}

func TestM5_Checkout_SuccessCreatesOrderAndClearsCart(t *testing.T) {
	router, _ := setupTestM5App()
	sessionID := "test-session-checkout"

	_ = doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{ProductID: 1, Quantity: 2})
	chkReq := dto.CheckoutRequest{
		CustomerID:     1,
		IdempotencyKey: "chk-session-20261109-001",
		Notes:          "Kirim ke fakultas teknik",
	}
	wChk := doReq(router, http.MethodPost, "/api/m5/v1/cart/checkout", "", sessionID, chkReq)
	if wChk.Code != http.StatusCreated {
		t.Fatalf("expected 201 for successful checkout, got %d: %s", wChk.Code, wChk.Body.String())
	}
	// Verify cart is cleared
	wCart := doReq(router, http.MethodGet, "/api/m5/v1/cart", "", sessionID, nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(wCart.Body.Bytes(), &res)
	cart := res.Data.(map[string]any)
	if int(cart["total_items"].(float64)) != 0 {
		t.Fatalf("expected cart total items 0 after checkout, got %v", cart["total_items"])
	}
}

func TestM5_Checkout_Idempotency(t *testing.T) {
	router, _ := setupTestM5App()
	sessionID := "test-session-idemp"
	key := "chk-idemp-unique-12345"

	_ = doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{ProductID: 1, Quantity: 1})
	w1 := doReq(router, http.MethodPost, "/api/m5/v1/cart/checkout", "", sessionID, dto.CheckoutRequest{CustomerID: 1, IdempotencyKey: key})
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for first checkout, got %d", w1.Code)
	}
	// Second checkout with same key
	_ = doReq(router, http.MethodPost, "/api/m5/v1/cart/items", "", sessionID, dto.AddToCartRequest{ProductID: 1, Quantity: 1})
	w2 := doReq(router, http.MethodPost, "/api/m5/v1/cart/checkout", "", sessionID, dto.CheckoutRequest{CustomerID: 1, IdempotencyKey: key})
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for second checkout with same key, got %d", w2.Code)
	}
	var res1, res2 response.SuccessResponse
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	_ = json.Unmarshal(w2.Body.Bytes(), &res2)
	id1 := res1.Data.(map[string]any)["order_id"]
	id2 := res2.Data.(map[string]any)["order_id"]
	if id1 != id2 {
		t.Fatalf("expected same order ID for idempotency key, got %v vs %v", id1, id2)
	}
}

func TestM5_AdminRoutes_RBAC(t *testing.T) {
	router, _ := setupTestM5App()
	// Unauthenticated
	wUnauth := doReq(router, http.MethodGet, "/api/m5/v1/admin/service-requests", "", "", nil)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated access, got %d", wUnauth.Code)
	}
	// Forbidden
	custToken := getCustToken()
	wForbid := doReq(router, http.MethodGet, "/api/m5/v1/admin/service-requests", custToken, "", nil)
	if wForbid.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for customer accessing admin requests, got %d", wForbid.Code)
	}
}
