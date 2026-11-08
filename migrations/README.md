# Database Migrations — Araya Business Hub Platform

Folder ini berisi file migrasi skema database PostgreSQL (Supabase) yang dapat dilacak (*version-controlled*) dan dijalankan ulang (*deterministic*).

## Struktur File
- `000001_init_schema.up.sql`: Menyiapkan skema baseline untuk M1 (User, Role, Permission, Audit), M2 (Business Unit), M3 (Partner), M4 (Transaction, Quotation, Invoice, Order), M5 (Service Request, Cart), dan Shared (Event Inbox).
- `000001_init_schema.down.sql`: Rollback untuk skema baseline.

## Menjalankan Migrasi

### Menggunakan CLI Supabase (Direkomendasikan)
```bash
supabase db push
```

### Menggunakan golang-migrate CLI
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

### Rollback
```bash
migrate -path ./migrations -database "$DATABASE_URL" down 1
```
