-- 000013_partner_portal_permissions.up.sql
-- Existing installations seeded by 000002 have only a subset of partner permissions.
-- Backfill every permission required by the current partner UI/routes and grant it
-- to the global system role `partner`. This is additive and does not alter custom roles.

INSERT INTO permissions (key, module, name, description, is_active)
VALUES
 ('m1.users.read','m1','Lihat Pengguna','Melihat anggota perusahaan',TRUE),
 ('m1.users.create','m1','Tambah Pengguna','Mengundang anggota perusahaan',TRUE),
 ('m1.users.update','m1','Ubah Pengguna','Mengubah anggota perusahaan',TRUE),
 ('m1.users.delete','m1','Nonaktifkan Pengguna','Menonaktifkan anggota perusahaan',TRUE),
 ('m1.roles.read','m1','Lihat Role','Melihat role perusahaan',TRUE),
 ('m1.roles.manage','m1','Kelola Role','Mengelola role dan hak akses perusahaan',TRUE),
 ('m1.audit.read','m1','Lihat Audit Log','Melihat audit log perusahaan',TRUE),
 ('m2.business_units.read','m2','Lihat Unit Usaha','Melihat unit usaha perusahaan',TRUE),
 ('m2.business_units.create','m2','Tambah Unit Usaha','Membuat unit usaha perusahaan',TRUE),
 ('m2.business_units.update','m2','Ubah Unit Usaha','Mengubah unit usaha perusahaan',TRUE),
 ('m2.categories.read','m2','Lihat Kategori','Melihat kategori unit usaha',TRUE),
 ('m2.categories.manage','m2','Kelola Kategori','Mengelola kategori unit usaha',TRUE),
 ('m2.divisions.read','m2','Lihat Divisi','Melihat divisi perusahaan',TRUE),
 ('m2.divisions.manage','m2','Kelola Divisi','Mengelola divisi perusahaan',TRUE),
 ('m2.dashboard.read','m2','Lihat Dashboard','Melihat dashboard perusahaan',TRUE),
 ('m4.products.read','m4','Lihat Produk','Melihat produk perusahaan',TRUE),
 ('m4.products.manage','m4','Kelola Produk','Mengelola barang dan jasa perusahaan',TRUE),
 ('m4.customers.manage','m4','Kelola Pelanggan','Mengelola pelanggan perusahaan',TRUE),
 ('m4.quotations.manage','m4','Kelola Quotation','Mengelola quotation perusahaan',TRUE),
 ('m4.invoices.manage','m4','Kelola Invoice','Mengelola invoice perusahaan',TRUE),
 ('m4.orders.manage','m4','Kelola Order','Mengelola order perusahaan',TRUE),
 ('m5.services.manage','m5','Kelola Permintaan Layanan','Memproses permintaan layanan perusahaan',TRUE)
ON CONFLICT (key) DO UPDATE SET
  module = EXCLUDED.module,
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  is_active = TRUE;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN (
 'm1.users.read','m1.users.create','m1.users.update','m1.users.delete',
 'm1.roles.read','m1.roles.manage','m1.audit.read',
 'm2.business_units.read','m2.business_units.create','m2.business_units.update',
 'm2.categories.read','m2.categories.manage',
 'm2.divisions.read','m2.divisions.manage','m2.dashboard.read',
 'm4.products.read','m4.products.manage','m4.customers.manage',
 'm4.quotations.manage','m4.invoices.manage','m4.orders.manage',
 'm5.services.manage'
)
WHERE r.name = 'partner' AND r.is_system = TRUE AND r.partner_id IS NULL
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Repair verified partner owner accounts that were linked before role assignment
-- became tenant-aware. Employees keep their own division/role assignments.
UPDATE users u
SET partner_id = p.id, division_id = NULL, updated_at = NOW()
FROM partners p
WHERE (p.user_id = u.id OR LOWER(p.email) = LOWER(u.email))
  AND p.status = 'terverifikasi'
  AND (p.user_id IS NULL OR p.user_id = u.id OR u.partner_id = p.id);

UPDATE partners p
SET user_id = u.id, updated_at = NOW()
FROM users u
WHERE LOWER(p.email) = LOWER(u.email)
  AND p.status = 'terverifikasi'
  AND u.partner_id = p.id
  AND p.user_id IS NULL;

DELETE FROM user_role_assignments ura
USING users u, partners p
WHERE ura.user_id = u.id
  AND p.status = 'terverifikasi'
  AND (p.user_id = u.id OR LOWER(p.email) = LOWER(u.email))
  AND u.partner_id = p.id;

INSERT INTO user_role_assignments (user_id, role_id, unit_usaha_id, created_at)
SELECT u.id, r.id, NULL, NOW()
FROM users u
JOIN partners p ON p.id = u.partner_id
JOIN roles r ON r.name = 'partner' AND r.is_system = TRUE AND r.partner_id IS NULL AND r.division_id IS NULL
WHERE (p.user_id = u.id OR LOWER(p.email) = LOWER(u.email))
  AND NOT EXISTS (
      SELECT 1 FROM user_role_assignments ura
      WHERE ura.user_id = u.id AND ura.role_id = r.id
  )
ON CONFLICT DO NOTHING;

-- Existing products inherit the tenant of their selected business unit.
-- Older rows created by platform-wide admins could have partner_id NULL even
-- though the selected business unit belonged to a partner.
UPDATE products prod
SET partner_id = bu.partner_id, updated_at = NOW()
FROM business_units bu
WHERE bu.id = prod.unit_usaha_id
  AND prod.partner_id IS DISTINCT FROM bu.partner_id;
