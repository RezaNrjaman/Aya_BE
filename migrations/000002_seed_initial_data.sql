-- ==============================================================================
-- Araya Business Hub Platform — Initial Seed Data (000002_seed_initial_data.sql)
-- Target Platform: PostgreSQL via Supabase SQL Editor
-- ==============================================================================

-- 1. Divisions
INSERT INTO divisions (name, description) VALUES
    ('Headquarter', 'Kantor Pusat Manajemen & Direksi'),
    ('Operasional & Bisnis', 'Divisi Operasional & Manajemen Unit Usaha'),
    ('Keuangan & Akuntansi', 'Divisi Finansial & Pembukuan'),
    ('Teknologi & Digital', 'Divisi IT, Rekayasa Perangkat Lunak & Sistem')
ON CONFLICT (name) DO NOTHING;

-- 2. Business Categories
INSERT INTO business_categories (name, description) VALUES
    ('Konstruksi & Bangunan', 'Jasa konstruksi sipil, arsitektur, dan perbaikan gedung'),
    ('Teknologi & Solusi Digital', 'Pengembangan software, infrastruktur IT, dan automasi'),
    ('Perdagangan & Suplai', 'Distribusi material, perlengkapan kantor, dan retail'),
    ('Jasa & Konsultansi Bisnis', 'Konsultasi manajemen, logistik, dan layanan profesional')
ON CONFLICT (name) DO NOTHING;

-- 3. Platform Roles
INSERT INTO roles (name, display_name, description, is_active) VALUES
    ('admin', 'Super Administrator', 'Akses penuh ke seluruh modul dan konfigurasi sistem', TRUE),
    ('staff', 'Staff Operasional', 'Mengelola transaksi harian, produk, dan permintaan layanan', TRUE),
    ('partner', 'Mitra Bisnis', 'Mitra terdaftar penyedia barang atau jasa', TRUE),
    ('customer', 'Pelanggan Portal', 'Akses publik dan checkout katalog portal pelanggan', TRUE)
ON CONFLICT (name) DO NOTHING;

-- 4. Baseline Permissions (M1 - M5)
INSERT INTO permissions (key, module, name, description, is_active) VALUES
    -- M1
    ('m1.users.read', 'm1', 'Lihat Daftar Pengguna', 'Membaca profil dan akun pengguna', TRUE),
    ('m1.users.create', 'm1', 'Tambah Pengguna', 'Membuat akun pengguna baru', TRUE),
    ('m1.users.update', 'm1', 'Ubah Pengguna', 'Memperbarui profil dan divisi pengguna', TRUE),
    ('m1.users.delete', 'm1', 'Nonaktifkan Pengguna', 'Menonaktifkan akun pengguna', TRUE),
    ('m1.roles.read', 'm1', 'Lihat Role & Permission', 'Melihat daftar role dan hak akses', TRUE),
    ('m1.roles.manage', 'm1', 'Kelola Role & Permission', 'Menambah, mengubah, dan memetakan permission ke role', TRUE),
    ('m1.audit.read', 'm1', 'Lihat Audit Log', 'Membaca riwayat aktivitas sistem', TRUE),
    -- M2
    ('m2.business_units.read', 'm2', 'Lihat Unit Usaha', 'Melihat profil dan daftar unit usaha', TRUE),
    ('m2.business_units.create', 'm2', 'Tambah Unit Usaha', 'Mendaftarkan unit usaha baru', TRUE),
    ('m2.business_units.update', 'm2', 'Ubah Unit Usaha', 'Memperbarui data dan status unit usaha', TRUE),
    ('m2.dashboard.read', 'm2', 'Lihat Dashboard', 'Mengakses dashboard operasional dan ringkasan data', TRUE),
    -- M3
    ('m3.partners.read', 'm3', 'Lihat Partner', 'Melihat data partner dan status pendaftaran', TRUE),
    ('m3.partners.register', 'm3', 'Daftar Partner', 'Mengajukan pendaftaran partner', TRUE),
    ('m3.partners.verify', 'm3', 'Verifikasi Partner', 'Menyetujui atau menolak pendaftaran partner', TRUE),
    ('m3.partners.suspend', 'm3', 'Tangguhkan Partner', 'Menangguhkan atau mengaktifkan kembali partner', TRUE),
    -- M4
    ('m4.products.read', 'm4', 'Lihat Produk', 'Melihat katalog produk internal', TRUE),
    ('m4.products.manage', 'm4', 'Kelola Produk', 'Menambah dan mengubah produk', TRUE),
    ('m4.customers.manage', 'm4', 'Kelola Pelanggan', 'Menambah dan mengubah data pelanggan', TRUE),
    ('m4.quotations.manage', 'm4', 'Kelola Quotation', 'Membuat dan menyetujui quotation', TRUE),
    ('m4.invoices.manage', 'm4', 'Kelola Invoice', 'Membuat dan mengelola invoice', TRUE),
    ('m4.invoices.pay', 'm4', 'Bayar Invoice', 'Melakukan simulasi pembayaran invoice sandbox', TRUE),
    ('m4.orders.manage', 'm4', 'Kelola Order', 'Memperbarui status order', TRUE),
    -- M5
    ('m5.catalog.read', 'm5', 'Akses Katalog Portal', 'Membaca katalog publik dan unit usaha', TRUE),
    ('m5.services.request', 'm5', 'Kirim Permintaan Layanan', 'Mengirim formulir permintaan layanan', TRUE),
    ('m5.services.manage', 'm5', 'Kelola Permintaan Layanan', 'Membaca dan memproses permintaan layanan customer', TRUE),
    ('m5.cart.manage', 'm5', 'Kelola Keranjang', 'Menambah dan mengubah item keranjang belanja', TRUE),
    ('m5.checkout.create', 'm5', 'Checkout Order', 'Melakukan checkout keranjang belanja ke kontrak order M4', TRUE)
ON CONFLICT (key) DO NOTHING;

-- 5. Role Permission Mappings
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Staff permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'staff' AND p.key IN (
    'm1.users.read',
    'm2.business_units.read', 'm2.dashboard.read',
    'm3.partners.read',
    'm4.products.read', 'm4.products.manage', 'm4.customers.manage',
    'm4.quotations.manage', 'm4.invoices.manage', 'm4.orders.manage',
    'm5.catalog.read', 'm5.services.manage'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Partner permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'partner' AND p.key IN (
    'm3.partners.read', 'm3.partners.register',
    'm4.products.read', 'm4.products.manage', 'm4.quotations.manage'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Customer permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'customer' AND p.key IN (
    'm5.catalog.read', 'm5.services.request', 'm5.cart.manage', 'm5.checkout.create'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 6. Default Super Admin User (Password: Admin123!)
INSERT INTO users (email, password_hash, name, phone, division_id, is_active)
SELECT 
    'admin@araya.com',
    '$2a$10$NMqdq89LCRUfWWrndQIZsewssdM0o7nJE.MWLguFCcxp/ZbTYUh5q',
    'Super Admin Araya',
    '081234567890',
    d.id,
    TRUE
FROM divisions d
WHERE d.name = 'Headquarter'
ON CONFLICT (email) DO NOTHING;

-- Assign Admin Role to Super Admin User
INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id)
SELECT u.id, r.id, NULL
FROM users u, roles r
WHERE u.email = 'admin@araya.com' AND r.name = 'admin'
ON CONFLICT (user_id, role_id, unit_usaha_id) DO NOTHING;

-- 7. Sample Business Units
INSERT INTO business_units (name, category_id, division_id, description, contact_email, contact_phone, address, is_active)
SELECT 
    'Araya Konstruksi Pratama',
    c.id,
    d.id,
    'Penyedia solusi konstruksi bangunan sipil, renovasi kantor, dan material arsitektur berkualitas tinggi.',
    'konstruksi@araya.com',
    '021-5551234',
    'Jl. Jend. Sudirman Kav. 21, Jakarta Selatan',
    TRUE
FROM business_categories c, divisions d
WHERE c.name = 'Konstruksi & Bangunan' AND d.name = 'Operasional & Bisnis'
ON CONFLICT DO NOTHING;

INSERT INTO business_units (name, category_id, division_id, description, contact_email, contact_phone, address, is_active)
SELECT 
    'Araya Solusi Teknologi',
    c.id,
    d.id,
    'Pengembangan platform digital, software enterprise, dan integrasi cloud modern.',
    'tech@araya.com',
    '021-5555678',
    'Jl. TB Simatupang No. 88, Jakarta Selatan',
    TRUE
FROM business_categories c, divisions d
WHERE c.name = 'Teknologi & Solusi Digital' AND d.name = 'Teknologi & Digital'
ON CONFLICT DO NOTHING;

-- 8. Sample Products
INSERT INTO products (unit_usaha_id, name, category, description, price, stock, is_active)
SELECT 
    bu.id,
    'Paket Renovasi Ruang Kantor Modern',
    'Layanan Konstruksi',
    'Layanan renovasi interior lengkap dengan partisi acoustic, elektrikal, dan pencahayaan LED.',
    15000000,
    50,
    TRUE
FROM business_units bu
WHERE bu.name = 'Araya Konstruksi Pratama'
ON CONFLICT DO NOTHING;

INSERT INTO products (unit_usaha_id, name, category, description, price, stock, is_active)
SELECT 
    bu.id,
    'Lisensi ERP Business Hub Core (Tahunan)',
    'Software Enterprise',
    'Paket cloud ERP lengkap mencakup modul Keuangan, Pengadaan, Mitra, dan E-Commerce.',
    7500000,
    100,
    TRUE
FROM business_units bu
WHERE bu.name = 'Araya Solusi Teknologi'
ON CONFLICT DO NOTHING;

