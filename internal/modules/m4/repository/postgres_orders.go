package repository

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m4/domain"
	"github.com/jackc/pgx/v5"
)

func (p *PostgresRepository) CreateOrder(ctx context.Context, o *domain.Order) error {
	sql := `INSERT INTO orders (order_no, customer_id, invoice_id, total_amount, status, idempotency_key, created_at, updated_at)
	        VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) RETURNING id, created_at, updated_at`
	if err := p.pool.QueryRow(ctx, sql, o.OrderNo, o.CustomerID, o.InvoiceID, o.TotalAmount, o.Status, o.IdempotencyKey).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return err
	}
	return p.insertOrderItems(ctx, o.ID, o.Items)
}

func (p *PostgresRepository) insertOrderItems(ctx context.Context, oID int, items []domain.OrderItem) error {
	itemQ := `INSERT INTO order_items (order_id, product_id, item_name, quantity, unit_price, subtotal) VALUES ($1, $2, $3, $4, $5, $6)`
	for i := range items {
		items[i].OrderID = oID
		if err := p.pool.QueryRow(ctx, itemQ+` RETURNING id`, oID, items[i].ProductID, items[i].ItemName, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal).Scan(&items[i].ID); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresRepository) GetOrderByID(ctx context.Context, id int) (*domain.Order, error) {
	sql := `SELECT id, order_no, customer_id, invoice_id, total_amount, status, COALESCE(idempotency_key, ''), created_at, updated_at FROM orders WHERE id = $1`
	var o domain.Order
	if err := p.pool.QueryRow(ctx, sql, id).Scan(&o.ID, &o.OrderNo, &o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := p.loadOrderItems(ctx, o.ID)
	o.Items = items
	return &o, err
}

func (p *PostgresRepository) GetOrderByIdempotency(ctx context.Context, key string) (*domain.Order, error) {
	sql := `SELECT id, order_no, customer_id, invoice_id, total_amount, status, idempotency_key, created_at, updated_at FROM orders WHERE idempotency_key = $1`
	var o domain.Order
	if err := p.pool.QueryRow(ctx, sql, key).Scan(&o.ID, &o.OrderNo, &o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	items, err := p.loadOrderItems(ctx, o.ID)
	o.Items = items
	return &o, err
}

func (p *PostgresRepository) loadOrderItems(ctx context.Context, oID int) ([]domain.OrderItem, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, order_id, product_id, item_name, quantity, unit_price, subtotal FROM order_items WHERE order_id = $1`, oID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.OrderItem
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ItemName, &it.Quantity, &it.UnitPrice, &it.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *PostgresRepository) UpdateOrder(ctx context.Context, o *domain.Order) error {
	sql := `UPDATE orders SET order_no = $1, status = $2, updated_at = NOW() WHERE id = $3`
	ct, err := p.pool.Exec(ctx, sql, o.OrderNo, o.Status, o.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (p *PostgresRepository) ListOrders(ctx context.Context, status string, limit, offset int) ([]domain.Order, int, error) {
	var total int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE $1 = '' OR status = $1`, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	sql := `SELECT id, order_no, customer_id, invoice_id, total_amount, status, COALESCE(idempotency_key, ''), created_at, updated_at FROM orders WHERE $1 = '' OR status = $1 ORDER BY id DESC LIMIT $2 OFFSET $3`
	rows, err := p.pool.Query(ctx, sql, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	res, err := p.scanOrders(rows)
	return res, total, err
}

func (p *PostgresRepository) scanOrders(rows pgx.Rows) ([]domain.Order, error) {
	var list []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.OrderNo, &o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, nil
}

func (p *PostgresRepository) AddOrderHistory(ctx context.Context, h *domain.OrderStatusHistory) error {
	sql := `INSERT INTO order_status_histories (order_id, from_status, to_status, notes, created_at) VALUES ($1, $2, $3, $4, NOW()) RETURNING id, created_at`
	return p.pool.QueryRow(ctx, sql, h.OrderID, h.FromStatus, h.ToStatus, h.Reason).Scan(&h.ID, &h.CreatedAt)
}

func (p *PostgresRepository) GetOrderHistories(ctx context.Context, orderID int) ([]domain.OrderStatusHistory, error) {
	sql := `SELECT id, order_id, COALESCE(from_status, ''), to_status, COALESCE(notes, ''), created_at FROM order_status_histories WHERE order_id = $1 ORDER BY created_at ASC`
	rows, err := p.pool.Query(ctx, sql, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.OrderStatusHistory
	for rows.Next() {
		var h domain.OrderStatusHistory
		if err := rows.Scan(&h.ID, &h.OrderID, &h.FromStatus, &h.ToStatus, &h.Reason, &h.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}
	return list, nil
}
