# Status Revisi M4/M5 — 11 November 2026

## Perubahan yang diterapkan
- Login Partner dengan permission dashboard tetap menuju Dashboard Partner.
- Role selain `partner` tidak diarahkan otomatis ke Dashboard; sistem memilih menu pertama yang diizinkan tanpa menyertakan menu Dashboard.
- Pengguna yang mencoba workspace yang bukan miliknya diarahkan ke halaman akses terbatas, bukan Dashboard.
- Produk mempunyai klasifikasi tetap `barang` atau `jasa`.
- Barang memakai stok; jasa tidak menggunakan stok.
- Checkout guest membuat/mencocokkan customer pada tenant penjual.
- Checkout membuat Order dan Invoice `belum_lunas` serta item invoice secara atomik dengan reservasi stok barang di PostgreSQL.
- Idempotency key otomatis dibuat bila request tidak menyertakannya.
- Customer dapat melacak order dengan nomor order dan email checkout.
- Customer dapat membayar invoice melalui sandbox tanpa login.
- Partner hanya dapat memproses order setelah invoice lunas.
- Halaman sukses checkout menyediakan tautan langsung ke pelacakan pesanan.

## Database
Migration baru: `migrations/000012_product_type.up.sql`.
Migration menambahkan `products.product_type` dengan default `barang`, constraint `barang|jasa`, index produk, dan index lower(email) untuk lookup guest.
Schema terbaru yang dikirim sudah mempunyai `orders.invoice_id`, `orders.customer_id`, `customers.email`, `invoice_items`, dan `order_status_histories`; karena itu fitur tracking tidak memerlukan kolom tambahan.

Jika migration 000011 sudah diterapkan di Supabase, cukup jalankan migration 000012 sebelum deploy backend ini. Backup dan uji di staging terlebih dahulu.

## Endpoint guest baru
- `POST /api/m4/v1/public/orders/track` — menerima order_no dan customer_email.
- `POST /api/m4/v1/public/orders/:order_no/pay` — menerima customer_email dan metode sandbox. Nominal tagihan selalu dibaca server-side.
Keduanya memakai rate limit.

## Batasan yang harus diketahui
- Pembayaran masih sandbox, bukan payment gateway produksi.
- Tracking dengan nomor order + email adalah verifikasi dasar; sebelum produksi disarankan menambah OTP atau magic link email.
- Satu order hanya dapat berisi produk dari satu Partner. Produk lintas Partner harus checkout terpisah.
- Belum ada refund otomatis.
- Customer tercatat saat checkout membuat order; data customer digunakan ulang berdasarkan email di tenant yang sama.

## Pemeriksaan
- TypeScript `tsc --noEmit -p tsconfig.app.json`: lulus.
- `gofmt`: lulus, semua file Go terformat.
- Vite production build: belum dapat dijalankan karena optional package `@typescript/typescript-linux-x64` tidak tersedia pada arsip node_modules.
- Go tests: belum dapat dijalankan karena runtime tersedia Go 1.23.2, proyek meminta Go 1.26.2, dan dependency belum dapat diunduh dari proxy.golang.org.
- PostgreSQL/Supabase integration test: belum dijalankan.

Source ini merupakan revisi implementasi, bukan klaim bebas bug atau siap produksi. Jalankan test BE dan build FE pada mesin pengembangan dengan dependency lengkap serta database staging.
