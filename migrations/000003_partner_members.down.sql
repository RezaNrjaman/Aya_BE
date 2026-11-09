-- ==============================================================================
-- 000003_partner_members.down.sql
-- Roll back partner-scoped member management.
-- ==============================================================================

DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'partner')
  AND permission_id IN (
      SELECT id FROM permissions
      WHERE key IN (
          'm1.users.read',
          'm1.users.create',
          'm1.users.update',
          'm1.users.delete'
      )
  );

ALTER TABLE partner_status_histories
    DROP COLUMN IF EXISTS changed_by_name;

DROP INDEX IF EXISTS idx_users_partner_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS partner_id;
