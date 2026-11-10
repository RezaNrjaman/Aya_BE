-- ARY-001 rollback: drop tenant ownership columns added in the up migration.
DROP INDEX IF EXISTS idx_audit_logs_partner_id;
DROP INDEX IF EXISTS idx_payments_partner_id;
DROP INDEX IF EXISTS idx_orders_partner_id;
DROP INDEX IF EXISTS idx_invoices_partner_id;
DROP INDEX IF EXISTS idx_customers_partner_id;

ALTER TABLE audit_logs DROP COLUMN IF EXISTS partner_id;
ALTER TABLE payments DROP COLUMN IF EXISTS partner_id;
ALTER TABLE orders DROP COLUMN IF EXISTS partner_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS partner_id;
ALTER TABLE customers DROP COLUMN IF EXISTS partner_id;
