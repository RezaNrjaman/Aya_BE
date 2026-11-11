-- 000009_access_boundaries.up.sql
-- Separate platform administration from tenant operations.
-- Run after 000003 through 000008.

-- Every role without a tenant owner is a platform/system role.
UPDATE roles SET is_system = TRUE WHERE partner_id IS NULL;

-- Super Admin: platform governance only (partner verification/suspension).
DELETE FROM role_permissions rp
USING roles r
WHERE rp.role_id = r.id AND r.name = 'admin';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN (
    'm3.partners.read',
    'm3.partners.verify',
    'm3.partners.suspend'
)
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Partner owner: tenant organization, transactions, service requests, and audit.
-- No M3 partner-governance permission is granted to tenant accounts.
DELETE FROM role_permissions rp
USING roles r
WHERE rp.role_id = r.id AND r.name = 'partner';

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN (
    'm1.users.read',
    'm1.users.create',
    'm1.users.update',
    'm1.users.delete',
    'm1.roles.read',
    'm1.roles.manage',
    'm1.audit.read',
    'm2.business_units.read',
    'm2.business_units.create',
    'm2.business_units.update',
    'm2.divisions.read',
    'm2.divisions.manage',
    'm2.dashboard.read',
    'm4.products.read',
    'm4.products.manage',
    'm4.customers.manage',
    'm4.quotations.manage',
    'm4.invoices.manage',
    'm4.orders.manage',
    'm5.services.manage'
)
WHERE r.name = 'partner'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Attribute historical audit rows to the actor's tenant when the actor was
-- already linked to a partner. Platform events remain NULL.
UPDATE audit_logs a
SET partner_id = u.partner_id
FROM users u
WHERE a.user_id = u.id
  AND a.partner_id IS NULL
  AND u.partner_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_partner_created
    ON audit_logs(partner_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_service_requests_unit_created
    ON service_requests(unit_usaha_id, created_at DESC);
