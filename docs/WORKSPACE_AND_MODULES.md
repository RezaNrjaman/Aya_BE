# Araya Business Hub — Model Workspace dan Cakupan Modul

Tanggal revisi: 11 November 2026

## 1. Prinsip workspace

### Super Admin (platform)
Super Admin hanya mengelola tata kelola mitra:
- Melihat ringkasan pendaftaran/status mitra.
- Menyetujui atau menolak pendaftaran beserta alasan.
- Menangguhkan mitra beserta alasan.
- Mengaktifkan kembali mitra.
- Melihat riwayat perubahan status mitra.

Super Admin tidak mengelola role, divisi, unit usaha, pengguna staf, produk, pelanggan, quotation, invoice, order, atau service request milik perusahaan.

Role lama `admin` dipertahankan sebagai nama teknis Super Admin agar kompatibel dengan akun dan data lama. Role `super_admin` juga dikenali oleh workspace guard, tetapi seed/migrasi saat ini menggunakan `admin`.

### Partner (tenant)
Setiap partner adalah tenant terpisah. Partner mengelola:
- Dashboard perusahaan.
- Pengguna dan penempatan divisi/unit.
- Role custom dan permission yang didelegasikan dari katalog platform.
- Divisi dan unit usaha.
- Produk/jasa dan pelanggan.
- Quotation, invoice, pembayaran, dan order.
- Inbox permintaan layanan.
- Audit log tenant.

Partner tidak boleh mengakses data partner lain. `partner_id` diambil dari identitas terautentikasi, bukan dari request body.

### Customer (guest)
Customer publik tidak memakai akun internal Partner. Customer dapat melihat katalog publik, membuat permintaan layanan, memakai cart, dan checkout. Checkout menolak cart yang berisi produk dari penjual/partner berbeda; customer perlu checkout terpisah.

## 2. Cakupan modul

- **M1 — Authentication, Users & RBAC:** login, profil, ganti password, user tenant, role custom, permission matrix, audit log tenant.
- **M2 — Organization & Dashboard:** profil/unit usaha, divisi, status unit, ringkasan operasional dan filter unit/periode.
- **M3 — Partner Governance:** registrasi publik, cek status, verifikasi, penolakan, suspend, reaktivasi. Endpoint administrasi M3 hanya untuk Super Admin.
- **M4 — Transaction Management:** produk/jasa, pelanggan, quotation, invoice, pembayaran sandbox, order dan riwayat status.
- **M5 — Customer Portal:** katalog publik, pencarian/filter, permintaan layanan, cart guest, checkout ke kontrak order M4.

## 3. Perilaku navigasi divisi

Divisi bersifat opsional. Pengguna boleh membuka halaman Divisi, menambahkan atau tidak menambahkan divisi, membatalkan modal, kembali ke Dashboard, dan tetap membuat unit usaha dengan pilihan `Tanpa divisi`.

## 4. Registrasi mitra

Pendaftaran dilakukan melalui halaman publik `/partner/register`. Super Admin tidak membuat password mitra secara manual. Pengajuan masuk dengan status `pending`, lalu diproses dari halaman `/partners`. Setelah disetujui, owner mengaktifkan akun menggunakan token aktivasi. Pada konfigurasi saat ini token ditampilkan melalui halaman status sebagai mekanisme sandbox; integrasi email aktivasi otomatis masih diperlukan sebelum produksi.

## 5. Batasan dan pekerjaan lanjutan

Perubahan workspace di frontend adalah UX; otorisasi backend tetap menjadi pengaman utama. Endpoint M1 user/role/audit, M2, M4 operasional, dan M5 inbox kini memerlukan principal tenant dengan `partner_id`. Endpoint M3 governance memerlukan Super Admin.

Pekerjaan yang masih perlu dituntaskan dan diuji sebelum produksi:
- Invitation aktivasi staf Partner (form user saat ini masih menggunakan password awal oleh admin).
- Pembayaran publik berbasis token invoice, verifikasi callback, replay protection, dan transaksi atomik payment+invoice.
- Pengurangan stok atomik ketika order berhasil dan pemodelan produk fisik vs jasa.
- Transaksi atomik penuh pada pembuatan invoice/quotation beserta item dan nomor dokumen.
- Rate limit, anti-spam, retensi, dan pengelolaan data pribadi customer secara menyeluruh.
- Integration/E2E test terhadap PostgreSQL/Supabase dan skenario multi-tenant.
- Email aktivasi dan notifikasi status yang terverifikasi.

Dokumen ini tidak menyatakan aplikasi bebas bug atau siap produksi. Semua fitur harus diverifikasi di CI dan database staging sebelum deploy.


## M4/M5 — E-commerce guest checkout
- Produk mempunyai jenis tetap `barang` atau `jasa`.
- Barang menggunakan stok; jasa tidak menggunakan stok.
- Checkout guest membuat/mencocokkan customer berdasarkan email pada tenant produk.
- Checkout membuat order dan invoice awal `belum_lunas`.
- Stok barang direservasi/dikurangi secara atomik dalam transaksi PostgreSQL saat order dibuat; jasa tidak mengurangi stok.
- Customer guest dapat melacak order menggunakan nomor order + email checkout.
- Customer guest dapat membayar invoice melalui endpoint sandbox publik yang memverifikasi email order dan dibatasi rate limit.
- Endpoint tracking/payment publik hanya menampilkan order yang cocok dengan nomor order dan email; email bukan pengganti autentikasi kuat untuk data sensitif.
- Migration 000012 menambahkan `products.product_type` dengan default `barang`.
