-- Rollback removes product classification. Existing inventory values remain.
DROP INDEX IF EXISTS idx_products_type_active_unit;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_product_type_check;
ALTER TABLE products DROP COLUMN IF EXISTS product_type;
