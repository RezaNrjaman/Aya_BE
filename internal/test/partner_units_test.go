package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ARY-005: partners manage their own business units. Unit ownership always
// comes from the authenticated actor; a partner can never see or mutate
// another partner's units.
func TestSecurity_ARY005_UnitUsahaPerPartner(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	const password = "KataSandiKuat#2026"

	_, tokA := OnboardVerifiedPartner(t, r, admin, "unit-a@mitra.com", "PT Unit A", password)
	_, tokB := OnboardVerifiedPartner(t, r, admin, "unit-b@mitra.com", "PT Unit B", password)

	// Each partner creates its own unit; the body partner_id is ignored.
	createA := DoReq(r, http.MethodPost, "/api/m2/v1/business-units", tokA, map[string]any{
		"name": "Unit A Milik Partner A", "category": "Produksi", "partner_id": 999,
	})
	if createA.Code != http.StatusCreated {
		t.Fatalf("create unit A failed: %s", createA.Body.String())
	}
	var unitA struct {
		Data struct {
			ID        int  `json:"id"`
			PartnerID *int `json:"partner_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(createA.Body.Bytes(), &unitA)
	if unitA.Data.PartnerID == nil || *unitA.Data.PartnerID == 999 {
		t.Fatalf("unit tenant must come from the actor, not the body: %s", createA.Body.String())
	}
	createB := DoReq(r, http.MethodPost, "/api/m2/v1/business-units", tokB, map[string]any{
		"name": "Unit B Milik Partner B", "category": "Kuliner",
	})
	if createB.Code != http.StatusCreated {
		t.Fatalf("create unit B failed: %s", createB.Body.String())
	}
	var unitB struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(createB.Body.Bytes(), &unitB)

	// A cannot read or update B's unit.
	if w := DoReq(r, http.MethodGet, unitURL(unitB.Data.ID), tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("partner A must not read partner B's unit, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPatch, unitURL(unitB.Data.ID), tokA, map[string]any{
		"name": "Direbut A",
	}); w.Code != http.StatusNotFound {
		t.Fatalf("partner A must not update partner B's unit, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPost, unitURL(unitB.Data.ID)+"/deactivate", tokA, nil); w.Code != http.StatusNotFound {
		t.Fatalf("partner A must not deactivate partner B's unit, got %d", w.Code)
	}

	// A's admin list contains only A's own unit.
	listA := DoReq(r, http.MethodGet, "/api/m2/v1/admin/business-units", tokA, nil)
	var list struct {
		Data []struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listA.Body.Bytes(), &list)
	for _, u := range list.Data {
		if u.ID == unitB.Data.ID {
			t.Fatalf("partner A's unit list leaked partner B's unit")
		}
	}

	// Super admin sees every unit, including platform units.
	listAdmin := DoReq(r, http.MethodGet, "/api/m2/v1/admin/business-units", admin, nil)
	_ = json.Unmarshal(listAdmin.Body.Bytes(), &list)
	sawA, sawB := false, false
	for _, u := range list.Data {
		if u.ID == unitA.Data.ID {
			sawA = true
		}
		if u.ID == unitB.Data.ID {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Fatalf("super admin must see all tenants' units (sawA=%v sawB=%v)", sawA, sawB)
	}
}

func unitURL(id int) string {
	return fmt.Sprintf("/api/m2/v1/business-units/%d", id)
}
