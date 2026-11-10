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
	q := `SELECT bu.id, bu.name, bu.category_id, COALESCE(bc.name, ''), bu.division_id, COALESCE(d.name, ''),
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
	q := `INSERT INTO divisions (partner_id, name, description, created_at, updated_at)
	      VALUES ($1, $2, $3, NOW(), NOW())
	      RETURNING id, is_active, created_at, updated_at`
	err := p.pool.QueryRow(ctx, q, d.PartnerID, d.Name, d.Description).
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
	q := `SELECT id, partner_id, name, COALESCE(description, ''), is_active, created_at, updated_at
	      FROM divisions WHERE id = $1`
	var d domain.Division
	err := p.pool.QueryRow(ctx, q, id).Scan(&d.ID, &d.PartnerID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDivisionNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (p *PostgresUnitRepository) UpdateDivision(ctx context.Context, d *domain.Division) error {
	q := `UPDATE divisions SET partner_id = $1, name = $2, description = $3, updated_at = NOW()
	      WHERE id = $4
	      RETURNING updated_at`
	err := p.pool.QueryRow(ctx, q, d.PartnerID, d.Name, d.Description, d.ID).Scan(&d.UpdatedAt)
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
	q := `SELECT id, partner_id, name, COALESCE(description, ''), is_active, created_at, updated_at
	      FROM divisions
	      WHERE ($1 = '%%' OR name ILIKE $1 OR COALESCE(description, '') ILIKE $1)
	        AND ($2::INT IS NULL OR partner_id = $2)
	      ORDER BY id ASC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, sPattern, partnerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var divisions []domain.Division
	for rows.Next() {
		var d domain.Division
		if err := rows.Scan(&d.ID, &d.PartnerID, &d.Name, &d.Description, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
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
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE division_id = $1`, id).Scan(&users); err != nil {
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
