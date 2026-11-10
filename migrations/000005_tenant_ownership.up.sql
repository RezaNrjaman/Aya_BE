-- ARY-001: Tenant ownership (partners are the tenants).
-- Adds partner_id to every business entity that lacked it, backfills what can
-- be derived unambiguously, and indexes the tenant column. Ambiguous rows are
-- left NULL and flagged, never assigned to an arbitrary tenant.

ALTER TABLE customers ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;

-- Backfill invoices: prefer the parent quotation tenant, else the tenant of
-- the invoice items' products (only when all items agree).
UPDATE invoices inv
SET partner_id = q.partner_id
FROM quotations q
WHERE inv.quotation_id = q.id AND inv.partner_id IS NULL;

UPDATE invoices inv
SET partner_id = (
  SELECT p.partner_id
  FROM invoice_items ii
  JOIN products p ON p.id = ii.product_id
  WHERE ii.invoice_id = inv.id AND p.partner_id IS NOT NULL
  GROUP BY p.partner_id
  HAVING COUNT(DISTINCT p.partner_id) = 1
)
WHERE inv.partner_id IS NULL;

-- Backfill orders from the tenant of the ordered products (only when all
-- items of the order belong to the same partner).
UPDATE orders o
SET partner_id = (
  SELECT p.partner_id
  FROM order_items oi
  JOIN products p ON p.id = oi.product_id
  WHERE oi.order_id = o.id AND p.partner_id IS NOT NULL
  GROUP BY p.partner_id
  HAVING COUNT(DISTINCT p.partner_id) = 1
)
WHERE o.partner_id IS NULL;

-- Backfill payments from their invoice.
UPDATE payments pay
SET partner_id = inv.partner_id
FROM invoices inv
WHERE pay.invoice_id = inv.id AND pay.partner_id IS NULL;

-- Backfill customers from quotations they appear on (only when that points to
-- exactly one tenant).
UPDATE customers c
SET partner_id = (
  SELECT q.partner_id
  FROM quotations q
  WHERE q.customer_id = c.id AND q.partner_id IS NOT NULL
  GROUP BY q.partner_id
  HAVING COUNT(DISTINCT q.partner_id) = 1
)
WHERE c.partner_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_customers_partner_id ON customers(partner_id);
CREATE INDEX IF NOT EXISTS idx_invoices_partner_id ON invoices(partner_id);
CREATE INDEX IF NOT EXISTS idx_orders_partner_id ON orders(partner_id);
CREATE INDEX IF NOT EXISTS idx_payments_partner_id ON payments(partner_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_partner_id ON audit_logs(partner_id);

-- Report ambiguous legacy rows (more than one candidate tenant) so operators
-- can resolve them explicitly instead of guessing.
-- (Informational only; safe to run as part of the migration.)
DO $$
DECLARE ambiguous INT;
BEGIN
  SELECT COUNT(*) INTO ambiguous FROM (
    SELECT oi.order_id
    FROM order_items oi
    JOIN products p ON p.id = oi.product_id
    WHERE p.partner_id IS NOT NULL
    GROUP BY oi.order_id
    HAVING COUNT(DISTINCT p.partner_id) > 1
  ) x;
  IF ambiguous > 0 THEN
    RAISE NOTICE 'ARY-001: % legacy orders have items spanning multiple partners and need manual tenant assignment', ambiguous;
  END IF;
END $$;
