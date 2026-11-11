# Araya Business Hub Platform — Backend (Go + Gin)

Backend resmi untuk Araya Business Hub Platform (M1–M5), dibangun menggunakan **Go (Golang)** dan framework **Gin**, terintegrasi dengan **PostgreSQL (Supabase)**.

## Fitur Fondasi Utama
- **Standard API Envelope**: Seluruh respons mengikuti envelope `success: true, data: ...` dan `success: false, error: { code, message, fields }`.
- **Error Handling Terpusat**: Error codes stabil (`UPPER_SNAKE_CASE`) dengan pesan Bahasa Indonesia yang user-friendly.
- **Request ID & Structured Logging**: Setiap request dilengkapi header `X-Request-ID` dan logging otomatis dengan masking/redaction pada data kredensial (`password`, `jwt`, `secret`, `service_key`).
- **Environment Validation**: Konfigurasi divalidasi saat startup sebelum server menerima traffic.
- **Graceful Shutdown**: Server menangani sinyal `SIGINT` dan `SIGTERM` secara tertib dengan context timeout 10 detik.
- **Database Connection Pool**: Terkonfigurasi untuk PostgreSQL / Supabase pooler dengan ping health check.
- **Database Migrations**: Baseline skema M1–M5 dan forward migrations di `./migrations/`. Untuk Supabase yang sudah berisi tabel baseline, ikuti `docs/SUPABASE_MIGRATIONS.md` dan jalankan migration 000003–000011 satu per satu; jangan jalankan ulang baseline.
- **Workspace boundaries**: Super Admin hanya mengelola verifikasi/suspend Partner; setiap Partner mengelola organisasi dan transaksi tenant-nya sendiri. Lihat `docs/WORKSPACE_AND_MODULES.md`.

## Prasyarat
- Go 1.26.2 (sesuai deklarasi `go 1.26.2` pada `go.mod`)
- Database PostgreSQL (lokal atau instans Supabase)

## Menjalankan Backend Lokal

1. Salin konfigurasi environment:
   ```bash
   cp .env.example .env
   ```
2. Sesuaikan nilai `DATABASE_URL` dan konfigurasi rahasia pada file `.env`.
3. Jalankan server:
   ```bash
   go run ./cmd/api
   ```
4. Untuk database yang sudah ada, jalankan forward migrations sesuai `docs/SUPABASE_MIGRATIONS.md` sebelum menjalankan backend. Migration 000010 wajib karena versi backend ini membaca `users.is_partner_suspended`. Migration 000011 menambahkan hierarki Unit Usaha → Kategori → Divisi → Role → Pengguna.
5. Verifikasi endpoint health check:
   - Root probe: `http://localhost:8080/health`
   - Platform health envelope: `http://localhost:8080/api/v1/health`

## Menjalankan Unit Test
```bash
go test -v ./...
```


M4/M5 checkout and product kind changes: apply `migrations/000012_product_type.up.sql` after reviewing the migration guide. Guest checkout now creates an order with an unpaid invoice; customers can track by order number + checkout email and pay using the sandbox flow.
