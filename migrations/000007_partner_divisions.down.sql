-- ARY-006 rollback: detach divisions from partner tenants. Existing rows are
-- kept; the per-tenant name index is replaced by the original global unique
-- constraint. Duplicate names across tenants must be resolved manually before
-- this rollback can succeed.

DROP INDEX IF EXISTS idx_divisions_partner_name;
DROP INDEX IF EXISTS idx_divisions_partner_id;
ALTER TABLE divisions DROP CONSTRAINT IF EXISTS divisions_name_key;
ALTER TABLE divisions ADD CONSTRAINT divisions_name_key UNIQUE (name);
ALTER TABLE divisions DROP COLUMN IF EXISTS updated_at;
ALTER TABLE divisions DROP COLUMN IF EXISTS is_active;
ALTER TABLE divisions DROP COLUMN IF EXISTS partner_id;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE key IN ('m2.divisions.read', 'm2.divisions.manage'));
DELETE FROM permissions WHERE key IN ('m2.divisions.read', 'm2.divisions.manage');
