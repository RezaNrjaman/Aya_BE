-- ==============================================================================
-- 000003_partner_members.up.sql
-- Partner-scoped member management and partner information access.
-- ==============================================================================

-- 1. Link member accounts to the partner organisation they belong to.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_users_partner_id ON users(partner_id);

-- Backfill existing partner owner accounts so they immediately see their own
-- partner profile and members instead of every partner in the system.
UPDATE users u
SET partner_id = p.id
FROM partners p
WHERE p.user_id = u.id
  AND u.partner_id IS NULL;

-- 2. Partners (PT) may manage their own member accounts.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'partner'
  AND p.key IN (
      'm1.users.read',
      'm1.users.create',
      'm1.users.update',
      'm1.users.delete'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 3. Preserve the actor name for partner status history entries.
ALTER TABLE partner_status_histories
    ADD COLUMN IF NOT EXISTS changed_by_name VARCHAR(150);
