package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"araya-business-hub/be/internal/modules/m4/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) CreateProduct(ctx context.Context, prod *domain.Product) error {
	q := `INSERT INTO products (unit_usaha_id, partner_id, name, product_type, category, description, price, stock, is_active, image_url, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()) RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, q, prod.UnitUsahaID, prod.PartnerID, prod.Name, prod.ProductType, prod.Category, prod.Description,
		prod.Price, prod.Stock, prod.IsActive, prod.ImageURL).Scan(&prod.ID, &prod.CreatedAt, &prod.UpdatedAt)
}

func (p *PostgresRepository) GetProductByID(ctx context.Context, id int) (*domain.Product, error) {
	q := `SELECT id, unit_usaha_id, partner_id, name, COALESCE(product_type, 'barang'), COALESCE(category, ''), COALESCE(description, ''),
	             price, stock, is_active, COALESCE(image_url, ''), created_at, updated_at
	      FROM products WHERE id = $1`
	var prod domain.Product
	err := p.pool.QueryRow(ctx, q, id).Scan(&prod.ID, &prod.UnitUsahaID, &prod.PartnerID, &prod.Name, &prod.ProductType, &prod.Category,
		&prod.Description, &prod.Price, &prod.Stock, &prod.IsActive, &prod.ImageURL, &prod.CreatedAt, &prod.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return &prod, nil
}

func (p *PostgresRepository) UpdateProduct(ctx context.Context, prod *domain.Product) error {
	q := `UPDATE products SET unit_usaha_id = $1, partner_id = $2, name = $3, product_type = $4, category = $5, description = $6,
	             price = $7, stock = $8, is_active = $9, image_url = $10, updated_at = NOW() WHERE id = $11`
	ct, err := p.pool.Exec(ctx, q, prod.UnitUsahaID, prod.PartnerID, prod.Name, prod.ProductType, prod.Category, prod.Description,
		prod.Price, prod.Stock, prod.IsActive, prod.ImageURL, prod.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (p *PostgresRepository) ValidateUnitOwnership(ctx context.Context, unitID int, partnerID *int) error {
	var owner sql.NullInt32
	err := p.pool.QueryRow(ctx, `SELECT partner_id FROM business_units WHERE id = $1`, unitID).Scan(&owner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrProductNotFound
		}
		return err
	}
	if partnerID == nil {
		if owner.Valid {
			return errors.New("unit usaha bukan unit platform")
		}
		return nil
	}
	if !owner.Valid || int(owner.Int32) != *partnerID {
		return errors.New("unit usaha bukan milik perusahaan Anda")
	}
	return nil
}

func (p *PostgresRepository) ListProducts(ctx context.Context, unitID *int, category, search string, activeOnly bool, partnerID *int, limit, offset int) ([]domain.Product, int, error) {
	pat := "%" + strings.TrimSpace(search) + "%"
	catPat := "%" + strings.TrimSpace(category) + "%"
	countQ := `SELECT count(*) FROM products
	           WHERE ($1::int IS NULL OR unit_usaha_id = $1) AND ($2 = '%%' OR category ILIKE $2)
	             AND ($3 = '%%' OR name ILIKE $3) AND ($4 = FALSE OR is_active = TRUE)
	             AND ($5::int IS NULL OR partner_id = $5)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, unitID, catPat, pat, activeOnly, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	prods, err := p.queryProducts(ctx, unitID, catPat, pat, activeOnly, partnerID, limit, offset)
	return prods, total, err
}

func (p *PostgresRepository) queryProducts(ctx context.Context, u *int, c, s string, active bool, partnerID *int, limit, offset int) ([]domain.Product, error) {
	q := `SELECT id, unit_usaha_id, partner_id, name, COALESCE(product_type, 'barang'), COALESCE(category, ''), COALESCE(description, ''),
	             price, stock, is_active, COALESCE(image_url, ''), created_at, updated_at
	      FROM products WHERE ($1::int IS NULL OR unit_usaha_id = $1) AND ($2 = '%%' OR category ILIKE $2)
	        AND ($3 = '%%' OR name ILIKE $3) AND ($4 = FALSE OR is_active = TRUE)
	        AND ($6::int IS NULL OR partner_id = $6)
	      ORDER BY id ASC LIMIT $5 OFFSET $7`
	rows, err := p.pool.Query(ctx, q, u, c, s, active, limit, partnerID, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return p.collectProducts(rows)
}

func (p *PostgresRepository) collectProducts(rows pgx.Rows) ([]domain.Product, error) {
	var prods []domain.Product
	for rows.Next() {
		var prod domain.Product
		if err := rows.Scan(&prod.ID, &prod.UnitUsahaID, &prod.PartnerID, &prod.Name, &prod.ProductType, &prod.Category,
			&prod.Description, &prod.Price, &prod.Stock, &prod.IsActive, &prod.ImageURL, &prod.CreatedAt, &prod.UpdatedAt); err != nil {
			return nil, err
		}
		prods = append(prods, prod)
	}
	return prods, nil
}

func (p *PostgresRepository) CreateCustomer(ctx context.Context, c *domain.Customer) error {
	q := `INSERT INTO customers (partner_id, name, email, phone, address, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, q, c.PartnerID, c.Name, c.Email, c.Phone, c.Address).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (p *PostgresRepository) GetCustomerByID(ctx context.Context, id int) (*domain.Customer, error) {
	q := `SELECT id, partner_id, name, email, COALESCE(phone, ''), COALESCE(address, ''), created_at, updated_at FROM customers WHERE id = $1`
	var c domain.Customer
	err := p.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.PartnerID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (p *PostgresRepository) GetCustomerByEmail(ctx context.Context, email string, partnerID *int) (*domain.Customer, error) {
	q := `SELECT id, partner_id, name, email, COALESCE(phone, ''), COALESCE(address, ''), created_at, updated_at
	      FROM customers
	      WHERE LOWER(email) = LOWER($1) AND partner_id IS NOT DISTINCT FROM $2
	      ORDER BY id ASC LIMIT 1`
	var c domain.Customer
	err := p.pool.QueryRow(ctx, q, strings.TrimSpace(email), partnerID).Scan(
		&c.ID, &c.PartnerID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (p *PostgresRepository) ListCustomers(ctx context.Context, search string, partnerID *int, limit, offset int) ([]domain.Customer, int, error) {
	pat := "%" + strings.TrimSpace(search) + "%"
	var total int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE ($1 = '%%' OR name ILIKE $1 OR email ILIKE $1) AND ($2::int IS NULL OR partner_id = $2)`, pat, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT id, partner_id, name, email, COALESCE(phone, ''), COALESCE(address, ''), created_at, updated_at
	      FROM customers WHERE ($1 = '%%' OR name ILIKE $1 OR email ILIKE $1) AND ($2::int IS NULL OR partner_id = $2)
	      ORDER BY id ASC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, pat, partnerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	custs, err := p.collectCustomers(rows)
	return custs, total, err
}

func (p *PostgresRepository) collectCustomers(rows pgx.Rows) ([]domain.Customer, error) {
	var custs []domain.Customer
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.ID, &c.PartnerID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		custs = append(custs, c)
	}
	return custs, nil
}
