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
	q := `INSERT INTO business_units (name, category_id, division_id, description, contact_email, contact_phone, address, logo_url, is_active, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	      RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, q, u.Name, u.CategoryID, u.DivisionID, u.Description, u.ContactEmail, u.ContactPhone, u.Address, u.LogoURL, u.IsActive).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (p *PostgresUnitRepository) GetByID(ctx context.Context, id int) (*domain.BusinessUnit, error) {
	q := `SELECT bu.id, bu.name, bu.category_id, COALESCE(bc.name, ''), bu.division_id, COALESCE(d.name, ''),
	             COALESCE(bu.description, ''), COALESCE(bu.contact_email, ''), COALESCE(bu.contact_phone, ''),
	             COALESCE(bu.address, ''), COALESCE(bu.logo_url, ''), bu.is_active, bu.created_at, bu.updated_at
	      FROM business_units bu
	      LEFT JOIN business_categories bc ON bu.category_id = bc.id
	      LEFT JOIN divisions d ON bu.division_id = d.id
	      WHERE bu.id = $1`
	var u domain.BusinessUnit
	err := p.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Name, &u.CategoryID, &u.CategoryName, &u.DivisionID, &u.DivisionName,
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
	q := `UPDATE business_units SET name = $1, category_id = $2, division_id = $3, description = $4,
	             contact_email = $5, contact_phone = $6, address = $7, logo_url = $8, is_active = $9, updated_at = NOW()
	      WHERE id = $10`
	ct, err := p.pool.Exec(ctx, q, u.Name, u.CategoryID, u.DivisionID, u.Description, u.ContactEmail, u.ContactPhone, u.Address, u.LogoURL, u.IsActive, u.ID)
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

func (p *PostgresUnitRepository) List(ctx context.Context, search, category string, activeOnly bool, limit, offset int) ([]domain.BusinessUnit, int, error) {
	sPattern := "%" + strings.TrimSpace(search) + "%"
	cPattern := "%" + strings.TrimSpace(category) + "%"
	countQ := `SELECT count(*) FROM business_units bu LEFT JOIN business_categories bc ON bu.category_id = bc.id
	           WHERE ($1 = '%%' OR bu.name ILIKE $1) AND ($2 = '%%' OR bc.name ILIKE $2) AND ($3 = FALSE OR bu.is_active = TRUE)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, sPattern, cPattern, activeOnly).Scan(&total); err != nil {
		return nil, 0, err
	}
	units, err := p.queryUnitList(ctx, sPattern, cPattern, activeOnly, limit, offset)
	return units, total, err
}

func (p *PostgresUnitRepository) queryUnitList(ctx context.Context, s, c string, active bool, limit, offset int) ([]domain.BusinessUnit, error) {
	q := `SELECT bu.id, bu.name, bu.category_id, COALESCE(bc.name, ''), bu.division_id, COALESCE(d.name, ''),
	             COALESCE(bu.description, ''), COALESCE(bu.contact_email, ''), COALESCE(bu.contact_phone, ''),
	             COALESCE(bu.address, ''), COALESCE(bu.logo_url, ''), bu.is_active, bu.created_at, bu.updated_at
	      FROM business_units bu
	      LEFT JOIN business_categories bc ON bu.category_id = bc.id
	      LEFT JOIN divisions d ON bu.division_id = d.id
	      WHERE ($1 = '%%' OR bu.name ILIKE $1) AND ($2 = '%%' OR bc.name ILIKE $2) AND ($3 = FALSE OR bu.is_active = TRUE)
	      ORDER BY bu.id ASC LIMIT $4 OFFSET $5`
	rows, err := p.pool.Query(ctx, q, s, c, active, limit, offset)
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
		if err := rows.Scan(&u.ID, &u.Name, &u.CategoryID, &u.CategoryName, &u.DivisionID, &u.DivisionName,
			&u.Description, &u.ContactEmail, &u.ContactPhone, &u.Address, &u.LogoURL, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, nil
}
