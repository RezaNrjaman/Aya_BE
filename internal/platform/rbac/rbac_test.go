package rbac_test

import (
	"context"
	"testing"

	"araya-business-hub/be/internal/platform/rbac"
)

func TestPermissionRegistry(t *testing.T) {
	reg := rbac.NewRegistry()

	// 1. Register valid permission
	p1 := rbac.Permission{
		Key:      "m1.users.read",
		Module:   "m1",
		Name:     "Lihat Pengguna",
		IsActive: true,
	}
	if err := reg.Register(p1); err != nil {
		t.Fatalf("failed to register valid permission: %v", err)
	}

	// 2. Reject duplicate key
	if err := reg.Register(p1); err == nil {
		t.Fatalf("expected error on duplicate permission key, got nil")
	}

	// 3. Reject invalid format (not lowercase namespaced)
	pBad := rbac.Permission{
		Key:    "badkey",
		Module: "m1",
		Name:   "Bad",
	}
	if err := reg.Register(pBad); err == nil {
		t.Fatalf("expected error for non-namespaced key, got nil")
	}

	// 4. Deactivate permission
	if err := reg.Deactivate("m1.users.read"); err != nil {
		t.Fatalf("failed to deactivate permission: %v", err)
	}

	p, err := reg.Get("m1.users.read")
	if err != nil || p.IsActive {
		t.Fatalf("expected permission to be deactivated, got active")
	}
}

func TestDynamicPolicyEvaluation_OD01(t *testing.T) {
	ctx := context.Background()
	reg := rbac.NewRegistry()
	repo := rbac.NewInMemoryRepository()

	// Register supported permissions
	_ = reg.Register(rbac.Permission{
		Key:      "m4.invoices.create",
		Module:   "m4",
		Name:     "Buat Invoice",
		IsActive: true,
	})
	_ = reg.Register(rbac.Permission{
		Key:      "m4.invoices.read",
		Module:   "m4",
		Name:     "Lihat Invoice",
		IsActive: true,
	})

	// Create Role 'kasir'
	_ = repo.CreateRole(ctx, rbac.Role{
		Name:        "kasir",
		DisplayName: "Kasir Unit",
		IsActive:    true,
	})

	// Assign permissions to 'kasir'
	_ = repo.AssignPermission(ctx, "kasir", "m4.invoices.read")
	_ = repo.AssignPermission(ctx, "kasir", "m4.invoices.create")

	policy := rbac.NewPolicyService(reg, repo)

	// A. Kasir can perform both
	if !policy.CanRolePerform(ctx, "kasir", "m4.invoices.read") {
		t.Fatalf("expected kasir to be allowed to read invoices")
	}
	if !policy.CanRolePerform(ctx, "kasir", "m4.invoices.create") {
		t.Fatalf("expected kasir to be allowed to create invoices")
	}

	// B. Administrator dynamically revokes 'm4.invoices.create' from 'kasir'
	_ = repo.RevokePermission(ctx, "kasir", "m4.invoices.create")

	// Verify OD-01: immediately denied without waiting for token expiration!
	if policy.CanRolePerform(ctx, "kasir", "m4.invoices.create") {
		t.Fatalf("expected immediate revocation of create permission (OD-01 mitigation failed)")
	}
	// But read permission is still retained
	if !policy.CanRolePerform(ctx, "kasir", "m4.invoices.read") {
		t.Fatalf("expected read permission to still be active")
	}

	// C. Deactivate the role completely
	_ = repo.DeactivateRole(ctx, "kasir")
	if policy.CanRolePerform(ctx, "kasir", "m4.invoices.read") {
		t.Fatalf("expected deactivated role to be denied all permissions")
	}
}

func TestScopePermitted(t *testing.T) {
	reg := rbac.NewRegistry()
	repo := rbac.NewInMemoryRepository()
	policy := rbac.NewPolicyService(reg, repo)

	unit3 := 3

	// Manager restricted to unit 3
	if !policy.IsScopePermitted("manajer", &unit3, 3) {
		t.Errorf("expected manager with unit 3 to access unit 3")
	}
	if policy.IsScopePermitted("manajer", &unit3, 5) {
		t.Errorf("expected manager with unit 3 to be denied unit 5")
	}

	// Admin has global bypass
	if !policy.IsScopePermitted("admin", &unit3, 5) {
		t.Errorf("expected admin to have global scope regardless of unit_usaha_id")
	}
}
