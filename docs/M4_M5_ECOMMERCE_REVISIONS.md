# Revisi Modul 4 & 5 — Transaksi dan Customer Portal

## Analisis terhadap 21 User Story capstone

Ruang lingkup dokumen sumber:
- M4-US1: input produk/jasa dan pelanggan.
- M4-US2: quotation, persetujuan quotation, dan invoice.
- M4-US3: pembayaran invoice melalui gateway sandbox.
- M4-US4: perubahan status order diproses/selesai.
- M5-US1: katalog seluruh unit usaha aktif.
- M5-US2: permintaan layanan.
- M5-US3: keranjang belanja dan quantity.
- M5-US4: checkout keranjang menjadi order melalui kontrak M4.
- M5-US5: pencarian/filter katalog.

## Perilaku e-commerce yang ditambahkan

1. Produk wajib mempunyai `product_type`: `barang` atau `jasa`.
2. Barang mempunyai stok dan stok direservasi/dikurangi secara atomik pada transaksi checkout.
3. Jasa tidak menggunakan stok; kolom stok disimpan 0 dan tidak memblokir checkout.
4. Guest checkout menerima nama dan email, lalu membuat atau menggunakan ulang record `customers` berdasarkan email dalam tenant partner penjual.
5. Checkout membuat order dan invoice `belum_lunas` beserta invoice items dalam transaksi database yang sama dengan reservasi stok.
6. Customer dapat melacak pesanan menggunakan nomor order + email checkout. Endpoint tidak memerlukan akun customer.
7. Customer dapat membayar invoice melalui sandbox dari halaman tracking. Endpoint mencari invoice dari order yang cocok dengan nomor order dan email; nominal tidak diterima dari browser tetapi dibaca dari invoice.
8. Order baru hanya dapat diproses partner setelah invoice lunas.
9. Customer dapat melihat riwayat status order dari halaman tracking.
10. Produk dari partner berbeda masih harus di-checkout terpisah; satu order tidak boleh memiliki lebih dari satu tenant.

## API publik baru

- `POST /api/m4/v1/public/orders/track`
  Body: `{ "order_no": "ORD-...", "customer_email": "buyer@example.com" }`
  Response: order, item, status history, invoice dan status invoice.

- `POST /api/m4/v1/public/orders/:order_no/pay`
  Body: `{ "customer_email": "buyer@example.com", "method": "sandbox_qris" }`
  Nominal dibaca server-side dari invoice. Endpoint dilindungi rate limit.

## Database

Migration baru: `migrations/000012_product_type.up.sql`
- Menambahkan `products.product_type` dengan nilai `barang` / `jasa`.
- Data produk lama otomatis default `barang`.
- Menambahkan index lower(email) untuk pencarian guest tracking.

Tidak diperlukan kolom customer/order/invoice baru untuk guest tracking karena skema sudah memiliki `orders.customer_id`, `orders.invoice_id`, dan `customers.email`.

## Data customer

Pada checkout, service M4 membuat customer jika email tersebut belum terdaftar untuk partner penjual. Jika sudah ada, record customer digunakan ulang. Karena itu customer akan muncul di tabel Pelanggan secara otomatis setelah order berhasil dibuat.

## Batasan dan catatan keamanan

- Pembayaran yang disediakan masih sandbox, bukan gateway produksi.
- Pelacakan menggunakan kombinasi nomor order + email dan rate limit; sebelum produksi sebaiknya tambahkan OTP/link verifikasi email untuk mengurangi risiko jika email dan nomor order diketahui pihak lain.
- Keranjang hanya dapat checkout satu partner pada satu order; pembelian dari beberapa partner perlu checkout terpisah.
- Belum ada fitur refund otomatis. Order yang sudah dibayar tidak boleh dibatalkan melalui alur pembatalan biasa tanpa proses refund.
- Migrasi 000012 wajib dijalankan sebelum deploy backend karena query produk membaca kolom `product_type`.
