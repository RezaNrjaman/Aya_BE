# Status Revisi Araya Business Hub — 11 November 2026

## Perubahan yang diterapkan pada source tree ini

### Workspace dan otorisasi
- FE memisahkan navigasi Super Admin (Dashboard Platform + Verifikasi Partner) dari navigasi Partner (Dashboard, Manajemen, Transaksi, Portal, Audit Log).
- Route FE operasional dibatasi ke workspace Partner; halaman Partner Management dibatasi ke workspace Platform.
- Backend menambahkan middleware `RequirePlatformAdmin` untuk endpoint governance M3.
- Backend menambahkan middleware `RequirePartnerTenant` untuk API user/role/audit M1, organisasi/dashboard M2, transaksi privat M4, dan inbox layanan M5.
- Route pembuatan order back-office dipisah ke `/api/m4/v1/partner/orders`; kontrak guest `/api/m4/v1/orders` dipertahankan untuk kompatibilitas customer/checkout.
- Referensi dashboard platform yang salah pada `nav-items.ts` diperbaiki.

### Organisasi dan Partner
- Halaman Divisi memiliki tombol kembali ke Dashboard, tautan "Lewati untuk sekarang" saat belum ada divisi, pembatalan modal, tombol tutup, klik backdrop, dan Escape.
- Divisi tetap opsional; form unit usaha telah menyediakan pilihan "Tanpa divisi".
- Halaman Super Admin menyediakan tautan yang jelas ke registrasi publik mitra; pendaftaran dilakukan di `/partner/register`, bukan dibuat dengan password manual oleh Super Admin.
- Penangguhan Partner sekarang menonaktifkan semua akun user yang terhubung dengan tenant, bukan hanya owner.
- Reaktivasi mengaktifkan kembali akun yang dinonaktifkan oleh suspend, tetapi tidak mengaktifkan akun yang dinonaktifkan manual.
- Query list user PostgreSQL diperbaiki agar jumlah kolom SELECT dan Scan sesuai (termasuk activation token dan status disable).

### M5 dan dokumentasi
- Endpoint publik permintaan layanan dan checkout diberi rate limit dasar.
- Ditambahkan `docs/WORKSPACE_AND_MODULES.md` dan `docs/SUPABASE_MIGRATIONS.md`.
- Ditambahkan migration forward `000010_partner_member_suspension.up.sql` untuk kolom `users.is_partner_suspended`.

## Migration Supabase

Untuk database yang sudah memiliki baseline, backup dulu lalu jalankan berurutan:
`000003_partner_members.up.sql` sampai `000010_partner_member_suspension.up.sql`.
Jangan jalankan ulang `000001_init_schema.up.sql` pada database existing. Detail ada di `docs/SUPABASE_MIGRATIONS.md`.

## Yang sudah ada dari source sebelumnya dan dipertahankan

- Registrasi publik Partner dan cek status menggunakan registration code.
- Token aktivasi owner sekali pakai dan tanpa password universal.
- Struktur role custom per partner dengan prefix tenant dan katalog permission.
- Kepemilikan partner pada unit usaha, divisi, produk, quotation, customer, invoice, order, dan audit (sesuai migration yang tersedia).
- Dashboard operasional memiliki provider agregasi dari repository M4 yang di-wire pada `cmd/api/main.go`.
- Cart guest dan checkout tanpa `customer_id` pada FE; server membuat/mencocokkan customer berdasarkan detail kontak dan menolak campuran seller dalam satu order.
- Katalog dan service request memeriksa status Partner yang boleh tampil publik.

## Belum boleh dianggap selesai / perlu implementasi dan verifikasi lanjutan

1. Undangan aktivasi staf Partner: form tambah pengguna masih meminta password dari admin, belum mengeluarkan invitation token sekali pakai.
2. Pembayaran publik invoice berbasis token guest belum tersedia; pembayaran sandbox masih perlu verifikasi callback lengkap, replay protection, dan transaksi atomik payments+invoices.
3. Pengurangan stok atomik belum dibuktikan/diimplementasikan pada commit order; model produk fisik vs jasa (tanpa stok) belum lengkap.
4. Pembuatan invoice/quotation beserta header, items, dan penomoran belum dibuktikan atomik end-to-end.
5. Idempotency checkout/order perlu validasi bahwa key yang sama tidak dapat dipakai ulang dengan payload berbeda.
6. Pengiriman email aktivasi belum terintegrasi; token aktivasi status page masih mekanisme sandbox.
7. Audit log belum menyimpan seluruh metadata actor/request secara lengkap; filter tenant sudah tersedia pada repository.
8. Pengujian integration/E2E lintas tenant, suspend pada sesi aktif, callback palsu/duplikat, dan Supabase staging masih wajib dilakukan.

## Hasil pemeriksaan lingkungan saat revisi

- `gofmt` berhasil untuk file Go yang diubah.
- Backend test belum dapat dijalankan: lingkungan hanya memiliki Go 1.23.2, sedangkan `go.mod` mensyaratkan Go 1.26.2. Percobaan lokal dengan versi 1.23 juga tidak dapat mengunduh dependency karena akses `proxy.golang.org` tidak tersedia.
- Frontend TypeScript check (`tsc --noEmit -p tsconfig.app.json`) berhasil setelah memperbaiki duplikasi `DashboardPage` dan menambahkan komponen `PartnerDashboardPage` yang sebelumnya direferensikan tetapi belum didefinisikan.
- Vite build, Vitest, dan Oxlint belum dapat dijalankan sampai selesai karena arsip `node_modules` yang tersedia tidak memiliki optional native binding Rolldown/Oxlint untuk Linux. `npm ci` tidak selesai karena akses registry tidak tersedia; jalankan ulang `npm ci` pada mesin pengembangan dengan akses internet sebelum build/test.
- PostgreSQL/psql tidak tersedia di lingkungan ini; migration dan query runtime belum dieksekusi.

**Kesimpulan:** ZIP ini berisi revisi source dan migration yang dapat ditinjau, tetapi belum merupakan rilis teruji, bebas bug, atau siap produksi. Jalankan migration di staging, install dependency dengan akses internet, lalu jalankan seluruh test dan E2E sebelum deployment.


---

# Revisi Modul Manajemen — Dropdown, Profil Unit, dan Hak Akses

## Perubahan
1. Nomor urutan di label sidebar Manajemen dihapus.
2. Form dan tabel Unit Usaha hanya menampilkan nama dan deskripsi (status dan aksi tetap tersedia sebagai kontrol operasional). Kolom kontak/alamat lama di database tidak dihapus agar data lama tetap aman.
3. Semua dropdown organisasi menggunakan `SearchableSelect` dengan pencarian langsung.
4. Role dan Pengguna memuat Unit Usaha, Kategori, dan Divisi secara berhalaman (maksimal 100 item per request) hingga seluruh halaman selesai. Ini memperbaiki bug lama: frontend meminta 200–1000 item padahal backend menolak `per_page` di atas 100, sehingga request gagal dan dropdown menjadi kosong.
5. Kategori disaring berdasarkan `unit_usaha_id`; Divisi disaring berdasarkan unit dan kategori; Role disaring berdasarkan divisi. Mengganti induk mengosongkan pilihan anak.
6. Pembuatan role mewajibkan minimal satu permission eksplisit. Opsi/placeholder “Mulai tanpa hak akses” dihapus; permission dipilih dari katalog yang boleh didelegasikan akun Partner. Backend juga menolak role Partner dengan permission kosong.

## Kesesuaian dengan schema Supabase terbaru
Schema yang diberikan sudah memiliki kolom relasi yang dipakai: `business_categories.unit_usaha_id`, `divisions.unit_usaha_id`, `divisions.category_id`, `roles.division_id`, serta `user_role_assignments.role_id` dan `unit_usaha_id`. `users` menyimpan `division_id` dan role pengguna tetap disimpan pada `user_role_assignments`.

Tidak ada migration database baru untuk revisi ini. Kolom kontak/alamat pada `business_units` dibiarkan sebagai kolom legacy/opsional dan tidak lagi diminta oleh UI.

## Verifikasi
- Frontend TypeScript check: lulus (`tsc --noEmit -p tsconfig.app.json`).
- Go formatting: lulus (`gofmt`).
- Go unit/integration test belum dapat dijalankan karena dependency belum tersedia lokal dan akses `proxy.golang.org` tidak tersedia pada lingkungan pemeriksaan.
- Vite production build perlu dijalankan di mesin pengembangan setelah dependency di-install ulang.
