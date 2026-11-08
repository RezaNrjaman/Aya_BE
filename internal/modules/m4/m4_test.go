package m4_test

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
	"araya-business-hub/be/internal/modules/m4"
	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/payment"
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

func setupTestM4App() (*gin.Engine, *m4.Module) {
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
	m3.InitModule(router, testSecret, policy, auditSvc)
	mod := m4.InitModule(router, testSecret, policy, auditSvc)
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

func TestM4_ProductReadContractK05_Public(t *testing.T) {
	router, _ := setupTestM4App()
	w := doReq(router, http.MethodGet, "/api/m4/v1/products?unit_usaha_id=1", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for Kontrak K-05, got %d: %s", w.Code, w.Body.String())
	}
	var res response.ListSuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if res.Meta.Total < 2 {
		t.Fatalf("expected at least 2 products for unit 1, got %d", res.Meta.Total)
	}
}

func TestM4_CreateProduct_SuccessAndValidation(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	reqInvalid := dto.CreateProductRequest{Name: "", Price: -100}
	wInv := doReq(router, http.MethodPost, "/api/m4/v1/products", token, reqInvalid)
	if wInv.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid product, got %d", wInv.Code)
	}

	reqValid := dto.CreateProductRequest{
		UnitUsahaID: 1, Name: "Map Raport K13 Emboss Foil", Category: "Percetakan", Price: 15000, Stock: 200,
	}
	wVal := doReq(router, http.MethodPost, "/api/m4/v1/products", token, reqValid)
	if wVal.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create product, got %d: %s", wVal.Code, wVal.Body.String())
	}
}

func TestM4_UpdateProduct_Success(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	newPrice := int64(85000)
	w := doReq(router, http.MethodPatch, "/api/m4/v1/products/1", token, dto.UpdateProductRequest{Price: &newPrice})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for update product, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["price"] != float64(85000) {
		t.Fatalf("expected price 85000, got %v", m["price"])
	}
}

func TestM4_Customer_CreateAndList(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	req := dto.CreateCustomerRequest{Name: "Dinas Pendidikan Kota Malang", Email: "disdik@malangkota.go.id"}
	wCreate := doReq(router, http.MethodPost, "/api/m4/v1/customers", token, req)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create customer, got %d", wCreate.Code)
	}

	wList := doReq(router, http.MethodGet, "/api/m4/v1/customers?page=1&per_page=10", token, nil)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 for list customer, got %d", wList.Code)
	}
}

func TestM4_Quotation_Lifecycle(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	req := dto.CreateQuotationRequest{
		UnitUsahaID: 1, CustomerID: 1,
		Items: []dto.QuotationItemInput{
			{ItemName: "Cetak Buku Panduan", Quantity: 10, UnitPrice: 50000},
		},
	}
	wCreate := doReq(router, http.MethodPost, "/api/m4/v1/quotations", token, req)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 for quotation, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	wApprove := doReq(router, http.MethodPost, "/api/m4/v1/quotations/1/approve", token, nil)
	if wApprove.Code != http.StatusOK {
		t.Fatalf("expected 200 for approve quotation, got %d", wApprove.Code)
	}
}

func TestM4_Invoice_FromApprovedQuotation(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	// 1. Create and approve quotation
	_ = doReq(router, http.MethodPost, "/api/m4/v1/quotations", token, dto.CreateQuotationRequest{
		UnitUsahaID: 1, CustomerID: 1,
		Items: []dto.QuotationItemInput{{ItemName: "Banner", Quantity: 2, UnitPrice: 75000}},
	})
	_ = doReq(router, http.MethodPost, "/api/m4/v1/quotations/1/approve", token, nil)

	// 2. Create invoice from approved quotation
	quotID := 1
	wInv := doReq(router, http.MethodPost, "/api/m4/v1/invoices", token, dto.CreateInvoiceRequest{
		QuotationID: &quotID, CustomerID: 1,
	})
	if wInv.Code != http.StatusCreated {
		t.Fatalf("expected 201 for invoice from approved quotation, got %d: %s", wInv.Code, wInv.Body.String())
	}
}

func TestM4_Invoice_FromDraftQuotationFails(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	// 1. Create quotation (status remains draft)
	_ = doReq(router, http.MethodPost, "/api/m4/v1/quotations", token, dto.CreateQuotationRequest{
		UnitUsahaID: 1, CustomerID: 1,
		Items: []dto.QuotationItemInput{{ItemName: "Banner Draft", Quantity: 1, UnitPrice: 75000}},
	})

	// 2. Creating invoice from unapproved quotation must fail with 422
	quotID := 1
	wInv := doReq(router, http.MethodPost, "/api/m4/v1/invoices", token, dto.CreateInvoiceRequest{
		QuotationID: &quotID, CustomerID: 1,
	})
	if wInv.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invoice from draft quotation, got %d: %s", wInv.Code, wInv.Body.String())
	}
}

func createTestInvoice(router *gin.Engine, token string, amount int64) int {
	w := doReq(router, http.MethodPost, "/api/m4/v1/invoices", token, dto.CreateInvoiceRequest{
		CustomerID: 1, Items: []dto.InvoiceItemInput{{ItemName: "Sewa", Quantity: 1, UnitPrice: amount}},
	})
	if w.Code != http.StatusCreated {
		return 0
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	return int(res.Data.(map[string]any)["id"].(float64))
}

func TestM4_Invoice_SandboxPaymentSuccess(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	invID := createTestInvoice(router, token, 250000)

	wPay := doReq(router, http.MethodPost, "/api/m4/v1/invoices/1/pay", token, dto.PayInvoiceRequest{
		Method: "sandbox_qris", Amount: 250000,
	})
	if wPay.Code != http.StatusOK {
		t.Fatalf("expected 200 for payment, got %d: %s", wPay.Code, wPay.Body.String())
	}
	wGet := doReq(router, http.MethodGet, "/api/m4/v1/invoices/1", token, nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(wGet.Body.Bytes(), &res)
	if res.Data.(map[string]any)["status"] != domain.InvoicePaid {
		t.Fatalf("expected invoice status lunas, got %v", res.Data)
	}
	_ = invID
}

func TestM4_Invoice_SandboxPaymentAmountMismatch(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	_ = createTestInvoice(router, token, 100000)

	wPay := doReq(router, http.MethodPost, "/api/m4/v1/invoices/1/pay", token, dto.PayInvoiceRequest{
		Method: "sandbox_va", Amount: 50000, // Amount mismatch
	})
	if wPay.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for payment amount mismatch, got %d", wPay.Code)
	}
}

func TestM4_Order_CreateContractK04(t *testing.T) {
	router, _ := setupTestM4App()
	req := dto.CreateOrderRequest{
		CustomerID: 1, IdempotencyKey: "check-123",
		Items: []dto.OrderItemInput{{ProductID: 1, Quantity: 2}}, // 75000 * 2 = 150000
	}
	w := doReq(router, http.MethodPost, "/api/m4/v1/orders", "", req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create order Kontrak K-04, got %d: %s", w.Code, w.Body.String())
	}
	var res response.SuccessResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	m := res.Data.(map[string]any)
	if m["total_amount"] != float64(150000) || m["status"] != domain.OrderNew {
		t.Fatalf("unexpected order payload: %+v", m)
	}
}

func TestM4_Order_IdempotencyContractK04(t *testing.T) {
	router, _ := setupTestM4App()
	req := dto.CreateOrderRequest{
		CustomerID: 1, IdempotencyKey: "idemp-same-key",
		Items: []dto.OrderItemInput{{ProductID: 1, Quantity: 1}},
	}
	w1 := doReq(router, http.MethodPost, "/api/m4/v1/orders", "", req)
	w2 := doReq(router, http.MethodPost, "/api/m4/v1/orders", "", req)
	var res1, res2 response.SuccessResponse
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	_ = json.Unmarshal(w2.Body.Bytes(), &res2)
	id1 := res1.Data.(map[string]any)["id"]
	id2 := res2.Data.(map[string]any)["id"]
	if id1 != id2 {
		t.Fatalf("idempotency failed: expected identical order ID, got %v and %v", id1, id2)
	}
}

func TestM4_Order_StateTransitionLifecycle(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	// 1. Create order
	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders", "", dto.CreateOrderRequest{
		CustomerID: 1, Items: []dto.OrderItemInput{{ProductID: 1, Quantity: 1}},
	})
	// 2. Process order
	wProc := doReq(router, http.MethodPost, "/api/m4/v1/orders/1/process", token, nil)
	if wProc.Code != http.StatusOK {
		t.Fatalf("expected 200 for process order, got %d", wProc.Code)
	}
	// 3. Complete order
	wComp := doReq(router, http.MethodPost, "/api/m4/v1/orders/1/complete", token, nil)
	if wComp.Code != http.StatusOK {
		t.Fatalf("expected 200 for complete order, got %d", wComp.Code)
	}
}

func TestM4_Order_CompleteOrderRejectsFurtherChanges(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders", "", dto.CreateOrderRequest{
		CustomerID: 1, Items: []dto.OrderItemInput{{ProductID: 1, Quantity: 1}},
	})
	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders/1/process", token, nil)
	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders/1/complete", token, nil)

	// Attempt to process again after complete must fail
	wReProc := doReq(router, http.MethodPost, "/api/m4/v1/orders/1/process", token, nil)
	if wReProc.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for modifying completed order, got %d", wReProc.Code)
	}
}

func TestM4_RBAC_UnauthorizedAndForbidden(t *testing.T) {
	router, _ := setupTestM4App()
	wUnauth := doReq(router, http.MethodGet, "/api/m4/v1/admin/orders", "", nil)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated orders list, got %d", wUnauth.Code)
	}
	custToken := getCustToken()
	wForbidden := doReq(router, http.MethodPost, "/api/m4/v1/products", custToken, dto.CreateProductRequest{Name: "X"})
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unauthorized create product, got %d", wForbidden.Code)
	}
}

func doCallbackReq(r *gin.Engine, secret string, body any, overrideSig string) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/m4/v1/payments/callback", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	sig := overrideSig
	if sig == "" {
		adapter := payment.NewSandboxAdapter(secret)
		sig = adapter.GenerateSignature(buf)
	}
	req.Header.Set("X-Callback-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestM4_Invoice_SandboxPaymentFailure(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	_ = createTestInvoice(router, token, 200000)

	payReq := dto.PayInvoiceRequest{
		Method: "sandbox_qris", Amount: 200000, SimulateFailure: true,
	}
	wPay := doReq(router, http.MethodPost, "/api/m4/v1/invoices/1/pay", token, payReq)
	if wPay.Code != http.StatusOK {
		t.Fatalf("expected 200 for simulated failure response, got %d", wPay.Code)
	}
	wInv := doReq(router, http.MethodGet, "/api/m4/v1/invoices/1", token, nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(wInv.Body.Bytes(), &res)
	status := res.Data.(map[string]any)["status"].(string)
	if status != domain.InvoiceUnpaid {
		t.Fatalf("expected invoice status to remain %s, got %s", domain.InvoiceUnpaid, status)
	}
}

func TestM4_Payment_CallbackSuccess(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	_ = createTestInvoice(router, token, 200000)

	cb := dto.PaymentCallbackRequest{
		PaymentReference: "PAY-REF-CB-001", InvoiceID: 1, Amount: 200000, Status: "SUCCESS",
	}
	w := doCallbackReq(router, "", cb, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid callback, got %d: %s", w.Code, w.Body.String())
	}
	wInv := doReq(router, http.MethodGet, "/api/m4/v1/invoices/1", token, nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(wInv.Body.Bytes(), &res)
	if res.Data.(map[string]any)["status"] != domain.InvoicePaid {
		t.Fatalf("expected invoice status lunas, got %v", res.Data.(map[string]any)["status"])
	}
}

func TestM4_Payment_CallbackFailure(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	_ = createTestInvoice(router, token, 200000)

	cb := dto.PaymentCallbackRequest{
		PaymentReference: "PAY-REF-CB-002", InvoiceID: 1, Amount: 200000, Status: "FAILED",
	}
	w := doCallbackReq(router, "", cb, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for failed callback, got %d", w.Code)
	}
	wInv := doReq(router, http.MethodGet, "/api/m4/v1/invoices/1", token, nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(wInv.Body.Bytes(), &res)
	if res.Data.(map[string]any)["status"] != domain.InvoiceUnpaid {
		t.Fatalf("expected invoice status to remain belum_lunas, got %v", res.Data.(map[string]any)["status"])
	}
}

func TestM4_Payment_CallbackInvalidSignature(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	_ = createTestInvoice(router, token, 200000)

	cb := dto.PaymentCallbackRequest{
		PaymentReference: "PAY-REF-CB-003", InvoiceID: 1, Amount: 200000, Status: "SUCCESS",
	}
	w := doCallbackReq(router, "", cb, "invalid-forged-signature-value")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid signature callback, got %d", w.Code)
	}
}

func TestM4_Payment_CallbackIdempotent(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()
	_ = createTestInvoice(router, token, 200000)

	cb := dto.PaymentCallbackRequest{
		PaymentReference: "PAY-REF-CB-004", InvoiceID: 1, Amount: 200000, Status: "SUCCESS",
	}
	w1 := doCallbackReq(router, "", cb, "")
	w2 := doCallbackReq(router, "", cb, "")
	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for both calls, got %d and %d", w1.Code, w2.Code)
	}
}

func TestM4_Order_CancelLifecycle(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders", "", dto.CreateOrderRequest{
		CustomerID: 1, Items: []dto.OrderItemInput{{ProductID: 1, Quantity: 1}},
	})
	wCancel := doReq(router, http.MethodPost, "/api/m4/v1/orders/1/cancel", token, map[string]string{
		"reason": "Customer membatalkan pesanan",
	})
	if wCancel.Code != http.StatusOK {
		t.Fatalf("expected 200 for cancel order, got %d", wCancel.Code)
	}
	wGet := doReq(router, http.MethodGet, "/api/m4/v1/orders/1", token, nil)
	var res response.SuccessResponse
	_ = json.Unmarshal(wGet.Body.Bytes(), &res)
	if res.Data.(map[string]any)["status"] != domain.OrderCancelled {
		t.Fatalf("expected status dibatalkan, got %v", res.Data.(map[string]any)["status"])
	}
}

func TestM4_Order_CancelCompletedOrderFails(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders", "", dto.CreateOrderRequest{
		CustomerID: 1, Items: []dto.OrderItemInput{{ProductID: 1, Quantity: 1}},
	})
	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders/1/process", token, nil)
	_ = doReq(router, http.MethodPost, "/api/m4/v1/orders/1/complete", token, nil)

	wCancel := doReq(router, http.MethodPost, "/api/m4/v1/orders/1/cancel", token, nil)
	if wCancel.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for cancelling completed order, got %d", wCancel.Code)
	}
}

func TestM4_Quotation_RejectLifecycle(t *testing.T) {
	router, _ := setupTestM4App()
	token := getAdminToken()

	_ = doReq(router, http.MethodPost, "/api/m4/v1/quotations", token, dto.CreateQuotationRequest{
		UnitUsahaID: 1, CustomerID: 1, Items: []dto.QuotationItemInput{{ItemName: "Cetak Banner", Quantity: 1, UnitPrice: 50000}},
	})
	wRej := doReq(router, http.MethodPost, "/api/m4/v1/quotations/1/reject", token, nil)
	if wRej.Code != http.StatusOK {
		t.Fatalf("expected 200 for reject quotation, got %d", wRej.Code)
	}
	qID := 1
	wInv := doReq(router, http.MethodPost, "/api/m4/v1/invoices", token, dto.CreateInvoiceRequest{
		CustomerID: 1, QuotationID: &qID,
	})
	if wInv.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invoice from rejected quotation, got %d", wInv.Code)
	}
}
