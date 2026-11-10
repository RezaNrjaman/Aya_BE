package repository

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m4/domain"
	"github.com/jackc/pgx/v5"
)

func (p *PostgresRepository) CreateInvoice(ctx context.Context, inv *domain.Invoice) error {
	sql := `INSERT INTO invoices (invoice_no, quotation_id, partner_id, customer_id, total_amount, status, due_date, paid_at, created_at, updated_at)
	        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING id, created_at, updated_at`
	if err := p.pool.QueryRow(ctx, sql, inv.InvoiceNo, inv.QuotationID, inv.PartnerID, inv.CustomerID, inv.TotalAmount, inv.Status, inv.DueDate, inv.PaidAt).
		Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return err
	}
	return p.insertInvoiceItems(ctx, inv.ID, inv.Items)
}

func (p *PostgresRepository) insertInvoiceItems(ctx context.Context, invID int, items []domain.InvoiceItem) error {
	itemQ := `INSERT INTO invoice_items (invoice_id, product_id, item_name, quantity, unit_price, subtotal) VALUES ($1, $2, $3, $4, $5, $6)`
	for i := range items {
		items[i].InvoiceID = invID
		if err := p.pool.QueryRow(ctx, itemQ+` RETURNING id`, invID, items[i].ProductID, items[i].ItemName, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal).Scan(&items[i].ID); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresRepository) GetInvoiceByID(ctx context.Context, id int) (*domain.Invoice, error) {
	sql := `SELECT id, invoice_no, quotation_id, partner_id, customer_id, total_amount, status, due_date, paid_at, created_at, updated_at FROM invoices WHERE id = $1`
	var inv domain.Invoice
	if err := p.pool.QueryRow(ctx, sql, id).Scan(&inv.ID, &inv.InvoiceNo, &inv.QuotationID, &inv.PartnerID, &inv.CustomerID, &inv.TotalAmount, &inv.Status, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, err
	}
	items, err := p.loadInvoiceItems(ctx, inv.ID)
	inv.Items = items
	return &inv, err
}

func (p *PostgresRepository) loadInvoiceItems(ctx context.Context, invID int) ([]domain.InvoiceItem, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, invoice_id, product_id, item_name, quantity, unit_price, subtotal FROM invoice_items WHERE invoice_id = $1`, invID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.InvoiceItem
	for rows.Next() {
		var it domain.InvoiceItem
		if err := rows.Scan(&it.ID, &it.InvoiceID, &it.ProductID, &it.ItemName, &it.Quantity, &it.UnitPrice, &it.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *PostgresRepository) UpdateInvoice(ctx context.Context, inv *domain.Invoice) error {
	sql := `UPDATE invoices SET invoice_no = $1, status = $2, paid_at = $3, updated_at = NOW() WHERE id = $4`
	ct, err := p.pool.Exec(ctx, sql, inv.InvoiceNo, inv.Status, inv.PaidAt, inv.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrInvoiceNotFound
	}
	return nil
}

func (p *PostgresRepository) ListInvoices(ctx context.Context, status string, partnerID *int, limit, offset int) ([]domain.Invoice, int, error) {
	var total int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM invoices WHERE ($1 = '' OR status = $1) AND ($2::int IS NULL OR partner_id = $2)`, status, partnerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	sql := `SELECT id, invoice_no, quotation_id, partner_id, customer_id, total_amount, status, due_date, paid_at, created_at, updated_at FROM invoices
	        WHERE ($1 = '' OR status = $1) AND ($2::int IS NULL OR partner_id = $2) ORDER BY id DESC LIMIT $3 OFFSET $4`
	rows, err := p.pool.Query(ctx, sql, status, partnerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	res, err := p.scanInvoices(rows)
	return res, total, err
}

func (p *PostgresRepository) scanInvoices(rows pgx.Rows) ([]domain.Invoice, error) {
	var list []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		if err := rows.Scan(&inv.ID, &inv.InvoiceNo, &inv.QuotationID, &inv.PartnerID, &inv.CustomerID, &inv.TotalAmount, &inv.Status, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, inv)
	}
	return list, nil
}

func (p *PostgresRepository) CreatePayment(ctx context.Context, pay *domain.Payment) error {
	sql := `INSERT INTO payments (payment_reference, invoice_id, amount, method, status, created_at, updated_at)
	        VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, sql, pay.PaymentReference, pay.InvoiceID, pay.Amount, pay.Method, pay.Status).
		Scan(&pay.ID, &pay.CreatedAt, &pay.UpdatedAt)
}

func (p *PostgresRepository) GetPaymentByReference(ctx context.Context, ref string) (*domain.Payment, error) {
	sql := `SELECT id, payment_reference, invoice_id, amount, method, status, created_at, updated_at FROM payments WHERE payment_reference = $1`
	var pay domain.Payment
	err := p.pool.QueryRow(ctx, sql, ref).Scan(&pay.ID, &pay.PaymentReference, &pay.InvoiceID, &pay.Amount, &pay.Method, &pay.Status, &pay.CreatedAt, &pay.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return &pay, nil
}

func (p *PostgresRepository) UpdatePayment(ctx context.Context, pay *domain.Payment) error {
	sql := `UPDATE payments SET status = $1, updated_at = NOW() WHERE id = $2`
	ct, err := p.pool.Exec(ctx, sql, pay.Status, pay.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}
	return nil
}

func (p *PostgresRepository) ListPayments(ctx context.Context, invoiceID int) ([]domain.Payment, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, payment_reference, invoice_id, amount, method, status, created_at, updated_at FROM payments WHERE invoice_id = $1 ORDER BY id DESC`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Payment
	for rows.Next() {
		var pay domain.Payment
		if err := rows.Scan(&pay.ID, &pay.PaymentReference, &pay.InvoiceID, &pay.Amount, &pay.Method, &pay.Status, &pay.CreatedAt, &pay.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, pay)
	}
	return list, nil
}
