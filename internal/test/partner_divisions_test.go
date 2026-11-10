package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ARY-006: each partner maintains its own divisions. Names may repeat across
// tenants but never within one; cross-tenant access is impossible; divisions
// still in use cannot be deleted.
func TestSecurity_ARY006_DivisiPerPartner(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	const password = "KataSandiKuat#2026"

	_, tokA := OnboardVerifiedPartner(t, r, admin, "div-a@mitra.com", "PT Divisi A", password)
	_, tokB := OnboardVerifiedPartner(t, r, admin, "div-b@mitra.com", "PT Divisi B", password)

	// Same division name under two different tenants is allowed (ARY-006:
	// uniqueness is per partner, not global).
	var mkDiv = func(tok, name string) int {
		w := DoReq(r, http.MethodPost, "/api/m2/v1/divisions", tok, map[string]any{"name": name})
		if w.Code != http.StatusCreated {
			t.Fatalf("create division %s failed: %s", name, w.Body.String())
		}
		var resp struct {
			Data struct {
				ID        int `json:"id"`
				PartnerID *int `json:"partner_id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.Data.ID
	}
	divA := mkDiv(tokA, "Finance")
	divB := mkDiv(tokB, "Finance")
	if divA == divB {
		t.Fatalf("expected distinct division ids across tenants")
	}

	// Duplicate within the same tenant is rejected.
	if w := DoReq(r, http.MethodPost, "/api/m2/v1/divisions", tokA, map[string]any{"name": "finance"}); w.Code == http.StatusCreated {
		t.Fatalf("duplicate division name within one tenant must be rejected")
	}

	// B cannot read, update, deactivate, or delete A's division.
	divURL := func(id int) string { return fmt.Sprintf("/api/m2/v1/divisions/%d", id) }
	if w := DoReq(r, http.MethodGet, divURL(divA), tokB, nil); w.Code != http.StatusNotFound {
		t.Fatalf("partner B must not read partner A's division, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPatch, divURL(divA), tokB, map[string]any{"name": "Direbut B"}); w.Code != http.StatusNotFound {
		t.Fatalf("partner B must not update partner A's division, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPost, divURL(divA)+"/deactivate", tokB, nil); w.Code != http.StatusNotFound {
		t.Fatalf("partner B must not deactivate partner A's division, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodDelete, divURL(divA), tokB, nil); w.Code != http.StatusNotFound {
		t.Fatalf("partner B must not delete partner A's division, got %d", w.Code)
	}

	// A attaches its division to a unit; deletion must then be refused.
	createUnit := DoReq(r, http.MethodPost, "/api/m2/v1/business-units", tokA, map[string]any{
		"name": "Unit Divisi A", "division_id": divA,
	})
	if createUnit.Code != http.StatusCreated {
		t.Fatalf("create unit failed: %s", createUnit.Body.String())
	}
	var unit struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(createUnit.Body.Bytes(), &unit)
	if w := DoReq(r, http.MethodDelete, divURL(divA), tokA, nil); w.Code != http.StatusConflict {
		t.Fatalf("deleting an in-use division must be refused with 409, got %d: %s", w.Code, w.Body.String())
	}

	// Deactivation (soft) still works and keeps history intact.
	if w := DoReq(r, http.MethodPost, divURL(divA)+"/deactivate", tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("deactivate in-use division failed: %s", w.Body.String())
	}
	var got struct {
		Data struct {
			IsActive bool `json:"is_active"`
		} `json:"data"`
	}
	getW := DoReq(r, http.MethodGet, divURL(divA), tokA, nil)
	_ = json.Unmarshal(getW.Body.Bytes(), &got)
	if got.Data.IsActive {
		t.Fatalf("division must be inactive after deactivation")
	}

	// A division without references can be deleted by its owner.
	unused := mkDiv(tokA, "Divisi Kosong")
	if w := DoReq(r, http.MethodDelete, divURL(unused), tokA, nil); w.Code != http.StatusOK {
		t.Fatalf("deleting an unused division failed: %s", w.Body.String())
	}
	if w := DoReq(r, http.MethodGet, divURL(unused), tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("deleted division must be gone, got %d", w.Code)
	}
}
