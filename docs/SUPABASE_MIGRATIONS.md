# Supabase — Urutan Forward Migration Araya Business Hub

Database Supabase yang sudah memiliki tabel baseline (`divisions`, `roles`, `permissions`, `users`, `audit_logs`, `partners`, `business_units`, `customers`, `products`, `quotations`, `invoices`, `payments`, `orders`, `service_requests`, `carts`) **jangan di-reset**.

Pastikan backup terlebih dahulu. Jalankan SQL dari `BE/migrations` melalui Supabase SQL Editor, satu file `.up.sql` pada satu waktu, sesuai urutan:

1. `000003_partner_members.up.sql`
2. `000004_partner_activation.up.sql`
3. `000005_tenant_ownership.up.sql`
4. `000006_partner_units.up.sql`
5. `000007_partner_divisions.up.sql`
6. `000008_partner_roles.up.sql`
7. `000009_access_boundaries.up.sql`
8. `000010_partner_member_suspension.up.sql`
9. `000011_management_hierarchy.up.sql`
10. `000012_product_type.up.sql`

Jangan menjalankan `000001_init_schema.up.sql` pada database yang tabel baseline-nya sudah ada. `000002_seed_initial_data.sql` bukan migration struktur; periksa data seed terlebih dahulu dan jangan menjalankannya berulang tanpa idempotency check. Jangan jalankan file `.down.sql` pada instalasi normal.

## Catatan penting

- Migration 000005 melakukan backfill `partner_id` hanya dari relasi yang dapat ditentukan secara jelas; periksa baris lama yang tetap NULL sebelum mengaktifkan multi-tenant.
- Migration 000008 menggunakan prefix role partner (`p{id}_...`) agar nama role tetap global-unambiguous walaupun nama tampilan boleh sama antartenant.
- Migration 000009 membatasi role `admin` ke izin tata kelola M3 dan memberi role `partner` izin operasional tenant.
- Migration 000010 menambahkan `users.is_partner_suspended`, yang dibaca oleh backend baru. Migration ini wajib dijalankan sebelum deploy backend revisi.
- `000011_management_hierarchy.up.sql` menambahkan relasi unit→kategori→divisi→role dan trigger untuk memastikan role dan user berada pada divisi serta tenant yang sama. Kategori/divisi legacy yang tidak dapat dipetakan secara pasti dibiarkan tanpa induk dan harus direkonsiliasi manual.
- `000010_partner_member_suspension.down.sql` menghapus marker suspensi. Jangan rollback kolom tersebut sebelum rollback versi backend dan meninjau akun yang sedang ditangguhkan.
- Jangan menambahkan `partner_id` ke baris lama yang ambigu secara manual tanpa rekonsiliasi bisnis.

## Pemeriksaan setelah migration

```sql
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = 'public'
  AND (
    (table_name = 'users' AND column_name IN (
      'partner_id', 'activation_token_hash', 'activation_expires_at',
      'is_manually_disabled', 'is_partner_suspended'
    ))
    OR (table_name = 'partners' AND column_name = 'registration_code')
    OR (table_name = 'business_units' AND column_name = 'partner_id')
    OR (table_name = 'divisions' AND column_name IN ('partner_id', 'is_active', 'updated_at'))
    OR (table_name = 'roles' AND column_name IN ('partner_id', 'is_system'))
    OR (table_name IN ('customers', 'invoices', 'orders', 'payments', 'audit_logs')
        AND column_name = 'partner_id')
  )
ORDER BY table_name, column_name;
```

Periksa role dan permission:

```sql
SELECT r.name, r.partner_id, r.is_system, p.key
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
WHERE r.name IN ('admin', 'partner')
ORDER BY r.name, p.key;
```

Role `admin` semestinya hanya memiliki permission `m3.partners.*`. Role `partner` memegang permission operasional tenant. Role custom milik partner memiliki `partner_id` sesuai tenant.

## Catatan implementasi

File migration ini adalah SQL forward migration untuk skema proyek. Terapkan terlebih dahulu pada staging/backup copy. Keberhasilan eksekusi SQL belum menggantikan integration test terhadap data aktual Supabase.
