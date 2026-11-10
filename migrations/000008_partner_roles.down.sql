-- ARY-007 rollback: detach custom roles from partners. Partner-owned roles
-- must be removed first (their slugs would otherwise collide under the
-- restored global unique constraint).

DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE partner_id IS NOT NULL);
DELETE FROM user_role_assignments
WHERE role_id IN (SELECT id FROM roles WHERE partner_id IS NOT NULL);
DELETE FROM roles WHERE partner_id IS NOT NULL;

DROP INDEX IF EXISTS idx_roles_partner_name;
DROP INDEX IF EXISTS idx_roles_partner_id;
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;
ALTER TABLE roles ADD CONSTRAINT roles_name_key UNIQUE (name);
ALTER TABLE roles DROP COLUMN IF EXISTS is_system;
ALTER TABLE roles DROP COLUMN IF EXISTS partner_id;
