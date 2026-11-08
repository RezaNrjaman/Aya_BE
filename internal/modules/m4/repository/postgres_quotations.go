package repository

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m4/domain"
	"github.com/jackc/pgx/v5"
)

func (p *PostgresRepository) CreateQuotation(ctx context.Context, q *domain.Quotation) error {
	sql := `INSERT INTO quotations (quotation_no, unit_usaha_id, partner_id, customer_id, total_amount, status, valid_until, created_at, updated_at)
	        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW()) RETURNING id, created_at, updated_at`
	if err := p.pool.QueryRow(ctx, sql, q.QuotationNo, q.UnitUsahaID, q.PartnerID, q.CustomerID, q.TotalAmount, q.Status, q.ValidUntil).
		Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return err
	}
	return p.insertQuotationItems(ctx, q.ID, q.Items)
}

func (p *PostgresRepository) insertQuotationItems(ctx context.Context, qID int, items []domain.QuotationItem) error {
	itemQ := `INSERT INTO quotation_items (quotation_id, product_id, item_name, quantity, unit_price, subtotal) VALUES ($1, $2, $3, $4, $5, $6)`
	for i := range items {
		items[i].QuotationID = qID
		if err := p.pool.QueryRow(ctx, itemQ+` RETURNING id`, qID, items[i].ProductID, items[i].ItemName, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal).Scan(&items[i].ID); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresRepository) GetQuotationByID(ctx context.Context, id int) (*domain.Quotation, error) {
	sql := `SELECT id, quotation_no, unit_usaha_id, partner_id, customer_id, total_amount, status, valid_until, created_at, updated_at FROM quotations WHERE id = $1`
	var q domain.Quotation
	if err := p.pool.QueryRow(ctx, sql, id).Scan(&q.ID, &q.QuotationNo, &q.UnitUsahaID, &q.PartnerID, &q.CustomerID, &q.TotalAmount, &q.Status, &q.ValidUntil, &q.CreatedAt, &q.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrQuotationNotFound
		}
		return nil, err
	}
	items, err := p.loadQuotationItems(ctx, q.ID)
	q.Items = items
	return &q, err
}

func (p *PostgresRepository) loadQuotationItems(ctx context.Context, qID int) ([]domain.QuotationItem, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, quotation_id, product_id, item_name, quantity, unit_price, subtotal FROM quotation_items WHERE quotation_id = $1`, qID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.QuotationItem
	for rows.Next() {
		var it domain.QuotationItem
		if err := rows.Scan(&it.ID, &it.QuotationID, &it.ProductID, &it.ItemName, &it.Quantity, &it.UnitPrice, &it.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *PostgresRepository) UpdateQuotation(ctx context.Context, q *domain.Quotation) error {
	sql := `UPDATE quotations SET quotation_no = $1, status = $2, updated_at = NOW() WHERE id = $3`
	ct, err := p.pool.Exec(ctx, sql, q.QuotationNo, q.Status, q.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrQuotationNotFound
	}
	return nil
}

func (p *PostgresRepository) ListQuotations(ctx context.Context, status string, limit, offset int) ([]domain.Quotation, int, error) {
	var total int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM quotations WHERE $1 = '' OR status = $1`, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	sql := `SELECT id, quotation_no, unit_usaha_id, partner_id, customer_id, total_amount, status, valid_until, created_at, updated_at FROM quotations WHERE $1 = '' OR status = $1 ORDER BY id DESC LIMIT $2 OFFSET $3`
	rows, err := p.pool.Query(ctx, sql, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	res, err := p.scanQuotations(rows)
	return res, total, err
}

func (p *PostgresRepository) scanQuotations(rows pgx.Rows) ([]domain.Quotation, error) {
	var list []domain.Quotation
	for rows.Next() {
		var q domain.Quotation
		if err := rows.Scan(&q.ID, &q.QuotationNo, &q.UnitUsahaID, &q.PartnerID, &q.CustomerID, &q.TotalAmount, &q.Status, &q.ValidUntil, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, q)
	}
	return list, nil
}
