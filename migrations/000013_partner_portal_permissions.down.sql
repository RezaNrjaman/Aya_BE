-- 000013_partner_portal_permissions.down.sql
-- Revoke only the additional tenant-management permissions granted here.
-- Keep baseline partner mappings from migration 000002 (m4.products.read,
-- m4.products.manage and m4.quotations.manage).
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.name = 'partner' AND r.is_system = TRUE
  AND p.key IN (
   'm1.users.read','m1.users.create','m1.users.update','m1.users.delete',
   'm1.roles.read','m1.roles.manage','m1.audit.read',
   'm2.business_units.read','m2.business_units.create','m2.business_units.update',
   'm2.categories.read','m2.categories.manage',
   'm2.divisions.read','m2.divisions.manage','m2.dashboard.read',
   'm4.customers.manage','m4.invoices.manage','m4.orders.manage',
   'm5.services.manage'
  );

-- Permission rows are retained because they may be referenced by other roles.
-- Partner owner linkage and products.partner_id reconciliation are data repairs
-- and intentionally are not automatically reversed.
