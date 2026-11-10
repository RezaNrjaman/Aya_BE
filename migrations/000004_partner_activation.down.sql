DROP INDEX IF EXISTS idx_users_activation_token_hash;
ALTER TABLE users DROP COLUMN IF EXISTS is_manually_disabled;
ALTER TABLE users DROP COLUMN IF EXISTS activation_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS activation_token_hash;
DROP INDEX IF EXISTS uq_partners_registration_code;
ALTER TABLE partners DROP COLUMN IF EXISTS registration_code;
