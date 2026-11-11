-- 000011_management_hierarchy.up.sql
-- Enforce Partner -> Business Unit -> Category -> Division -> Role -> User.
-- Existing rows are preserved. Legacy categories/divisions that cannot be
-- mapped unambiguously remain nullable and must be assigned by a Partner admin.

ALTER TABLE business_categories
    ADD COLUMN IF NOT EXISTS partner_id INT REFERENCES partners(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS unit_usaha_id INT REFERENCES business_units(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Remove global category-name uniqueness: names are unique within a unit.
ALTER TABLE business_categories DROP CONSTRAINT IF EXISTS business_categories_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_business_categories_unit_name
    ON business_categories(unit_usaha_id, LOWER(name))
    WHERE unit_usaha_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_business_categories_partner_unit
    ON business_categories(partner_id, unit_usaha_id);

-- Legacy category rows referenced by a unit are assigned to that unit.
-- When a category is shared by multiple units, create a separate category row
-- for each additional unit and repoint that unit, preserving the original data.
DO $$
DECLARE
    rec RECORD;
    new_category_id INT;
BEGIN
    FOR rec IN
        SELECT bu.id AS unit_id, bu.partner_id, bu.category_id, bc.name, bc.description
        FROM business_units bu
        JOIN business_categories bc ON bc.id = bu.category_id
        WHERE bu.category_id IS NOT NULL
        ORDER BY bu.id
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM business_categories
            WHERE id = rec.category_id AND unit_usaha_id IS NULL
        ) THEN
            CONTINUE;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM business_categories
            WHERE id = rec.category_id AND unit_usaha_id IS NULL
        ) THEN
            CONTINUE;
        END IF;

        -- First referencing unit takes ownership of the legacy row.
        UPDATE business_categories
        SET unit_usaha_id = rec.unit_id, partner_id = rec.partner_id, updated_at = NOW()
        WHERE id = rec.category_id AND unit_usaha_id IS NULL;
    END LOOP;

    -- For shared legacy category IDs, duplicate a category for every unit
    -- whose unit does not match the category's new owner.
    FOR rec IN
        SELECT bu.id AS unit_id, bu.partner_id, bu.category_id, bc.name, bc.description, bc.unit_usaha_id
        FROM business_units bu
        JOIN business_categories bc ON bc.id = bu.category_id
        WHERE bu.category_id IS NOT NULL
          AND bc.unit_usaha_id IS DISTINCT FROM bu.id
    LOOP
        INSERT INTO business_categories(name, description, partner_id, unit_usaha_id, is_active, created_at, updated_at)
        VALUES (rec.name, rec.description, rec.partner_id, rec.unit_id, TRUE, NOW(), NOW())
        RETURNING id INTO new_category_id;
        UPDATE business_units SET category_id = new_category_id WHERE id = rec.unit_id;
    END LOOP;
END $$;

-- Legacy divisions: keep them, but they must be assigned to a unit/category
-- before they can be used for new role/user assignments.
ALTER TABLE divisions
    ADD COLUMN IF NOT EXISTS unit_usaha_id INT REFERENCES business_units(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS category_id INT REFERENCES business_categories(id) ON DELETE RESTRICT;

-- Infer a unit only when exactly one unit references the legacy division.
WITH division_unit_candidates AS (
    SELECT division_id, id AS unit_id
    FROM business_units
    WHERE division_id IS NOT NULL
    UNION
    SELECT u.division_id, ura.unit_usaha_id AS unit_id
    FROM users u
    JOIN user_role_assignments ura ON ura.user_id = u.id
    WHERE u.division_id IS NOT NULL AND ura.unit_usaha_id IS NOT NULL
), unique_unit AS (
    SELECT division_id, MIN(unit_id) AS unit_id
    FROM division_unit_candidates
    GROUP BY division_id
    HAVING COUNT(DISTINCT unit_id) = 1
)
UPDATE divisions d
SET unit_usaha_id = u.unit_id
FROM unique_unit u
WHERE d.id = u.division_id AND d.unit_usaha_id IS NULL;

-- Derive category from the inferred unit.
UPDATE divisions d
SET category_id = bu.category_id,
    partner_id = bu.partner_id
FROM business_units bu
WHERE d.unit_usaha_id = bu.id
  AND d.category_id IS NULL
  AND bu.category_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_divisions_unit_category
    ON divisions(unit_usaha_id, category_id);
DROP INDEX IF EXISTS idx_divisions_partner_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_divisions_category_name
    ON divisions(category_id, LOWER(name))
    WHERE category_id IS NOT NULL;

-- Roles now belong to one division. Legacy system/platform roles remain NULL.
ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS division_id INT REFERENCES divisions(id) ON DELETE RESTRICT;

-- Backfill a legacy role's division only when all current assignments point
-- to exactly one division. Ambiguous roles remain NULL for manual review.
WITH unique_role_division AS (
    SELECT ura.role_id, MIN(u.division_id) AS division_id
    FROM user_role_assignments ura
    JOIN users u ON u.id = ura.user_id
    WHERE u.division_id IS NOT NULL
    GROUP BY ura.role_id
    HAVING COUNT(DISTINCT u.division_id) = 1
)
UPDATE roles r
SET division_id = x.division_id
FROM unique_role_division x
WHERE r.id = x.role_id AND r.partner_id IS NOT NULL AND r.division_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_roles_division_id ON roles(division_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_division_name
    ON roles(division_id, LOWER(name))
    WHERE division_id IS NOT NULL;

-- A user's division is the authoritative organizational placement.
-- Role assignment must match both the user's tenant and division.
CREATE OR REPLACE FUNCTION enforce_user_role_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    user_partner INT;
    user_division INT;
    role_partner INT;
    role_division INT;
    role_unit INT;
BEGIN
    SELECT partner_id, division_id INTO user_partner, user_division
    FROM users WHERE id = NEW.user_id;

    SELECT r.partner_id, r.division_id, d.unit_usaha_id
      INTO role_partner, role_division, role_unit
    FROM roles r
    LEFT JOIN divisions d ON d.id = r.division_id
    WHERE r.id = NEW.role_id;

    -- Platform system roles remain unscoped.
    IF role_partner IS NULL AND role_division IS NULL
       AND user_partner IS NULL AND user_division IS NULL
       AND EXISTS (SELECT 1 FROM roles WHERE id = NEW.role_id AND is_system = TRUE) THEN
        RETURN NEW;
    END IF;

    -- Special bootstrap exception: the Partner owner account keeps the
    -- tenant-wide system role "partner" until it creates its first org tree.
    IF role_partner IS NULL AND role_division IS NULL
       AND user_partner IS NOT NULL AND user_division IS NULL THEN
        IF EXISTS (SELECT 1 FROM roles WHERE id = NEW.role_id AND name = 'partner' AND is_system = TRUE) THEN
            RETURN NEW;
        END IF;
    END IF;

    IF user_partner IS NULL OR role_partner IS NULL OR user_partner <> role_partner THEN
        RAISE EXCEPTION 'role dan pengguna harus berasal dari Partner yang sama'
            USING ERRCODE = '23514';
    END IF;

    IF user_division IS NULL OR role_division IS NULL OR user_division <> role_division THEN
        RAISE EXCEPTION 'role dan pengguna harus berada pada divisi yang sama'
            USING ERRCODE = '23514';
    END IF;

    IF NEW.unit_usaha_id IS DISTINCT FROM role_unit THEN
        RAISE EXCEPTION 'unit usaha penugasan harus sama dengan unit usaha divisi role'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_user_role_hierarchy ON user_role_assignments;
CREATE TRIGGER trg_user_role_hierarchy
BEFORE INSERT OR UPDATE OF user_id, role_id ON user_role_assignments
FOR EACH ROW EXECUTE FUNCTION enforce_user_role_hierarchy();

-- Prevent a division's category and unit from being mixed across tenants.
CREATE OR REPLACE FUNCTION enforce_category_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    unit_partner INT;
BEGIN
    IF NEW.unit_usaha_id IS NULL OR NEW.partner_id IS NULL THEN
        RAISE EXCEPTION 'kategori wajib memiliki Partner dan unit usaha'
            USING ERRCODE = '23514';
    END IF;
    SELECT partner_id INTO unit_partner FROM business_units WHERE id = NEW.unit_usaha_id;
    IF unit_partner IS DISTINCT FROM NEW.partner_id THEN
        RAISE EXCEPTION 'kategori dan unit usaha harus berasal dari Partner yang sama'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_category_hierarchy ON business_categories;
CREATE TRIGGER trg_category_hierarchy
BEFORE INSERT OR UPDATE OF partner_id, unit_usaha_id ON business_categories
FOR EACH ROW EXECUTE FUNCTION enforce_category_hierarchy();

CREATE OR REPLACE FUNCTION enforce_division_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    category_unit INT;
    category_partner INT;
    unit_partner INT;
BEGIN
    IF NEW.unit_usaha_id IS NULL OR NEW.category_id IS NULL THEN
        RAISE EXCEPTION 'divisi wajib memiliki unit usaha dan kategori'
            USING ERRCODE = '23514';
    END IF;

    SELECT unit_usaha_id, partner_id INTO category_unit, category_partner
    FROM business_categories WHERE id = NEW.category_id;

    SELECT partner_id INTO unit_partner
    FROM business_units WHERE id = NEW.unit_usaha_id;

    IF category_unit IS DISTINCT FROM NEW.unit_usaha_id THEN
        RAISE EXCEPTION 'kategori tidak termasuk dalam unit usaha yang dipilih'
            USING ERRCODE = '23514';
    END IF;

    IF category_partner IS DISTINCT FROM unit_partner OR NEW.partner_id IS DISTINCT FROM unit_partner THEN
        RAISE EXCEPTION 'unit usaha, kategori, dan divisi harus berasal dari Partner yang sama'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_division_hierarchy ON divisions;
CREATE TRIGGER trg_division_hierarchy
BEFORE INSERT OR UPDATE OF partner_id, unit_usaha_id, category_id ON divisions
FOR EACH ROW EXECUTE FUNCTION enforce_division_hierarchy();

-- Ensure role division belongs to the same tenant as the role.
CREATE OR REPLACE FUNCTION enforce_role_division_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    division_partner INT;
BEGIN
    IF NEW.division_id IS NULL THEN
        IF NEW.partner_id IS NOT NULL AND COALESCE(NEW.is_system, FALSE) = FALSE THEN
            RAISE EXCEPTION 'role Partner wajib terikat ke divisi'
                USING ERRCODE = '23514';
        END IF;
        RETURN NEW;
    END IF;

    SELECT partner_id INTO division_partner FROM divisions WHERE id = NEW.division_id;
    IF division_partner IS DISTINCT FROM NEW.partner_id THEN
        RAISE EXCEPTION 'role dan divisi harus berasal dari Partner yang sama'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_role_division_hierarchy ON roles;
CREATE TRIGGER trg_role_division_hierarchy
BEFORE INSERT OR UPDATE OF partner_id, division_id ON roles
FOR EACH ROW EXECUTE FUNCTION enforce_role_division_hierarchy();

CREATE INDEX IF NOT EXISTS idx_users_partner_division
    ON users(partner_id, division_id);


INSERT INTO permissions(key, module, name, description, is_active)
VALUES
 ('m2.categories.read', 'm2', 'Lihat Kategori', 'Melihat kategori di dalam unit usaha', TRUE),
 ('m2.categories.manage', 'm2', 'Kelola Kategori', 'Membuat, mengubah, dan menonaktifkan kategori', TRUE)
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN ('m2.categories.read', 'm2.categories.manage')
WHERE r.name = 'partner' AND r.partner_id IS NULL
ON CONFLICT DO NOTHING;
