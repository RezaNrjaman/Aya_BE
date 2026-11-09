package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/platform/rbac"
	"github.com/gin-gonic/gin"
)

// Access-control tests: role/permission configuration done by an administrator
// must be enforced by the backend immediately and safely.

func loginAs(t *testing.T, r *gin.Engine, email, password string) (string, []string) {
	t.Helper()
	w := DoReq(r, http.MethodPost, "/api/m1/v1/login", "", dto.LoginRequest{Email: email, Password: password})
	if w.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d %s", email, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				Permissions []string `json:"permissions"`
			} `json:"user"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data.Token, resp.Data.User.Permissions
}

func createUserAs(t *testing.T, r *gin.Engine, token, email, role string) int {
	t.Helper()
	w := DoReq(r, http.MethodPost, "/api/m1/v1/users", token, map[string]any{
		"email": email, "password": "Rahasia123!", "name": email, "role": role,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create user %s: %d %s", email, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data.ID
}

func expectStatus(t *testing.T, w interface{ Result() *http.Response }, want int, label string) {
	t.Helper()
	if got := w.Result().StatusCode; got != want {
		t.Fatalf("%s: expected %d, got %d", label, want, got)
	}
}

func setRolePerms(r *gin.Engine, token, role string, keys []string) int {
	w := DoReq(r, http.MethodPut, "/api/m1/v1/roles/"+role+"/permissions", token, map[string]any{"permission_keys": keys})
	return w.Code
}

func TestAccess_PermissionMatrixAppliesImmediately(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	createUserAs(t, r, admin, "kasir1@araya.com", "kasir")
	kasir, perms := loginAs(t, r, "kasir1@araya.com", "Rahasia123!")
	if len(perms) == 0 {
		t.Fatal("kasir should start with default permissions")
	}
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m4/v1/invoices", kasir, nil), http.StatusOK, "kasir reads invoices")

	if code := setRolePerms(r, admin, "kasir", []string{"m4.products.read"}); code != http.StatusOK {
		t.Fatalf("set permissions failed: %d", code)
	}
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m4/v1/invoices", kasir, nil), http.StatusForbidden, "revoked takes effect with same token")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m4/v1/products", kasir, nil), http.StatusOK, "granted takes effect with same token")

	w := DoReq(r, http.MethodGet, "/api/m1/v1/me", kasir, nil)
	var me struct {
		Data struct {
			Permissions []string `json:"permissions"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &me)
	if len(me.Data.Permissions) != 1 || me.Data.Permissions[0] != "m4.products.read" {
		t.Fatalf("profile should expose updated permissions, got %v", me.Data.Permissions)
	}
}

func TestAccess_InactiveRoleGrantsNothing(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	createUserAs(t, r, admin, "kasir2@araya.com", "kasir")
	kasir, _ := loginAs(t, r, "kasir2@araya.com", "Rahasia123!")
	expectStatus(t, DoReq(r, http.MethodPatch, "/api/m1/v1/roles/kasir", admin, map[string]any{"is_active": false}), http.StatusOK, "deactivate role")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m4/v1/invoices", kasir, nil), http.StatusForbidden, "inactive role")
	expectStatus(t, DoReq(r, http.MethodPost, "/api/m1/v1/users", admin, map[string]any{
		"email": "x@araya.com", "password": "Rahasia123!", "name": "X", "role": "kasir",
	}), http.StatusUnprocessableEntity, "cannot assign inactive role")
}

func TestAccess_RejectsUnknownPermissionAndBadRole(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	if code := setRolePerms(r, admin, "kasir", []string{"m9.hack.all"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown permission must be rejected, got %d", code)
	}
	bad := DoReq(r, http.MethodPost, "/api/m1/v1/roles", admin, map[string]any{"name": "Bad Role!", "display_name": "Bad"})
	expectStatus(t, bad, http.StatusUnprocessableEntity, "invalid role name")
	ok := DoReq(r, http.MethodPost, "/api/m1/v1/roles", admin, map[string]any{
		"name": "supervisor", "display_name": "Supervisor", "permission_keys": []string{"m4.orders.manage"},
	})
	expectStatus(t, ok, http.StatusCreated, "create role with permissions")
	dup := DoReq(r, http.MethodPost, "/api/m1/v1/roles", admin, map[string]any{"name": "supervisor", "display_name": "Lagi"})
	expectStatus(t, dup, http.StatusConflict, "duplicate role")
	expectStatus(t, DoReq(r, http.MethodPost, "/api/m1/v1/users", admin, map[string]any{
		"email": "y@araya.com", "password": "Rahasia123!", "name": "Y", "role": "tidak_ada",
	}), http.StatusUnprocessableEntity, "unknown role on create")
}

func TestAccess_LockoutProtection(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	if code := setRolePerms(r, admin, "admin", []string{"m1.users.read"}); code != http.StatusForbidden {
		t.Fatalf("removing roles.manage from admin must be blocked, got %d", code)
	}
	expectStatus(t, DoReq(r, http.MethodPatch, "/api/m1/v1/roles/admin", admin, map[string]any{"is_active": false}), http.StatusForbidden, "deactivate admin role")
	expectStatus(t, DoReq(r, http.MethodDelete, "/api/m1/v1/roles/admin/permissions/m1.roles.manage", admin, nil), http.StatusForbidden, "revoke roles.manage from admin")
	expectStatus(t, DoReq(r, http.MethodPost, "/api/m1/v1/users/1/roles", admin, map[string]any{"role_name": "kasir"}), http.StatusForbidden, "self role change")
}

func TestAccess_RoleChangeRequiresRolesManage(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	DoReq(r, http.MethodPost, "/api/m1/v1/roles", admin, map[string]any{
		"name": "hrd", "display_name": "HRD", "permission_keys": []string{"m1.users.read", "m1.users.create", "m1.users.update"},
	})
	createUserAs(t, r, admin, "hrd@araya.com", "hrd")
	victim := createUserAs(t, r, admin, "biasa@araya.com", "kasir")
	hrd, _ := loginAs(t, r, "hrd@araya.com", "Rahasia123!")
	expectStatus(t, DoReq(r, http.MethodPatch, fmt.Sprintf("/api/m1/v1/users/%d", victim), hrd, map[string]any{"role": "admin"}), http.StatusForbidden, "escalate via PATCH")
	expectStatus(t, DoReq(r, http.MethodPost, "/api/m1/v1/users", hrd, map[string]any{
		"email": "z@araya.com", "password": "Rahasia123!", "name": "Z", "role": "admin",
	}), http.StatusForbidden, "escalate via create")
	expectStatus(t, DoReq(r, http.MethodPatch, fmt.Sprintf("/api/m1/v1/users/%d", victim), hrd, map[string]any{"name": "Nama Baru"}), http.StatusOK, "normal profile edit")
}

func TestAccess_UserRoleChangeAndDeactivationApplyToOpenSessions(t *testing.T) {
	r, _ := SetupFullTestApp()
	admin := LoginAdmin(r)
	id := createUserAs(t, r, admin, "pindah@araya.com", "kasir")
	tok, _ := loginAs(t, r, "pindah@araya.com", "Rahasia123!")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m4/v1/invoices", tok, nil), http.StatusOK, "kasir before change")

	expectStatus(t, DoReq(r, http.MethodPost, fmt.Sprintf("/api/m1/v1/users/%d/roles", id), admin, map[string]any{"role_name": "manajer"}), http.StatusOK, "assign manajer")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m4/v1/invoices", tok, nil), http.StatusForbidden, "old role no longer applies")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m2/v1/dashboard-summary", tok, nil), http.StatusOK, "new role applies")

	expectStatus(t, DoReq(r, http.MethodDelete, fmt.Sprintf("/api/m1/v1/users/%d", id), admin, nil), http.StatusOK, "deactivate")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m2/v1/dashboard-summary", tok, nil), http.StatusUnauthorized, "deactivated user")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m1/v1/me", tok, nil), http.StatusUnauthorized, "deactivated profile")

	expectStatus(t, DoReq(r, http.MethodPatch, fmt.Sprintf("/api/m1/v1/users/%d", id), admin, map[string]any{"is_active": true}), http.StatusOK, "reactivate")
	expectStatus(t, DoReq(r, http.MethodGet, "/api/m2/v1/dashboard-summary", tok, nil), http.StatusOK, "reactivated user")
}

func TestAccess_SeedKeepsConfiguredPermissions(t *testing.T) {
	repo := rbac.NewInMemoryRepository()
	ctx := context.Background()
	_ = rbac.SeedInitialRoles(ctx, repo)
	_ = repo.SetRolePermissions(ctx, "kasir", []string{"m4.products.read"})
	if err := rbac.SeedInitialRoles(ctx, repo); err != nil {
		t.Fatalf("re-seed failed: %v", err)
	}
	keys, _ := repo.GetRolePermissions(ctx, "kasir")
	if len(keys) != 1 || keys[0] != "m4.products.read" {
		t.Fatalf("re-seeding must not restore default permissions, got %v", keys)
	}
}
