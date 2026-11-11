# Modul Manajemen — Hierarki Organisasi Partner

## Struktur wajib
Partner -> Unit Usaha -> Kategori -> Divisi -> Role -> Pengguna.

- Satu Partner memiliki banyak unit usaha.
- Satu unit usaha memiliki banyak kategori.
- Satu kategori memiliki banyak divisi.
- Satu divisi memiliki banyak role.
- Satu role dapat diberikan kepada banyak pengguna.
- Setiap pengguna operasional memiliki satu unit usaha, satu divisi, dan satu role aktif.
- Divisi dan role harus berasal dari rantai induk yang sama dan Partner yang sama.
- Permission dipilih dari katalog permission platform dan dipetakan pada role; tidak dibuat bebas oleh Partner.
- Super Admin hanya mengelola tata kelola Partner, bukan struktur organisasi tenant.

## Implementasi backend dan database
Migration `000011_management_hierarchy.up.sql` menambahkan:
- `business_categories.partner_id`, `unit_usaha_id`, `is_active`, `updated_at`.
- `divisions.unit_usaha_id` dan `category_id`.
- `roles.division_id`.
- Backfill kategori lama yang terhubung ke unit usaha.
- Backfill divisi dan role lama hanya jika relasinya dapat ditentukan secara tidak ambigu.
- Index per induk dan trigger PostgreSQL untuk mencegah relasi lintas Partner / unit / kategori / divisi.
- Trigger assignment untuk memastikan role pengguna berasal dari divisi yang sama dan unit assignment cocok dengan unit divisi role.

Data lama yang relasinya ambigu tidak dipaksakan masuk ke rantai baru. Data tersebut harus direkonsiliasi manual oleh pemilik tenant sebelum dipakai sebagai role/pengguna operasional.

## Alur UI
1. Unit Usaha dibuat langsung oleh Partner.
2. Kategori dibuat dengan memilih Unit Usaha.
3. Divisi dibuat dengan memilih Unit Usaha, lalu kategori yang berada di unit tersebut.
4. Role dibuat dengan memilih Unit Usaha, kategori, lalu divisi. Role hanya berlaku pada divisi itu.
5. Pengguna dibuat dengan memilih Unit Usaha, kategori, divisi, lalu role yang berasal dari divisi tersebut.

Pilihan turunan otomatis dikosongkan ketika pilihan induk diganti, untuk mencegah ID anak lama terkirim bersama induk baru.

## Aturan operasional
- Kategori/divisi/role yang sudah dipakai sebaiknya dinonaktifkan, bukan dihapus.
- Unit usaha/kategori/divisi tidak boleh dihapus jika masih memiliki anak atau relasi transaksi/pengguna.
- Pindah pengguna ke divisi lain harus disertai role baru dari divisi tujuan.
- Partner tidak dapat memilih ID milik Partner lain; backend mengambil `partner_id` dari token autentikasi dan memvalidasi seluruh rantai di database.
- Role sistem owner `partner` hanya digunakan untuk bootstrap pemilik tenant sebelum struktur organisasi pertama dibuat. Role itu bukan role operasional lintas divisi.

## Migration Supabase
Pada database yang telah menjalankan migration 000003–000010, jalankan:
`000011_management_hierarchy.up.sql`

Backup database terlebih dahulu. Jalankan pada staging sebelum production. Jangan menjalankan file `.down.sql` untuk instalasi normal.

Sesudah migration, cek bahwa:
- `business_categories` mempunyai `partner_id`, `unit_usaha_id`, `is_active`, `updated_at`.
- `divisions` mempunyai `partner_id`, `unit_usaha_id`, `category_id`.
- `roles` mempunyai `partner_id`, `is_system`, `division_id`.

Migration menjaga data lama. Kategori/divisi/role yang tidak dapat di-backfill secara unik dapat tetap belum terpetakan; review sebelum dipakai.

## Catatan pengujian
Frontend cascading selector telah diperjelas pada form Role dan Pengguna. TypeScript/build dan PostgreSQL integration test tetap harus dijalankan di lingkungan proyek dengan dependency dan database staging yang sesuai sebelum dinyatakan siap produksi.
