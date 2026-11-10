package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ARY-001: two tenants may coexist with the same resources, yet neither can
// read, update, or create on behalf of the other. Tenant identity always comes
// from the authenticated actor, never from the request body.
func TestSecurity_ARy001_PartnerIsolation_Products(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)

	activate := func(email, company string) string {
		regW := DoReq(r, http.MethodPost, "/api/m3/v1/partners/register", "", map[string]any{
			"company_name": company, "contact_person": "PIC", "email": email, "phone": "0812000000",
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
		if vW := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m3/v1/partners/%d/verify", reg.Data.ID), admin, map[string]string{
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
			"token": st.Data.ActivationToken, "password": "KataSandiKuat#2026",
		}); aW.Code != http.StatusOK {
			t.Fatalf("activate %s failed: %s", email, aW.Body.String())
		}
		lW := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", map[string]string{
			"email": email, "password": "KataSandiKuat#2026",
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
		return login.Data.Token
	}

	tokA := activate("mitra-a@isolasi.com", "PT Mitra A")
	tokB := activate("mitra-b@isolasi.com", "PT Mitra B")

	// Each partner creates a product; the request body partner_id is ignored.
	createA := DoReq(r, http.MethodPost, "/api/m4/v1/products", tokA, map[string]any{
		"unit_usaha_id": 1, "name": "Produk Milik A", "category": "Percetakan", "price": 10000, "stock": 5, "partner_id": 999,
	})
	if createA.Code != http.StatusCreated {
		t.Fatalf("create product A failed: %s", createA.Body.String())
	}
	var prodA struct {
		Data struct {
			ID        int  `json:"id"`
			PartnerID *int `json:"partner_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(createA.Body.Bytes(), &prodA)
	if prodA.Data.PartnerID == nil || *prodA.Data.PartnerID == 999 {
		t.Fatalf("product tenant must come from the actor, not the body: %s", createA.Body.String())
	}
	createB := DoReq(r, http.MethodPost, "/api/m4/v1/products", tokB, map[string]any{
		"unit_usaha_id": 1, "name": "Produk Milik B", "category": "Percetakan", "price": 20000, "stock": 5,
	})
	if createB.Code != http.StatusCreated {
		t.Fatalf("create product B failed: %s", createB.Body.String())
	}
	var prodB struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(createB.Body.Bytes(), &prodB)

	// A cannot read or update B's product even knowing its ID.
	if w := DoReq(r, http.MethodGet, fmt.Sprintf("/api/m4/v1/products/%d", prodB.Data.ID), tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("partner A must not read partner B's product, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPatch, fmt.Sprintf("/api/m4/v1/products/%d", prodB.Data.ID), tokA, map[string]any{
		"price": 1,
	}); w.Code != http.StatusNotFound {
		t.Fatalf("partner A must not update partner B's product, got %d", w.Code)
	}

	// A's admin list contains only A's own product.
	listA := DoReq(r, http.MethodGet, "/api/m4/v1/admin/products", tokA, nil)
	var list struct {
		Data []struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listA.Body.Bytes(), &list)
	for _, p := range list.Data {
		if p.ID == prodB.Data.ID {
			t.Fatalf("partner A's product list leaked partner B's product")
		}
	}

	// Super admin still sees everything.
	listAdmin := DoReq(r, http.MethodGet, "/api/m4/v1/admin/products", admin, nil)
	_ = json.Unmarshal(listAdmin.Body.Bytes(), &list)
	var sawA, sawB bool
	for _, p := range list.Data {
		if p.ID == prodA.Data.ID {
			sawA = true
		}
		if p.ID == prodB.Data.ID {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Fatalf("super admin must see both tenants' products (sawA=%v sawB=%v)", sawA, sawB)
	}
}
