package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ARY-004: suspending a partner must immediately remove its products from the
// public storefront (catalog, cart and checkout). Reactivation restores them,
// but an account that was disabled manually stays disabled.
func TestSecurity_ARY004_SuspendedPartnerHiddenFromStorefront(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	const password = "KataSandiKuat#2026"

	// --- onboard a verified partner with an activated account ---
	regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
		"company_name": "PT Storefront Mitra", "contact_person": "PIC Mitra",
		"email": "storefront@mitra.com", "phone": "081234567890",
	})
	if regW.Code != http.StatusCreated {
		t.Fatalf("register failed: %s", regW.Body.String())
	}
	var reg struct {
		Data struct {
			ID               int    `json:"id"`
			RegistrationCode string `json:"registration_code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &reg)
	if vW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/verify", reg.Data.ID), admin, map[string]string{
		"action": "approve", "reason": "ok",
	}); vW.Code != http.StatusOK {
		t.Fatalf("verify failed: %s", vW.Body.String())
	}
	var st struct {
		Data struct {
			ActivationToken string `json:"activation_token"`
		} `json:"data"`
	}
	stW := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code="+reg.Data.RegistrationCode, "", nil)
	_ = json.Unmarshal(stW.Body.Bytes(), &st)
	if aW := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": st.Data.ActivationToken, "password": password,
	}); aW.Code != http.StatusOK {
		t.Fatalf("activate failed: %s", aW.Body.String())
	}
	lW := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": "storefront@mitra.com", "password": password,
	})
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(lW.Body.Bytes(), &login)
	if login.Data.Token == "" {
		t.Fatalf("partner login failed: %s", lW.Body.String())
	}
	partnerTok := login.Data.Token
	partnerID := reg.Data.ID

	// --- the partner publishes a product ---
	createW := DoReq(r, http.MethodPost, "/api/m4/v1/products", partnerTok, map[string]any{
		"unit_usaha_id": 1, "name": "Produk Mitra Storefront", "category": "Percetakan",
		"price": 15000, "stock": 20,
	})
	if createW.Code != http.StatusCreated {
		t.Fatalf("create product failed: %s", createW.Body.String())
	}
	var prod struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(createW.Body.Bytes(), &prod)
	productID := prod.Data.ID

	catalogContains := func(id int) bool {
		w := DoReq(r, http.MethodGet, "/api/m5/v1/products", "", nil)
		var list struct {
			Data []struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		for _, p := range list.Data {
			if p.ID == id {
				return true
			}
		}
		return false
	}
	if !catalogContains(productID) {
		t.Fatalf("verified partner's product must be listed in the public catalog")
	}

	// A shopper already holds the product in an open cart.
	const sess = "session-ary004"
	if w := DoReqWithHeaders(r, http.MethodPost, "/api/m5/v1/cart/items", "", map[string]any{
		"product_id": productID, "quantity": 2,
	}, map[string]string{"X-Session-ID": sess}); w.Code != http.StatusOK {
		t.Fatalf("add to cart failed: %s", w.Body.String())
	}

	// --- suspend the partner ---
	if w := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/suspend", partnerID), admin, map[string]string{
		"reason": "Pelanggaran SLA",
	}); w.Code != http.StatusOK {
		t.Fatalf("suspend failed: %s", w.Body.String())
	}

	// The product disappears from the catalog...
	if catalogContains(productID) {
		t.Fatalf("suspended partner's product must not appear in the public catalog")
	}
	// ...new cart additions are rejected...
	if w := DoReqWithHeaders(r, http.MethodPost, "/api/m5/v1/cart/items", "", map[string]any{
		"product_id": productID, "quantity": 1,
	}, map[string]string{"X-Session-ID": "session-ary004-b"}); w.Code == http.StatusOK {
		t.Fatalf("adding a suspended partner's product to the cart must be rejected")
	}
	// ...and checkout of an existing cart is rejected.
	if w := DoReqWithHeaders(r, http.MethodPost, "/api/m5/v1/cart/checkout", "", map[string]any{
		"customer_id": 1,
	}, map[string]string{"X-Session-ID": sess}); w.Code == http.StatusOK {
		t.Fatalf("checkout of a suspended partner's product must be rejected")
	}

	// Login is also blocked while suspended (account disabled by suspension).
	lW = DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": "storefront@mitra.com", "password": password,
	})
	if lW.Code == http.StatusOK {
		t.Fatalf("suspended partner's owner must not be able to log in")
	}

	// --- reactivate: storefront access is restored ---
	if w := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/reactivate", partnerID), admin, nil); w.Code != http.StatusOK {
		t.Fatalf("reactivate failed: %s", w.Body.String())
	}
	if !catalogContains(productID) {
		t.Fatalf("reactivated partner's product must reappear in the public catalog")
	}
	lW = DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": "storefront@mitra.com", "password": password,
	})
	if lW.Code != http.StatusOK {
		t.Fatalf("reactivated partner's owner must be able to log in again: %s", lW.Body.String())
	}
}

// ARY-004: an account disabled directly by an admin (not via suspension) must
// not be re-enabled as a side effect of a partner suspend/reactivate cycle.
func TestSecurity_ARY004_ManuallyDisabledAccountStaysDisabled(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	const password = "KataSandiKuat#2026"
	const email = "manualoff@mitra.com"

	regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
		"company_name": "PT Manual Disable", "contact_person": "PIC Manual",
		"email": email, "phone": "081298765000",
	})
	if regW.Code != http.StatusCreated {
		t.Fatalf("register failed: %s", regW.Body.String())
	}
	var reg struct {
		Data struct {
			ID               int    `json:"id"`
			RegistrationCode string `json:"registration_code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &reg)
	if vW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/verify", reg.Data.ID), admin, map[string]string{
		"action": "approve", "reason": "ok",
	}); vW.Code != http.StatusOK {
		t.Fatalf("verify failed: %s", vW.Body.String())
	}
	var st struct {
		Data struct {
			ActivationToken string `json:"activation_token"`
		} `json:"data"`
	}
	stW := DoReq(r, http.MethodGet, "/api/m3/v1/partners/status?code="+reg.Data.RegistrationCode, "", nil)
	_ = json.Unmarshal(stW.Body.Bytes(), &st)
	if aW := DoReq(r, http.MethodPost, "/api/m1/v1/auth/activate", "", map[string]string{
		"token": st.Data.ActivationToken, "password": password,
	}); aW.Code != http.StatusOK {
		t.Fatalf("activate failed: %s", aW.Body.String())
	}

	// Resolve the owner account ID from the partner record.
	detW := DoReq(r, http.MethodGet, fmt.Sprintf("/api/m3/v1/partners/%d", reg.Data.ID), admin, nil)
	var det struct {
		Data struct {
			Partner struct {
				UserID *int `json:"user_id"`
			} `json:"partner"`
		} `json:"data"`
	}
	_ = json.Unmarshal(detW.Body.Bytes(), &det)
	if det.Data.Partner.UserID == nil {
		t.Fatalf("partner owner account must be linked after verification")
	}

	// Admin disables the account directly (not via suspension).
	if w := DoReq(r, http.MethodDelete, fmt.Sprintf("/api/m1/v1/users/%d", *det.Data.Partner.UserID), admin, nil); w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("deactivate owner account failed: %s", w.Body.String())
	}

	// Suspend then reactivate the partner.
	_ = DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/suspend", reg.Data.ID), admin, map[string]string{"reason": "Audit"})
	_ = DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/reactivate", reg.Data.ID), admin, nil)

	// The manually disabled account must remain unusable.
	lW := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
		"email": email, "password": password,
	})
	if lW.Code == http.StatusOK {
		t.Fatalf("manually disabled account must not be re-enabled by partner reactivation")
	}
}
