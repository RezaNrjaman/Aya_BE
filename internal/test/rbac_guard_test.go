package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// ARY-003: partner members must never be able to create platform roles,
// alter platform permissions, or escalate to a super-admin role — even when
// their role mistakenly holds m1.roles.manage.
func TestSecurity_ARY003_PartnerCannotManagePlatformRoles(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)

	// Deliberate misconfiguration: the partner role gains m1.roles.manage.
	if w := DoReq(r, http.MethodPost, "/api/m1/v1/roles/partner/permissions", admin, map[string]string{
		"permission_key": "m1.roles.manage",
	}); w.Code != http.StatusOK {
		t.Fatalf("grant failed: %s", w.Body.String())
	}

	_, partnerTok := OnboardVerifiedPartner(t, r, admin, "eskalasi@mitra.com", "PT Eskalasi Nol", "KataSandiKuat#2026")

	// Creating a platform role is denied at the service layer.
	if w := DoReq(r, http.MethodPost, "/api/m1/v1/roles", partnerTok, map[string]any{
		"name": "super_admin_palsu", "display_name": "Super Admin Palsu",
		"permission_keys": []string{"m1.users.manage", "m1.roles.manage", "m3.partners.verify"},
	}); w.Code != http.StatusForbidden {
		t.Fatalf("partner must not create platform roles, got %d: %s", w.Code, w.Body.String())
	}

	// Altering platform role permissions is denied, including on its own role
	// and on the protected admin role.
	if w := DoReq(r, http.MethodPost, "/api/m1/v1/roles/partner/permissions", partnerTok, map[string]string{
		"permission_key": "m3.partners.suspend",
	}); w.Code != http.StatusForbidden {
		t.Fatalf("partner must not assign platform permissions, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPut, "/api/m1/v1/roles/admin/permissions", partnerTok, map[string]any{
		"permission_keys": []string{"m1.roles.read"},
	}); w.Code != http.StatusForbidden {
		t.Fatalf("partner must not replace the admin role's permissions, got %d", w.Code)
	}
	if w := DoReq(r, http.MethodPatch, "/api/m1/v1/roles/admin", partnerTok, map[string]any{
		"is_active": false,
	}); w.Code != http.StatusForbidden {
		t.Fatalf("partner must not deactivate platform roles, got %d", w.Code)
	}

	// Escalating another tenant member to a super-admin role is denied.
	if w := DoReq(r, http.MethodPost, "/api/m1/v1/users", partnerTok, map[string]any{
		"email": "staf@eskalasi.com", "name": "Staf", "password": "KataSandiKuat#2026",
	}); w.Code != http.StatusCreated {
		t.Fatalf("partner create user failed: %s", w.Body.String())
	}
	listW := DoReq(r, http.MethodGet, "/api/m1/v1/users?search=staf@eskalasi.com", partnerTok, nil)
	var list struct {
		Data []struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listW.Body.Bytes(), &list)
	if len(list.Data) == 0 {
		t.Fatalf("created staff user not found")
	}
	if w := DoReq(r, http.MethodPost, fmt.Sprintf("/api/m1/v1/users/%d/roles", list.Data[0].ID), partnerTok, map[string]any{
		"role_name": "admin",
	}); w.Code == http.StatusOK {
		t.Fatalf("partner must not assign the admin role to its members")
	}

	// The admin role's permission set is untouched.
	getW := DoReq(r, http.MethodGet, "/api/m1/v1/roles", admin, nil)
	var roles struct {
		Data []struct {
			Name        string `json:"name"`
			Permissions []struct {
				Key string `json:"key"`
			} `json:"permissions"`
		} `json:"data"`
	}
	_ = json.Unmarshal(getW.Body.Bytes(), &roles)
	for _, role := range roles.Data {
		if role.Name != "admin" {
			continue
		}
		var hasUsersManage, hasPartnersVerify bool
		for _, p := range role.Permissions {
			if p.Key == "m1.users.create" {
				hasUsersManage = true
			}
			if p.Key == "m3.partners.verify" {
				hasPartnersVerify = true
			}
		}
		if !hasUsersManage || !hasPartnersVerify {
			t.Fatalf("admin role permissions were altered: %s", getW.Body.String())
		}
	}
}
