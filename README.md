# Araya Business Hub Platform — Backend (Go + Gin)

Backend resmi untuk Araya Business Hub Platform (M1–M5), dibangun menggunakan **Go (Golang)** dan framework **Gin**, terintegrasi dengan **PostgreSQL (Supabase)**.

## Fitur Fondasi Utama
- **Standard API Envelope**: Seluruh respons mengikuti envelope `success: true, data: ...` dan `success: false, error: { code, message, fields }`.
- **Error Handling Terpusat**: Error codes stabil (`UPPER_SNAKE_CASE`) dengan pesan Bahasa Indonesia yang user-friendly.
- **Request ID & Structured Logging**: Setiap request dilengkapi header `X-Request-ID` dan logging otomatis dengan masking/redaction pada data kredensial (`password`, `jwt`, `secret`, `service_key`).
- **Environment Validation**: Konfigurasi divalidasi saat startup sebelum server menerima traffic.
- **Graceful Shutdown**: Server menangani sinyal `SIGINT` dan `SIGTERM` secara tertib dengan context timeout 10 detik.
- **Database Connection Pool**: Terkonfigurasi untuk PostgreSQL / Supabase pooler dengan ping health check.
- **Database Migrations**: Baseline skema M1–M5 di `./migrations/000001_init_schema.up.sql`.

## Prasyarat
- Go 1.22+ (teruji pada Go 1.26+)
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
4. Verifikasi endpoint health check:
   - Root probe: `http://localhost:8080/health`
   - Platform health envelope: `http://localhost:8080/api/v1/health`

## Menjalankan Unit Test
```bash
go test -v ./...
```
