-- ARY-005: business units may belong to a partner tenant. NULL partner_id
-- means the unit is operated by the platform itself (legacy rows keep working
-- unchanged).

ALTER TABLE business_units ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_business_units_partner_id ON business_units(partner_id);

-- ARY-005: partners manage their own business units. Grant the unit-management
-- permissions to the seeded "partner" role for databases that already exist
-- (the Go seed only populates an empty store).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN ('m2.business_units.read', 'm2.business_units.create', 'm2.business_units.update')
WHERE r.name = 'partner'
ON CONFLICT DO NOTHING;
