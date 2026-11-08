-- ==============================================================================
-- Araya Business Hub Platform — Baseline Database Migration (000001_init_schema.up.sql)
-- Target Platform: PostgreSQL via Supabase
-- Modules Covered: M1, M2, M3, M4, M5, and Shared Platform
-- ==============================================================================

-- Enable UUID extension if not already available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==============================================================================
-- 1. M1 — Authentication, Dynamic RBAC, User Management, and Audit Log
-- ==============================================================================

CREATE TABLE IF NOT EXISTS divisions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE, -- e.g. "m1.users.read", "m2.business_units.update"
    module VARCHAR(20) NOT NULL,      -- "m1", "m2", "m3", "m4", "m5"
    name VARCHAR(150) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(30),
    division_id INT REFERENCES divisions(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User Role Assignments with scoping (division / business unit)
CREATE TABLE IF NOT EXISTS user_role_assignments (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    unit_usaha_id INT, -- scoped to business unit if applicable
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id, unit_usaha_id)
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE SET NULL,
    user_name VARCHAR(150),
    action VARCHAR(100) NOT NULL,       -- e.g. "user.created", "permission.assigned"
    entity_type VARCHAR(100) NOT NULL,  -- e.g. "user", "role", "business_unit"
    entity_id VARCHAR(100),
    details JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- ==============================================================================
-- 2. M2 — Business Unit Management
-- ==============================================================================

CREATE TABLE IF NOT EXISTS business_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS business_units (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    category_id INT REFERENCES business_categories(id) ON DELETE SET NULL,
    division_id INT REFERENCES divisions(id) ON DELETE SET NULL,
    description TEXT,
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    address TEXT,
    logo_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_business_units_active ON business_units(is_active);

-- ==============================================================================
-- 3. M3 — Partner Management
-- ==============================================================================

CREATE TABLE IF NOT EXISTS partners (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE SET NULL,
    company_name VARCHAR(200) NOT NULL,
    contact_person VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(50) NOT NULL,
    address TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending', -- pending, terverifikasi, ditolak, ditangguhkan
    rejection_reason TEXT,
    suspension_reason TEXT,
    verified_at TIMESTAMPTZ,
    verified_by INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS partner_status_histories (
    id SERIAL PRIMARY KEY,
    partner_id INT NOT NULL REFERENCES partners(id) ON DELETE CASCADE,
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    reason TEXT,
    changed_by INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_partners_status ON partners(status);

-- ==============================================================================
-- 4. M4 — Transaction Management (Product, Customer, Quotation, Invoice, Order)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    unit_usaha_id INT NOT NULL REFERENCES business_units(id) ON DELETE RESTRICT,
    partner_id INT REFERENCES partners(id) ON DELETE SET NULL,
    name VARCHAR(200) NOT NULL,
    category VARCHAR(100),
    description TEXT,
    price BIGINT NOT NULL, -- Integer Rupiah
    stock INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_unit_usaha ON products(unit_usaha_id);
CREATE INDEX IF NOT EXISTS idx_products_active ON products(is_active);

CREATE TABLE IF NOT EXISTS quotations (
    id SERIAL PRIMARY KEY,
    quotation_no VARCHAR(50) NOT NULL UNIQUE,
    unit_usaha_id INT NOT NULL REFERENCES business_units(id) ON DELETE RESTRICT,
    partner_id INT REFERENCES partners(id) ON DELETE SET NULL,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    total_amount BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'draft', -- draft, diajukan, disetujui, ditolak
    valid_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quotation_items (
    id SERIAL PRIMARY KEY,
    quotation_id INT NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
    product_id INT REFERENCES products(id) ON DELETE RESTRICT,
    item_name VARCHAR(200) NOT NULL,
    quantity INT NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS invoices (
    id SERIAL PRIMARY KEY,
    invoice_no VARCHAR(50) NOT NULL UNIQUE,
    quotation_id INT REFERENCES quotations(id) ON DELETE SET NULL,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    total_amount BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'belum_lunas', -- belum_lunas, lunas, dibatalkan
    due_date DATE,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invoice_items (
    id SERIAL PRIMARY KEY,
    invoice_id INT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id INT REFERENCES products(id) ON DELETE RESTRICT,
    item_name VARCHAR(200) NOT NULL,
    quantity INT NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    payment_reference VARCHAR(100) NOT NULL UNIQUE,
    invoice_id INT NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,
    method VARCHAR(50) NOT NULL,      -- e.g. "sandbox_qris", "sandbox_va"
    status VARCHAR(30) NOT NULL,      -- "pending", "success", "failed"
    gateway_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    order_no VARCHAR(50) NOT NULL UNIQUE,
    customer_id INT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    invoice_id INT REFERENCES invoices(id) ON DELETE SET NULL,
    total_amount BIGINT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'baru', -- baru, diproses, selesai, dibatalkan
    idempotency_key VARCHAR(100) UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    item_name VARCHAR(200) NOT NULL,
    quantity INT NOT NULL,
    unit_price BIGINT NOT NULL,
    subtotal BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_status_histories (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    notes TEXT,
    changed_by INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders(customer_id);

-- ==============================================================================
-- 5. M5 — Customer Portal (Service Requests, Carts)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS service_requests (
    id SERIAL PRIMARY KEY,
    unit_usaha_id INT NOT NULL REFERENCES business_units(id) ON DELETE RESTRICT,
    customer_name VARCHAR(150) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(50),
    service_type VARCHAR(100),
    description TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending', -- pending, diproses, selesai
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS carts (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(100) NOT NULL UNIQUE,
    customer_id INT REFERENCES customers(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INT NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(cart_id, product_id)
);

-- ==============================================================================
-- 6. Shared Platform — Event Inbox for Idempotent Cross-Module Events
-- ==============================================================================

CREATE TABLE IF NOT EXISTS event_inbox (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    source_module VARCHAR(20) NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
