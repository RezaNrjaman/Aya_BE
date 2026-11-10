-- ARY-002: Remove the universal initial partner password and replace the
-- public "check status by sequential ID" endpoint with an unguessable
-- registration tracking code.

-- 1) Partner tracking code (random, unguessable) used by the public status page.
ALTER TABLE partners ADD COLUMN IF NOT EXISTS registration_code VARCHAR(64);

-- Backfill existing partners with a random code (md5 is only used as a random
-- generator here; the result is never a secret beyond its owner).
UPDATE partners
SET registration_code = MD5(random()::text || clock_timestamp()::text || id::text)
WHERE registration_code IS NULL OR registration_code = '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_partners_registration_code ON partners(registration_code);

-- 2) Per-user activation / password-reset invitation tokens.
-- The plaintext token is never stored: only its SHA-256 hash, with an expiry.
ALTER TABLE users ADD COLUMN IF NOT EXISTS activation_token_hash VARCHAR(128);
ALTER TABLE users ADD COLUMN IF NOT EXISTS activation_expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_activation_token_hash ON users(activation_token_hash);

-- 3) Distinguish accounts manually disabled by an administrator from accounts
-- disabled because their partner was suspended, so a reactivate never
-- re-enables manually disabled accounts (ARY-004).
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_manually_disabled BOOLEAN NOT NULL DEFAULT FALSE;

-- NOTE (ops): any account that previously logged in with the universal
-- "Partner123!" password must have its password rotated/reset before this
-- release is considered fully deployed. Run:
--   UPDATE users SET is_active = FALSE WHERE password_hash <> '' AND partner_id IS NOT NULL;
-- then re-issue activation invitations per partner, or force a password reset.

-- Down is provided in 000004_partner_activation.down.sql.
