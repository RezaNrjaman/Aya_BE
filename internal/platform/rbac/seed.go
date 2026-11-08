package rbac

import (
	"context"
	"fmt"
)

// SeedPlatformPermissions registers the official baseline permissions supported by modules M1-M5.
func SeedPlatformPermissions(registry *Registry) error {
	permissions := []Permission{
		// M1 — Authentication & Users
		{Key: "m1.users.read", Module: "m1", Name: "Lihat Daftar Pengguna", Description: "Membaca profil dan akun pengguna", IsActive: true},
		{Key: "m1.users.create", Module: "m1", Name: "Tambah Pengguna", Description: "Membuat akun pengguna baru", IsActive: true},
		{Key: "m1.users.update", Module: "m1", Name: "Ubah Pengguna", Description: "Memperbarui profil dan divisi pengguna", IsActive: true},
		{Key: "m1.users.delete", Module: "m1", Name: "Nonaktifkan Pengguna", Description: "Menonaktifkan akun pengguna", IsActive: true},
		{Key: "m1.roles.read", Module: "m1", Name: "Lihat Role & Permission", Description: "Melihat daftar role dan hak akses", IsActive: true},
		{Key: "m1.roles.manage", Module: "m1", Name: "Kelola Role & Permission", Description: "Menambah, mengubah, dan memetakan permission ke role", IsActive: true},
		{Key: "m1.audit.read", Module: "m1", Name: "Lihat Audit Log", Description: "Membaca riwayat aktivitas sistem", IsActive: true},

		// M2 — Business Units & Dashboard
		{Key: "m2.business_units.read", Module: "m2", Name: "Lihat Unit Usaha", Description: "Melihat profil dan daftar unit usaha", IsActive: true},
		{Key: "m2.business_units.create", Module: "m2", Name: "Tambah Unit Usaha", Description: "Mendaftarkan unit usaha baru", IsActive: true},
		{Key: "m2.business_units.update", Module: "m2", Name: "Ubah Unit Usaha", Description: "Memperbarui data dan status unit usaha", IsActive: true},
		{Key: "m2.dashboard.read", Module: "m2", Name: "Lihat Dashboard", Description: "Mengakses dashboard operasional dan ringkasan data", IsActive: true},

		// M3 — Partner Management
		{Key: "m3.partners.read", Module: "m3", Name: "Lihat Partner", Description: "Melihat data partner dan status pendaftaran", IsActive: true},
		{Key: "m3.partners.register", Module: "m3", Name: "Daftar Partner", Description: "Mengajukan pendaftaran partner", IsActive: true},
		{Key: "m3.partners.verify", Module: "m3", Name: "Verifikasi Partner", Description: "Menyetujui atau menolak pendaftaran partner", IsActive: true},
		{Key: "m3.partners.suspend", Module: "m3", Name: "Tangguhkan Partner", Description: "Menangguhkan atau mengaktifkan kembali partner", IsActive: true},

		// M4 — Transaction Management
		{Key: "m4.products.read", Module: "m4", Name: "Lihat Produk", Description: "Melihat katalog produk internal", IsActive: true},
		{Key: "m4.products.manage", Module: "m4", Name: "Kelola Produk", Description: "Menambah dan mengubah produk", IsActive: true},
		{Key: "m4.customers.manage", Module: "m4", Name: "Kelola Pelanggan", Description: "Menambah dan mengubah data pelanggan", IsActive: true},
		{Key: "m4.quotations.manage", Module: "m4", Name: "Kelola Quotation", Description: "Membuat dan menyetujui quotation", IsActive: true},
		{Key: "m4.invoices.manage", Module: "m4", Name: "Kelola Invoice", Description: "Membuat dan mengelola invoice", IsActive: true},
		{Key: "m4.invoices.pay", Module: "m4", Name: "Bayar Invoice", Description: "Melakukan simulasi pembayaran invoice sandbox", IsActive: true},
		{Key: "m4.orders.manage", Module: "m4", Name: "Kelola Order", Description: "Memperbarui status order", IsActive: true},

		// M5 — Customer Portal
		{Key: "m5.catalog.read", Module: "m5", Name: "Akses Katalog Portal", Description: "Membaca katalog publik dan unit usaha", IsActive: true},
		{Key: "m5.services.request", Module: "m5", Name: "Kirim Permintaan Layanan", Description: "Mengirim formulir permintaan layanan", IsActive: true},
		{Key: "m5.services.manage", Module: "m5", Name: "Kelola Permintaan Layanan", Description: "Membaca dan memproses permintaan layanan customer", IsActive: true},
		{Key: "m5.cart.manage", Module: "m5", Name: "Kelola Keranjang", Description: "Menambah dan mengubah item keranjang belanja", IsActive: true},
		{Key: "m5.checkout.create", Module: "m5", Name: "Checkout Order", Description: "Melakukan checkout keranjang belanja ke kontrak order M4", IsActive: true},
	}

	return registry.RegisterBatch(permissions)
}

// SeedInitialRoles creates baseline roles (admin, manajer, partner, kasir) with default permissions.
func SeedInitialRoles(ctx context.Context, repo Repository) error {
	roles := []struct {
		Role  Role
		Perms []string
	}{
		{
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator Sistem",
				Description: "Akses penuh manajemen operasional dan konfigurasi",
				IsActive:    true,
			},
			Perms: []string{
				"m1.users.read", "m1.users.create", "m1.users.update", "m1.users.delete",
				"m1.roles.read", "m1.roles.manage", "m1.audit.read",
				"m2.business_units.read", "m2.business_units.create", "m2.business_units.update", "m2.dashboard.read",
				"m3.partners.read", "m3.partners.verify", "m3.partners.suspend",
				"m4.products.read", "m4.products.manage", "m4.customers.manage",
				"m4.quotations.manage", "m4.invoices.manage", "m4.invoices.pay", "m4.orders.manage",
				"m5.catalog.read", "m5.services.request", "m5.services.manage", "m5.cart.manage", "m5.checkout.create",
			},
		},
		{
			Role: Role{
				Name:        "manajer",
				DisplayName: "Manajer Operasional",
				Description: "Monitoring kinerja dashboard dan operasional unit usaha",
				IsActive:    true,
			},
			Perms: []string{
				"m2.business_units.read", "m2.dashboard.read",
				"m4.products.read",
			},
		},
		{
			Role: Role{
				Name:        "partner",
				DisplayName: "Partner BPU",
				Description: "Penyedia jasa/produk terverifikasi",
				IsActive:    true,
			},
			Perms: []string{
				"m2.dashboard.read",
				"m3.partners.read", "m3.partners.register",
				"m4.products.read", "m4.products.manage", "m4.customers.manage",
				"m4.quotations.manage", "m4.invoices.manage", "m4.orders.manage",
			},
		},
		{
			Role: Role{
				Name:        "kasir",
				DisplayName: "Kasir Transaksi",
				Description: "Menangani pembayaran invoice dan kasir",
				IsActive:    true,
			},
			Perms: []string{
				"m4.invoices.manage", "m4.invoices.pay",
			},
		},
	}

	for _, r := range roles {
		if err := repo.CreateRole(ctx, r.Role); err != nil && err != ErrDuplicateRole {
			return fmt.Errorf("failed creating role %s: %w", r.Role.Name, err)
		}
		for _, p := range r.Perms {
			_ = repo.AssignPermission(ctx, r.Role.Name, p)
		}
	}

	return nil
}
