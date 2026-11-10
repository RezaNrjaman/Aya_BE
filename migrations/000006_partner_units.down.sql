-- ARY-005 rollback: detach business units from partner tenants.

DROP INDEX IF EXISTS idx_business_units_partner_id;
ALTER TABLE business_units DROP COLUMN IF EXISTS partner_id;
