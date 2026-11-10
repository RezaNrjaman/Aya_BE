-- ARY-006: divisions belong to a partner tenant. Platform divisions (legacy
-- rows) keep partner_id NULL and continue to work unchanged.

ALTER TABLE divisions ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;
ALTER TABLE divisions ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE divisions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Name uniqueness becomes per-tenant: platform divisions count as tenant 0.
-- The old global unique constraint must go, otherwise two partners could not
-- both have a "Finance" division.
ALTER TABLE divisions DROP CONSTRAINT IF EXISTS divisions_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_divisions_partner_name ON divisions (COALESCE(partner_id, 0), name);

CREATE INDEX IF NOT EXISTS idx_divisions_partner_id ON divisions(partner_id);

-- New granular permissions (existing databases receive them via SQL; fresh
-- installs are seeded by the application as well).
INSERT INTO permissions (key, module, name, description)
VALUES
    ('m2.divisions.read',   'm2', 'Lihat Divisi', 'Melihat daftar divisi milik tenant'),
    ('m2.divisions.manage', 'm2', 'Kelola Divisi', 'Membuat, mengubah, dan menonaktifkan divisi')
ON CONFLICT (key) DO NOTHING;

-- Partners manage their own divisions; admins manage all of them.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'partner')
  AND p.key IN ('m2.divisions.read', 'm2.divisions.manage')
ON CONFLICT (role_id, permission_id) DO NOTHING;
