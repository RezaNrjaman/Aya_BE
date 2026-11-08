package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type orderResult struct {
	Data struct {
		ID      int    `json:"id"`
		OrderNo string `json:"order_no"`
	} `json:"data"`
}

type paymentResult struct {
	Data struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	} `json:"data"`
}

type idResult struct {
	Data struct {
		ID int `json:"id"`
	} `json:"data"`
}

func parseOrderResult(w *httptest.ResponseRecorder) orderResult {
	var r orderResult
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return r
}

func parsePaymentResult(w *httptest.ResponseRecorder) paymentResult {
	var r paymentResult
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return r
}

func parseIDResult(w *httptest.ResponseRecorder) idResult {
	var r idResult
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return r
}

func newOrderPayload(key string) map[string]any {
	return map[string]any{
		"customer_id":     1,
		"idempotency_key": key,
		"items": []map[string]any{
			{"product_id": 1, "item_name": "Bibit Kelapa Sawit", "quantity": 10, "unit_price": 50000},
		},
	}
}

func newInvoicePayload() map[string]any {
	return map[string]any{
		"customer_id": 1,
		"items": []map[string]any{
			{"item_name": "Konsultasi IT", "quantity": 1, "unit_price": 300000},
		},
	}
}

func TestIdempotency_OrderCreation_SameOrderReturned(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	payload := newOrderPayload("unique-idemp-key-order-12345")

	res1 := parseOrderResult(DoReq(r, http.MethodPost, "/api/m4/v1/orders", token, payload))
	res2 := parseOrderResult(DoReq(r, http.MethodPost, "/api/m4/v1/orders", token, payload))

	if res1.Data.ID == 0 || res1.Data.ID != res2.Data.ID || res1.Data.OrderNo != res2.Data.OrderNo {
		t.Fatalf("idempotency violation: %v vs %v", res1, res2)
	}
}

func createInvoiceAndPay(r *gin.Engine, token string) (*httptest.ResponseRecorder, *httptest.ResponseRecorder) {
	invW := DoReq(r, http.MethodPost, "/api/m4/v1/invoices", token, newInvoicePayload())
	inv := parseIDResult(invW)
	payPayload := map[string]any{"method": "sandbox_qris", "amount": 300000}
	payURL := fmt.Sprintf("/api/m4/v1/invoices/%d/pay", inv.Data.ID)
	return DoReq(r, http.MethodPost, payURL, token, payPayload),
		DoReq(r, http.MethodPost, payURL, token, payPayload)
}

func TestIdempotency_InvoiceDoublePayment_Idempotent(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	pay1, pay2 := createInvoiceAndPay(r, token)

	if pay1.Code != http.StatusOK || pay2.Code != http.StatusOK {
		t.Fatalf("expected both 200 OK")
	}
	p1 := parsePaymentResult(pay1)
	p2 := parsePaymentResult(pay2)
	if p1.Data.ID != p2.Data.ID || p1.Data.Status != "success" {
		t.Fatalf("idempotency violation: %v vs %v", p1, p2)
	}
}

func createAndCancelOrder(r *gin.Engine, token string) int {
	ordW := DoReq(r, http.MethodPost, "/api/m4/v1/orders", token, map[string]any{
		"customer_id": 1,
		"items": []map[string]any{
			{"product_id": 1, "item_name": "Pupuk NPK", "quantity": 1, "unit_price": 50000},
		},
	})
	ord := parseIDResult(ordW)
	DoReq(r, http.MethodPost, fmt.Sprintf("/api/m4/v1/orders/%d/cancel", ord.Data.ID), token, nil)
	return ord.Data.ID
}

func TestIntegrity_OrderInvalidStateTransition_Rejected(t *testing.T) {
	r, _ := SetupFullTestApp()
	token := LoginAdmin(r)
	ordID := createAndCancelOrder(r, token)
	procW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m4/v1/orders/%d/process", ordID), token, nil)
	if procW.Code == http.StatusOK {
		t.Fatalf("illegal transition: cancelled → processed")
	}
}
