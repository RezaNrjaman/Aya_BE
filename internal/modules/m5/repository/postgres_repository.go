package repository

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m5/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) CreateServiceRequest(ctx context.Context, req *domain.ServiceRequest) error {
	q := `INSERT INTO service_requests (unit_usaha_id, customer_name, customer_email, customer_phone, service_type, description, status, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, q, req.UnitUsahaID, req.CustomerName, req.CustomerEmail, req.CustomerPhone,
		req.ServiceType, req.Description, req.Status).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

func (p *PostgresRepository) GetServiceRequestByID(ctx context.Context, id int) (*domain.ServiceRequest, error) {
	q := `SELECT id, unit_usaha_id, customer_name, customer_email, COALESCE(customer_phone, ''), COALESCE(service_type, ''),
	             description, status, created_at, updated_at FROM service_requests WHERE id = $1`
	var req domain.ServiceRequest
	err := p.pool.QueryRow(ctx, q, id).Scan(&req.ID, &req.UnitUsahaID, &req.CustomerName, &req.CustomerEmail,
		&req.CustomerPhone, &req.ServiceType, &req.Description, &req.Status, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRequestNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (p *PostgresRepository) UpdateServiceRequest(ctx context.Context, req *domain.ServiceRequest) error {
	q := `UPDATE service_requests SET status = $1, updated_at = NOW() WHERE id = $2`
	ct, err := p.pool.Exec(ctx, q, req.Status, req.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrRequestNotFound
	}
	return nil
}

func (p *PostgresRepository) ListServiceRequests(ctx context.Context, unitID *int, status string, limit, offset int) ([]domain.ServiceRequest, int, error) {
	countQ := `SELECT count(*) FROM service_requests WHERE ($1::int IS NULL OR unit_usaha_id = $1) AND ($2 = '' OR status = $2)`
	var total int
	if err := p.pool.QueryRow(ctx, countQ, unitID, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT id, unit_usaha_id, customer_name, customer_email, COALESCE(customer_phone, ''), COALESCE(service_type, ''),
	             description, status, created_at, updated_at
	      FROM service_requests WHERE ($1::int IS NULL OR unit_usaha_id = $1) AND ($2 = '' OR status = $2)
	      ORDER BY id DESC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, q, unitID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	reqs, err := p.collectServiceRequests(rows)
	return reqs, total, err
}

func (p *PostgresRepository) collectServiceRequests(rows pgx.Rows) ([]domain.ServiceRequest, error) {
	var list []domain.ServiceRequest
	for rows.Next() {
		var req domain.ServiceRequest
		if err := rows.Scan(&req.ID, &req.UnitUsahaID, &req.CustomerName, &req.CustomerEmail,
			&req.CustomerPhone, &req.ServiceType, &req.Description, &req.Status, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, req)
	}
	return list, nil
}

func (p *PostgresRepository) GetOrCreateCart(ctx context.Context, sessionID string) (*domain.Cart, error) {
	c, err := p.GetCartBySession(ctx, sessionID)
	if err == nil {
		return c, nil
	}
	q := `INSERT INTO carts (session_id, created_at, updated_at) VALUES ($1, NOW(), NOW())
	      ON CONFLICT (session_id) DO UPDATE SET updated_at = NOW()
	      RETURNING id, session_id, customer_id, created_at, updated_at`
	var newCart domain.Cart
	if err := p.pool.QueryRow(ctx, q, sessionID).Scan(&newCart.ID, &newCart.SessionID, &newCart.CustomerID, &newCart.CreatedAt, &newCart.UpdatedAt); err != nil {
		return nil, err
	}
	newCart.Items = []domain.CartItem{}
	return &newCart, nil
}

func (p *PostgresRepository) GetCartBySession(ctx context.Context, sessionID string) (*domain.Cart, error) {
	q := `SELECT id, session_id, customer_id, created_at, updated_at FROM carts WHERE session_id = $1`
	var c domain.Cart
	err := p.pool.QueryRow(ctx, q, sessionID).Scan(&c.ID, &c.SessionID, &c.CustomerID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCartEmpty
		}
		return nil, err
	}
	items, err := p.loadCartItems(ctx, c.ID)
	c.Items = items
	return &c, err
}

func (p *PostgresRepository) loadCartItems(ctx context.Context, cartID int) ([]domain.CartItem, error) {
	q := `SELECT ci.id, ci.cart_id, ci.product_id, COALESCE(p.name, ''), ci.quantity, COALESCE(p.price, 0), ci.created_at, ci.updated_at
	      FROM cart_items ci LEFT JOIN products p ON ci.product_id = p.id WHERE ci.cart_id = $1`
	rows, err := p.pool.Query(ctx, q, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.CartItem
	for rows.Next() {
		var it domain.CartItem
		if err := rows.Scan(&it.ID, &it.CartID, &it.ProductID, &it.ItemName, &it.Quantity, &it.UnitPrice, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.Subtotal = int64(it.Quantity) * it.UnitPrice
		items = append(items, it)
	}
	return items, nil
}

func (p *PostgresRepository) SaveCart(ctx context.Context, cart *domain.Cart) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cart.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cart.ID); err != nil {
		return err
	}
	for i := range cart.Items {
		itemQ := `INSERT INTO cart_items (cart_id, product_id, quantity, created_at, updated_at)
		          VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`
		if err := tx.QueryRow(ctx, itemQ, cart.ID, cart.Items[i].ProductID, cart.Items[i].Quantity).Scan(&cart.Items[i].ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
