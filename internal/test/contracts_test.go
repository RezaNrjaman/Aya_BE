package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestContract_K01_GetMe_Success(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	w := DoReq(r, http.MethodGet, "/api/m1/v1/me", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID          int      `json:"id"`
			Email       string   `json:"email"`
			Role        string   `json:"role"`
			Permissions []string `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || !resp.Success || resp.Data.Email == "" {
		t.Fatalf("contract K-01 schema violation: %v", err)
	}
}

func TestContract_K01_GetMe_Fail_Unauthorized(t *testing.T) {
	r, _ := SetupFullTestApp()
	w := DoReq(r, http.MethodGet, "/api/m1/v1/me", "invalid-token", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}
	var errResp struct {
		Success bool `json:"success"`
		Error   any  `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil || errResp.Success {
		t.Fatalf("contract K-01 error envelope violation: %v", err)
	}
}

func TestContract_K02_BusinessUnits_Success(t *testing.T) {
	r, _ := SetupFullTestApp()
	w := DoReq(r, http.MethodGet, "/api/m5/v1/business-units", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	var resp struct {
		Success bool  `json:"success"`
		Data    []any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || !resp.Success {
		t.Fatalf("contract K-02 schema violation: %v", err)
	}
}

func TestContract_K03_DashboardSummary_Success(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	w := DoReq(r, http.MethodGet, "/api/m2/v1/dashboard-summary", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalOrders int64 `json:"total_orders"`
			Revenue     int64 `json:"revenue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || !resp.Success {
		t.Fatalf("contract K-03 schema violation: %v", err)
	}
}

func TestContract_K03_DashboardSummary_Fail_Unauthenticated(t *testing.T) {
	r, _ := SetupFullTestApp()
	w := DoReq(r, http.MethodGet, "/api/m2/v1/dashboard-summary", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestContract_K04_CreateOrder_Success(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	payload := map[string]any{
		"customer_id": 1,
		"items": []map[string]any{
			{"product_id": 1, "item_name": "Benih Padi", "quantity": 2, "unit_price": 50000},
		},
	}
	w := DoReq(r, http.MethodPost, "/api/m4/v1/orders", token, payload)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContract_K04_CreateOrder_Fail_EmptyItems(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	payload := map[string]any{"customer_id": 1, "items": []any{}}
	w := DoReq(r, http.MethodPost, "/api/m4/v1/orders", token, payload)
	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Fatalf("expected 422 or 400, got %d", w.Code)
	}
}

func TestContract_K05_Products_Success(t *testing.T) {
	r, _ := SetupFullTestApp()
	w := DoReq(r, http.MethodGet, "/api/m5/v1/products", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	var resp struct {
		Success bool  `json:"success"`
		Data    []any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || !resp.Success {
		t.Fatalf("contract K-05 schema violation: %v", err)
	}
}

func TestContract_K06_PartnerStatus_Success(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
		"company_name":   "Mitra Agrikultur Nusantara",
		"contact_person": "Budi Santoso",
		"email":          "mitra@nusantara.com",
		"phone":          "08123456789",
	})
	var regResp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &regResp)
	w := DoReq(r, http.MethodGet, fmt.Sprintf("/api/m3/v1/partners/%d", regResp.Data.ID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContract_K06_PartnerStatus_Fail_NotFound(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	w := DoReq(r, http.MethodGet, "/api/m3/v1/partners/99999", token, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}
