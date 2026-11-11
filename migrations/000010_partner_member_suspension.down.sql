-- Rollback is destructive to the suspension marker; export affected IDs first.
DROP INDEX IF EXISTS idx_users_partner_suspended;
ALTER TABLE users DROP COLUMN IF EXISTS is_partner_suspended;
