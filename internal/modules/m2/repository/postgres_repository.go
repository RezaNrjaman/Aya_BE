package repository

import (
	"context"
	"errors"
	"strings"

	"araya-business-hub/be/internal/modules/m2/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUnitRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUnitRepository(pool *pgxpool.Pool) *PostgresUnitRepository {
	return &PostgresUnitRepository{pool: pool}
}

func (p *PostgresUnitRepository) Create(ctx context.Context, u *domain.BusinessUnit) error {
	q := `INSERT INTO business_units (partner_id, name, category_id, division_id, description, contact_email, contact_phone, address, logo_url, is_active, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	      RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, q, u.PartnerID, u.Name, u.CategoryID, u.DivisionID, u.Description, u.ContactEmail, u.ContactPhone, u.Address, u.LogoURL, u.IsActive).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (p *PostgresUnitRepository) GetByID(ctx context.Context, id int) (*domain.BusinessUnit, error) {
	q := `SELECT bu.id, bu.partner_id, bu.name, bu.category_id, COALESCE(bc.name, ''), bu.division_id, COALESCE(d.name, ''),
	             COALESCE(bu.description, ''), COALESCE(bu.contact_email, ''), COALESCE(bu.contact_phone, ''),
	             COALESCE(bu.address, ''), COALESCE(bu.logo_url, ''), bu.is_active, bu.created_at, bu.updated_at
	      FROM business_units bu
	      LEFT JOIN business_categories bc ON bu.category_id = bc.id
	      LEFT JOIN divisions d ON bu.division_id = d.id
	      WHERE bu.id = $1`
	var u domain.BusinessUnit
	err := p.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.PartnerID, &u.Name, &u.CategoryID, &u.CategoryName, &u.DivisionID, &u.DivisionName,
		&u.Description, &u.ContactEmail, &u.ContactPhone, &u.Address, &u.LogoURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUnitNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (p *PostgresUnitRepository) Update(ctx context.Context, u *domain.BusinessUnit) error {
	q := `UPDATE business_units SET partner_id = $1, name = $2, category_id = $3, division_id = $4, description = $5,
	             contact_email = $6, contact_phone = $7, address = $8, logo_url = $9, is_active = $10, updated_at = NOW()
	      WHERE id = $11`
	ct, err := p.pool.Exec(ctx, q, u.PartnerID, u.Name, u.CategoryID, u.DivisionID, u.Description, u.ContactEmail, u.ContactPhone, u.Address, u.LogoURL, u.IsActive, u.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUnitNotFound
	}
	return nil
}

func (p *PostgresUnitRepository) GetOrCreateCategoryID(ctx context.Context, name string) (*int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var id int
	err := p.pool.QueryRow(ctx, `INSERT INTO business_categories (name)
		VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (p *PostgresUnitRepository) ValidatePartnerDivision(ctx context.Context, partnerID int, divisionID *int) error {
	if divisionID == nil {
		return nil
	}
	var exists bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM divisions WHERE id = $1 AND partner_id = $2 AND is_active = TRUE
	)`, *divisionID, partnerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrDivisionNotFound
	}
	return nil
}

func (p *PostgresUnitRepository) SetStatus(ctx context.Context, id int, active bool) error {
	q := `UPDATE business_units SET is_active = $1, updated_at = NOW() WHERE id = $2`
	ct, err := p.pool.Exec(ctx, q, active, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrUnitNotFound
	}
	return nil
}

func (p *PostgresUnitRepository) ListActive(ctx context.Context) ([]domain.BusinessUnit, error) {
	q := `SELECT bu.id, bu.partner_id, bu.name, bu.category_id, COALESCE(bc.name, ''), bu.division_id, COALESCE(d.name, ''),
	             COALESCE(bu.description, ''), COALESCE(bu.contact_email, ''), COALESCE(bu.contact_phone, ''),
	             COALESCE(bu.address, ''), COALESCE(bu.logo_url, ''), bu.is_active, bu.created_at, bu.updated_at
	      FROM business_units bu
	      LEFT JOIN business_categories bc ON bu.category_id = bc.id
	      LEFT JOIN divisions d ON bu.division_id = d.id
	      WHERE bu.is_active = TRUE ORDER BY bu.name ASC`
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return p.collectRows(rows)
}

func (p *PostgresUnitRepository) List(ctx context.Context, search, category string, activeOnly bool, partnerID *int, limit, offset int) ([]domain.BusinessUnit, int, error) {
	sPattern := "%" + strings.TrimSpace(search) + "%"
	cPattern := "%" + strings.TrimSpace(category) + "%"
	countQ := `SELECT count(*) FROM business_units bu LEFT JOIN business_categories bc ON bu.category_id = bc.id
	           WHERE ($1 = '%%' OR bu.name ILIKE $1) AND ($2 = '%%' OR bc.name ILIKE $2) AND ($3 = FALSE OR bu.is_active = TRUE)
	             AND ($4::INT IS NULL OR bu.partner_id = $4)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, sPattern, cPattern, activeOnly, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	units, err := p.queryUnitList(ctx, sPattern, cPattern, activeOnly, partnerID, limit, offset)
	return units, total, err
}

func (p *PostgresUnitRepository) queryUnitList(ctx context.Context, s, c string, active bool, partnerID *int, limit, offset int) ([]domain.BusinessUnit, error) {
	q := `SELECT bu.id, bu.partner_id, bu.name, bu.category_id, COALESCE(bc.name, ''), bu.division_id, COALESCE(d.name, ''),
	             COALESCE(bu.description, ''), COALESCE(bu.contact_email, ''), COALESCE(bu.contact_phone, ''),
	             COALESCE(bu.address, ''), COALESCE(bu.logo_url, ''), bu.is_active, bu.created_at, bu.updated_at
	      FROM business_units bu
	      LEFT JOIN business_categories bc ON bu.category_id = bc.id
	      LEFT JOIN divisions d ON bu.division_id = d.id
	      WHERE ($1 = '%%' OR bu.name ILIKE $1) AND ($2 = '%%' OR bc.name ILIKE $2) AND ($3 = FALSE OR bu.is_active = TRUE)
	        AND ($6::INT IS NULL OR bu.partner_id = $6)
	      ORDER BY bu.id ASC LIMIT $4 OFFSET $5`
	rows, err := p.pool.Query(ctx, q, s, c, active, limit, offset, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return p.collectRows(rows)
}

func (p *PostgresUnitRepository) collectRows(rows pgx.Rows) ([]domain.BusinessUnit, error) {
	var units []domain.BusinessUnit
	for rows.Next() {
		var u domain.BusinessUnit
		if err := rows.Scan(&u.ID, &u.PartnerID, &u.Name, &u.CategoryID, &u.CategoryName, &u.DivisionID, &u.DivisionName,
			&u.Description, &u.ContactEmail, &u.ContactPhone, &u.Address, &u.LogoURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, nil
}

// --- DivisionRepository (ARY-006) ---

func (p *PostgresUnitRepository) CreateDivision(ctx context.Context, d *domain.Division) error {
	q := `INSERT INTO divisions (partner_id, unit_usaha_id, category_id, name, description, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	      RETURNING id, is_active, created_at, updated_at`
	err := p.pool.QueryRow(ctx, q, d.PartnerID, d.UnitUsahaID, d.CategoryID, d.Name, d.Description).
		Scan(&d.ID, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicateDivision
		}
		return err
	}
	return nil
}

func (p *PostgresUnitRepository) GetDivisionByID(ctx context.Context, id int) (*domain.Division, error) {
	q := `SELECT d.id, d.partner_id, COALESCE(d.unit_usaha_id,0), COALESCE(d.category_id,0),
	             COALESCE(bu.name,''), COALESCE(bc.name,''), d.name, COALESCE(d.description,''), d.is_active, d.created_at, d.updated_at
	      FROM divisions d
	      LEFT JOIN business_units bu ON bu.id=d.unit_usaha_id
	      LEFT JOIN business_categories bc ON bc.id=d.category_id
	      WHERE d.id = $1`
	var d domain.Division
	err := p.pool.QueryRow(ctx, q, id).Scan(&d.ID, &d.PartnerID, &d.UnitUsahaID, &d.CategoryID, &d.UnitName, &d.CategoryName,
		&d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDivisionNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (p *PostgresUnitRepository) UpdateDivision(ctx context.Context, d *domain.Division) error {
	q := `UPDATE divisions SET partner_id = $1, unit_usaha_id = $2, category_id = $3,
	             name = $4, description = $5, updated_at = NOW()
	      WHERE id = $6 RETURNING updated_at`
	err := p.pool.QueryRow(ctx, q, d.PartnerID, d.UnitUsahaID, d.CategoryID, d.Name, d.Description, d.ID).Scan(&d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrDivisionNotFound
		}
		if isUniqueViolation(err) {
			return domain.ErrDuplicateDivision
		}
		return err
	}
	return nil
}

func (p *PostgresUnitRepository) ListDivisions(ctx context.Context, search string, partnerID *int, limit, offset int) ([]domain.Division, int, error) {
	sPattern := "%" + strings.TrimSpace(search) + "%"
	countQ := `SELECT count(*) FROM divisions
	           WHERE ($1 = '%%' OR name ILIKE $1 OR COALESCE(description, '') ILIKE $1)
	             AND ($2::INT IS NULL OR partner_id = $2)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, sPattern, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT d.id, d.partner_id, COALESCE(d.unit_usaha_id,0), COALESCE(d.category_id,0),
	             COALESCE(bu.name,''), COALESCE(bc.name,''), d.name, COALESCE(d.description,''), d.is_active, d.created_at, d.updated_at
	      FROM divisions d
	      LEFT JOIN business_units bu ON bu.id=d.unit_usaha_id
	      LEFT JOIN business_categories bc ON bc.id=d.category_id
	      WHERE ($1 = '%%' OR d.name ILIKE $1 OR COALESCE(d.description, '') ILIKE $1)
	        AND ($2::INT IS NULL OR d.partner_id = $2)
	      ORDER BY d.id ASC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, sPattern, partnerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var divisions []domain.Division
	for rows.Next() {
		var d domain.Division
		if err := rows.Scan(&d.ID, &d.PartnerID, &d.UnitUsahaID, &d.CategoryID, &d.UnitName, &d.CategoryName,
			&d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		divisions = append(divisions, d)
	}
	return divisions, total, nil
}

func (p *PostgresUnitRepository) SetDivisionStatus(ctx context.Context, id int, active bool) error {
	q := `UPDATE divisions SET is_active = $1, updated_at = NOW() WHERE id = $2`
	ct, err := p.pool.Exec(ctx, q, active, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrDivisionNotFound
	}
	return nil
}

func (p *PostgresUnitRepository) DeleteDivision(ctx context.Context, id int) error {
	q := `DELETE FROM divisions WHERE id = $1`
	ct, err := p.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrDivisionNotFound
	}
	return nil
}

func (p *PostgresUnitRepository) CountDivisionReferences(ctx context.Context, id int) (int, int, error) {
	var users, units int
	if err := p.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM users WHERE division_id = $1) +
		(SELECT count(*) FROM roles WHERE division_id = $1)`, id).Scan(&users); err != nil {
		return 0, 0, err
	}
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM business_units WHERE division_id = $1`, id).Scan(&units); err != nil {
		return 0, 0, err
	}
	return users, units, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

// --- CategoryRepository (management hierarchy) ---

func (p *PostgresUnitRepository) CreateCategory(ctx context.Context, c *domain.BusinessCategory) error {
	q := `INSERT INTO business_categories(partner_id, unit_usaha_id, name, description, is_active, created_at, updated_at)
	      SELECT bu.partner_id, bu.id, $3, $4, TRUE, NOW(), NOW()
	      FROM business_units bu WHERE bu.id = $1 AND bu.partner_id = $2
	      RETURNING id, partner_id, created_at, updated_at`
	err := p.pool.QueryRow(ctx, q, c.UnitUsahaID, c.PartnerID, c.Name, c.Description).
		Scan(&c.ID, &c.PartnerID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return errors.New("nama kategori sudah digunakan pada unit usaha ini")
		}
		return err
	}
	c.IsActive = true
	return nil
}

func (p *PostgresUnitRepository) ListCategories(ctx context.Context, unitID *int, partnerID *int, search string, limit, offset int) ([]domain.BusinessCategory, int, error) {
	pattern := "%" + strings.TrimSpace(search) + "%"
	var total int
	err := p.pool.QueryRow(ctx, `SELECT count(*) FROM business_categories c
		WHERE ($1::INT IS NULL OR c.unit_usaha_id = $1)
		  AND ($2::INT IS NULL OR c.partner_id = $2)
		  AND c.unit_usaha_id IS NOT NULL
		  AND ($3 = '%%' OR c.name ILIKE $3)`, unitID, partnerID, pattern).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := p.pool.Query(ctx, `SELECT c.id, c.partner_id, c.unit_usaha_id, c.name,
		COALESCE(c.description,''), c.is_active, c.created_at, c.updated_at
		FROM business_categories c
		WHERE ($1::INT IS NULL OR c.unit_usaha_id = $1)
		  AND ($2::INT IS NULL OR c.partner_id = $2)
		  AND c.unit_usaha_id IS NOT NULL
		  AND ($3 = '%%' OR c.name ILIKE $3)
		ORDER BY c.unit_usaha_id, c.name LIMIT $4 OFFSET $5`, unitID, partnerID, pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.BusinessCategory{}
	for rows.Next() {
		var c domain.BusinessCategory
		if err := rows.Scan(&c.ID, &c.PartnerID, &c.UnitUsahaID, &c.Name, &c.Description, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

func (p *PostgresUnitRepository) GetCategoryByID(ctx context.Context, id int) (*domain.BusinessCategory, error) {
	var c domain.BusinessCategory
	err := p.pool.QueryRow(ctx, `SELECT id, partner_id, unit_usaha_id, name, COALESCE(description,''), is_active, created_at, updated_at
		FROM business_categories WHERE id = $1`, id).
		Scan(&c.ID, &c.PartnerID, &c.UnitUsahaID, &c.Name, &c.Description, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (p *PostgresUnitRepository) UpdateCategory(ctx context.Context, c *domain.BusinessCategory) error {
	tag, err := p.pool.Exec(ctx, `UPDATE business_categories SET name=$1, description=$2, updated_at=NOW()
		WHERE id=$3 AND partner_id=$4 AND unit_usaha_id=$5`, c.Name, c.Description, c.ID, c.PartnerID, c.UnitUsahaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("kategori tidak ditemukan pada unit usaha ini")
	}
	return nil
}

func (p *PostgresUnitRepository) SetCategoryStatus(ctx context.Context, id int, active bool) error {
	tag, err := p.pool.Exec(ctx, `UPDATE business_categories SET is_active=$1, updated_at=NOW() WHERE id=$2`, active, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("kategori tidak ditemukan")
	}
	return nil
}

func (p *PostgresUnitRepository) ValidateCategoryPlacement(ctx context.Context, partnerID int, unitID, categoryID int) error {
	var exists bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM business_categories c
		JOIN business_units bu ON bu.id = c.unit_usaha_id
		WHERE c.id=$1 AND c.unit_usaha_id=$2 AND c.partner_id=$3
		  AND bu.partner_id=$3 AND bu.is_active=TRUE AND c.is_active=TRUE
	)`, categoryID, unitID, partnerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("kategori tidak termasuk dalam unit usaha Partner ini atau tidak aktif")
	}
	return nil
}

func (p *PostgresUnitRepository) ValidateDivisionPlacement(ctx context.Context, partnerID, unitID, categoryID int) error {
	var exists bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM business_categories c
		JOIN business_units bu ON bu.id=c.unit_usaha_id
		WHERE c.id=$1 AND c.unit_usaha_id=$2 AND c.partner_id=$3
		  AND bu.partner_id=$3 AND bu.is_active=TRUE AND c.is_active=TRUE
	)`, categoryID, unitID, partnerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrDivisionNotFound
	}
	return nil
}
