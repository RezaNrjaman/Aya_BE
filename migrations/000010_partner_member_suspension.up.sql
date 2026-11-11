-- 000010_partner_member_suspension.up.sql
-- Track tenant suspension separately from manual user deactivation so a
-- reactivated partner does not accidentally re-enable manually disabled users.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_partner_suspended BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_partner_suspended
    ON users(partner_id, is_partner_suspended)
    WHERE is_partner_suspended = TRUE;
