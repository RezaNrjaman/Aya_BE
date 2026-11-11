package repository

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m4/domain"
	"github.com/jackc/pgx/v5"
)

func (p *PostgresRepository) CreateOrder(ctx context.Context, o *domain.Order) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sql := `INSERT INTO orders (order_no, partner_id, customer_id, invoice_id, total_amount, status, idempotency_key, created_at, updated_at)
	        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING id, created_at, updated_at`
	if err := tx.QueryRow(ctx, sql, o.OrderNo, o.PartnerID, o.CustomerID, o.InvoiceID, o.TotalAmount, o.Status, o.IdempotencyKey).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return err
	}
	itemQ := `INSERT INTO order_items (order_id, product_id, item_name, quantity, unit_price, subtotal) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	for i := range o.Items {
		o.Items[i].OrderID = o.ID
		if err := tx.QueryRow(ctx, itemQ, o.ID, o.Items[i].ProductID, o.Items[i].ItemName, o.Items[i].Quantity, o.Items[i].UnitPrice, o.Items[i].Subtotal).Scan(&o.Items[i].ID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (p *PostgresRepository) CreateOrderWithInvoice(ctx context.Context, o *domain.Order, inv *domain.Invoice) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Reserve physical inventory atomically. Services do not consume stock.
	for _, item := range o.Items {
		var productType string
		var active bool
		var stock int
		err := tx.QueryRow(ctx, `SELECT COALESCE(product_type, 'barang'), is_active, stock
			FROM products WHERE id = $1 FOR UPDATE`, item.ProductID).Scan(&productType, &active, &stock)
		if err != nil {
			return err
		}
		if !active {
			return domain.ErrProductNotFound
		}
		if productType != "jasa" {
			if stock < item.Quantity {
				return errors.New("stok produk tidak mencukupi")
			}
			if _, err := tx.Exec(ctx, `UPDATE products SET stock = stock - $1, updated_at = NOW()
				WHERE id = $2`, item.Quantity, item.ProductID); err != nil {
				return err
			}
		}
	}

	err = tx.QueryRow(ctx, `INSERT INTO orders
		(order_no, partner_id, customer_id, total_amount, status, idempotency_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW()) RETURNING id, created_at, updated_at`,
		o.OrderNo, o.PartnerID, o.CustomerID, o.TotalAmount, o.Status, o.IdempotencyKey).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return err
	}

	itemQ := `INSERT INTO order_items (order_id, product_id, item_name, quantity, unit_price, subtotal)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`
	for i := range o.Items {
		o.Items[i].OrderID = o.ID
		if err := tx.QueryRow(ctx, itemQ, o.ID, o.Items[i].ProductID, o.Items[i].ItemName,
			o.Items[i].Quantity, o.Items[i].UnitPrice, o.Items[i].Subtotal).Scan(&o.Items[i].ID); err != nil {
			return err
		}
	}

	err = tx.QueryRow(ctx, `INSERT INTO invoices
		(invoice_no, partner_id, customer_id, total_amount, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,NOW(),NOW()) RETURNING id, created_at, updated_at`,
		inv.InvoiceNo, inv.PartnerID, inv.CustomerID, inv.TotalAmount, inv.Status).
		Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return err
	}

	invItemQ := `INSERT INTO invoice_items (invoice_id, product_id, item_name, quantity, unit_price, subtotal)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`
	for i := range inv.Items {
		inv.Items[i].InvoiceID = inv.ID
		if err := tx.QueryRow(ctx, invItemQ, inv.ID, inv.Items[i].ProductID, inv.Items[i].ItemName,
			inv.Items[i].Quantity, inv.Items[i].UnitPrice, inv.Items[i].Subtotal).Scan(&inv.Items[i].ID); err != nil {
			return err
		}
	}
	o.InvoiceID = &inv.ID
	if _, err := tx.Exec(ctx, `UPDATE orders SET invoice_id = $1 WHERE id = $2`, inv.ID, o.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *PostgresRepository) GetOrderByID(ctx context.Context, id int) (*domain.Order, error) {
	sql := `SELECT id, order_no, partner_id, customer_id, invoice_id, total_amount, status, COALESCE(idempotency_key, ''), created_at, updated_at FROM orders WHERE id = $1`
	var o domain.Order
	if err := p.pool.QueryRow(ctx, sql, id).Scan(&o.ID, &o.OrderNo, &o.PartnerID, &o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := p.loadOrderItems(ctx, o.ID)
	o.Items = items
	return &o, err
}

func (p *PostgresRepository) GetOrderByOrderNoAndEmail(ctx context.Context, orderNo, email string) (*domain.Order, error) {
	q := `SELECT o.id, o.order_no, o.partner_id, o.customer_id, o.invoice_id, o.total_amount,
	             o.status, COALESCE(o.idempotency_key, ''), o.created_at, o.updated_at
	      FROM orders o JOIN customers c ON c.id = o.customer_id
	      WHERE o.order_no = $1 AND LOWER(c.email) = LOWER($2)`
	var o domain.Order
	if err := p.pool.QueryRow(ctx, q, orderNo, email).Scan(&o.ID, &o.OrderNo, &o.PartnerID,
		&o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := p.loadOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	histories, err := p.GetOrderHistories(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Histories = histories
	return &o, nil
}

func (p *PostgresRepository) GetOrderByIdempotency(ctx context.Context, key string) (*domain.Order, error) {
	sql := `SELECT id, order_no, partner_id, customer_id, invoice_id, total_amount, status, idempotency_key, created_at, updated_at FROM orders WHERE idempotency_key = $1`
	var o domain.Order
	if err := p.pool.QueryRow(ctx, sql, key).Scan(&o.ID, &o.OrderNo, &o.PartnerID, &o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
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
	sql := `UPDATE orders SET order_no = $1, status = $2, invoice_id = $3, updated_at = NOW() WHERE id = $4`
	ct, err := p.pool.Exec(ctx, sql, o.OrderNo, o.Status, o.InvoiceID, o.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (p *PostgresRepository) ListOrders(ctx context.Context, status string, partnerID *int, limit, offset int) ([]domain.Order, int, error) {
	var total int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE ($1 = '' OR status = $1) AND ($2::int IS NULL OR partner_id = $2)`, status, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	sql := `SELECT id, order_no, partner_id, customer_id, invoice_id, total_amount, status, COALESCE(idempotency_key, ''), created_at, updated_at FROM orders
	        WHERE ($1 = '' OR status = $1) AND ($2::int IS NULL OR partner_id = $2) ORDER BY id DESC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, sql, status, partnerID, limit, offset)
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
		if err := rows.Scan(&o.ID, &o.OrderNo, &o.PartnerID, &o.CustomerID, &o.InvoiceID, &o.TotalAmount, &o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
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
