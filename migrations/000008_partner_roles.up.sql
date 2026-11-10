-- ARY-007: dynamic, partner-owned roles.
-- A role with partner_id NULL is a platform/system role. A role owned by a
-- partner is a custom organisational role (Finance, Kasir, ...) that only
-- that partner may see, edit, and assign.

ALTER TABLE roles ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE CASCADE;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

-- Mark the seeded baseline roles as system roles: they can never be renamed,
-- deactivated, or captured by partner-level role management.
UPDATE roles SET is_system = TRUE WHERE name IN ('admin', 'manajer', 'partner', 'kasir');

-- Index the tenant FK (speeds up per-partner role listings and CASCADEs).
CREATE INDEX IF NOT EXISTS idx_roles_partner_id ON roles(partner_id);

-- Role names must be unique per tenant, not globally: two partners may both
-- create a "Finance" role. System roles count as tenant 0.
-- The application additionally prefixes partner role slugs with the partner
-- id (e.g. "p7_finance") so authorization lookups stay globally unambiguous.
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_partner_name ON roles (COALESCE(partner_id, 0), name);
