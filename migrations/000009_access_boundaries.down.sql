-- Rollback for 000009_access_boundaries.up.sql.
-- Restores the legacy broad admin role and legacy partner permissions.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN (
    'm1.users.read', 'm1.users.create', 'm1.users.update', 'm1.users.delete',
    'm2.business_units.read', 'm2.business_units.create', 'm2.business_units.update',
    'm2.divisions.read', 'm2.divisions.manage', 'm2.dashboard.read',
    'm3.partners.read', 'm3.partners.register',
    'm4.products.read', 'm4.products.manage', 'm4.customers.manage',
    'm4.quotations.manage', 'm4.invoices.manage', 'm4.orders.manage'
)
WHERE r.name = 'partner'
ON CONFLICT (role_id, permission_id) DO NOTHING;

DROP INDEX IF EXISTS idx_audit_logs_partner_created;
DROP INDEX IF EXISTS idx_service_requests_unit_created;
