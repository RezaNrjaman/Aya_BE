-- 000012_product_type.up.sql
-- Products are explicitly classified as physical goods or services.
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS product_type VARCHAR(16) NOT NULL DEFAULT 'barang';

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_product_type_check;

ALTER TABLE products
    ADD CONSTRAINT products_product_type_check
    CHECK (product_type IN ('barang', 'jasa'));

CREATE INDEX IF NOT EXISTS idx_products_type_active_unit
    ON products(unit_usaha_id, product_type, is_active);

CREATE INDEX IF NOT EXISTS idx_customers_email_lower ON customers (LOWER(email));
