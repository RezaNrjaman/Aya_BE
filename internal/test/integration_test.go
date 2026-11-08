package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestIntegration_Login_To_PrivateEndpoints(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	for _, ep := range []string{"/api/m1/v1/users", "/api/m2/v1/admin/business-units", "/api/m3/v1/partners", "/api/m4/v1/admin/orders"} {
		w := DoReq(r, http.MethodGet, ep, token, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("access to %s failed with status %d", ep, w.Code)
		}
	}
}

func TestIntegration_M2ActiveUnit_To_M5Catalog(t *testing.T) {
	r, _ := SetupFullTestApp()
	w1 := DoReq(r, http.MethodGet, "/api/m5/v1/business-units", "", nil)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}
	var resp struct {
		Data []struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp)
	if len(resp.Data) == 0 {
		t.Fatalf("expected active business units in catalog")
	}
}

func TestIntegration_M3Partner_VerifyWorkflow(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
		"company_name": "PT Mitra Sawit Mandiri", "contact_person": "Joko", "email": "sawit@mandiri.com", "phone": "081299991",
	})
	var reg struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &reg)
	vW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/verify", reg.Data.ID), token, map[string]string{
		"action": "approve", "reason": "Dokumen NIB dan NPWP lengkap",
	})
	if vW.Code != http.StatusOK {
		t.Fatalf("verify failed: %s", vW.Body.String())
	}
}

func TestIntegration_M4Product_To_M5Catalog(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	createW := DoReq(r, http.MethodPost, "/api/m4/v1/products", token, map[string]any{
		"unit_usaha_id": 1, "name": "Beras Organik Pandanwangi", "category": "Pangan", "price": 85000, "stock": 50,
	})
	if createW.Code != http.StatusCreated {
		t.Fatalf("failed creating product in M4: %s", createW.Body.String())
	}
	catW := DoReq(r, http.MethodGet, "/api/m5/v1/products?search=Pandanwangi", "", nil)
	var catResp struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(catW.Body.Bytes(), &catResp)
	if len(catResp.Data) == 0 || catResp.Data[0].Name != "Beras Organik Pandanwangi" {
		t.Fatalf("product created in M4 did not appear in M5 public catalog")
	}
}

func TestIntegration_M5CartCheckout_To_M4Order(t *testing.T) {
	r, _ := SetupFullTestApp()
	sid := "test-session-int-003"
	hdr := map[string]string{"X-Session-ID": sid}
	addW := DoReqWithHeaders(r, http.MethodPost, "/api/m5/v1/cart/items", "", map[string]any{"product_id": 1, "quantity": 3}, hdr)
	if addW.Code != http.StatusOK {
		t.Fatalf("failed add to cart: %s", addW.Body.String())
	}
	coW := DoReqWithHeaders(r, http.MethodPost, "/api/m5/v1/cart/checkout", "", map[string]any{
		"customer_id": 1, "notes": "Pengiriman kilat", "idempotency_key": "int-checkout-key-1",
	}, hdr)
	if coW.Code != http.StatusCreated && coW.Code != http.StatusOK {
		t.Fatalf("failed checkout: %s", coW.Body.String())
	}
	var coResp struct {
		Data struct {
			OrderNo string `json:"order_no"`
		} `json:"data"`
	}
	_ = json.Unmarshal(coW.Body.Bytes(), &coResp)
	if coResp.Data.OrderNo == "" {
		t.Fatalf("order was not generated on checkout")
	}
}

func TestIntegration_M4PaymentSandbox_InvoiceTransition(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	invW := DoReq(r, http.MethodPost, "/api/m4/v1/invoices", token, map[string]any{
		"customer_id": 1,
		"items": []map[string]any{
			{"item_name": "Jasa Pemeliharaan Mesin", "quantity": 1, "unit_price": 500000},
		},
	})
	var invResp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(invW.Body.Bytes(), &invResp)
	payW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m4/v1/invoices/%d/pay", invResp.Data.ID), token, map[string]any{
		"method": "sandbox_qris", "amount": 500000,
	})
	if payW.Code != http.StatusOK {
		t.Fatalf("payment sandbox failed: %s", payW.Body.String())
	}
}
