-- 000011_management_hierarchy.down.sql
-- Rollback removes hierarchy enforcement and added columns. Data created under
-- the new per-unit category model is retained as legacy category rows.
DROP TRIGGER IF EXISTS trg_category_hierarchy ON business_categories;
DROP FUNCTION IF EXISTS enforce_category_hierarchy();
DROP TRIGGER IF EXISTS trg_user_role_hierarchy ON user_role_assignments;
DROP FUNCTION IF EXISTS enforce_user_role_hierarchy();
DROP TRIGGER IF EXISTS trg_division_hierarchy ON divisions;
DROP FUNCTION IF EXISTS enforce_division_hierarchy();
DROP TRIGGER IF EXISTS trg_role_division_hierarchy ON roles;
DROP FUNCTION IF EXISTS enforce_role_division_hierarchy();

DROP INDEX IF EXISTS idx_users_partner_division;
DROP INDEX IF EXISTS idx_roles_division_name;
DROP INDEX IF EXISTS idx_roles_division_id;
DROP INDEX IF EXISTS idx_divisions_category_name;
DROP INDEX IF EXISTS idx_divisions_unit_category;
DROP INDEX IF EXISTS idx_business_categories_partner_unit;
DROP INDEX IF EXISTS idx_business_categories_unit_name;

ALTER TABLE roles DROP COLUMN IF EXISTS division_id;
ALTER TABLE divisions DROP COLUMN IF EXISTS category_id;
ALTER TABLE divisions DROP COLUMN IF EXISTS unit_usaha_id;
ALTER TABLE business_categories DROP COLUMN IF EXISTS unit_usaha_id;
ALTER TABLE business_categories DROP COLUMN IF EXISTS partner_id;
ALTER TABLE business_categories DROP COLUMN IF EXISTS is_active;
ALTER TABLE business_categories DROP COLUMN IF EXISTS updated_at;
