package repository

import (
	"context"
	"time"

	m2dto "araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m4/domain"
)

func (p *PostgresRepository) GetDashboardData(ctx context.Context, partnerID, unitID *int, from, to *time.Time) (m2dto.DashboardMetrics, []m2dto.OrderSummaryItem, error) {
	var metrics m2dto.DashboardMetrics

	revenueQ := `SELECT COALESCE(SUM(i.total_amount), 0)
		FROM invoices i
		WHERE i.status = $5
		  AND ($1::int IS NULL OR i.partner_id = $1)
		  AND ($2::int IS NULL
		       OR EXISTS (SELECT 1 FROM quotations q WHERE q.id = i.quotation_id AND q.unit_usaha_id = $2)
		       OR EXISTS (
		          SELECT 1 FROM invoice_items ii
		          JOIN products pr ON pr.id = ii.product_id
		          WHERE ii.invoice_id = i.id AND pr.unit_usaha_id = $2
		       ))
		  AND ($3::date IS NULL OR i.created_at >= $3::date)
		  AND ($4::date IS NULL OR i.created_at < $4::date + INTERVAL '1 day')`
	if err := p.pool.QueryRow(ctx, revenueQ, partnerID, unitID, from, to, domain.InvoicePaid).Scan(&metrics.TotalRevenue); err != nil {
		return metrics, nil, err
	}

	ordersQ := `SELECT COUNT(DISTINCT o.id)
		FROM orders o
		WHERE ($1::int IS NULL OR o.partner_id = $1)
		  AND ($2::int IS NULL OR EXISTS (
		      SELECT 1 FROM order_items oi
		      JOIN products pr ON pr.id = oi.product_id
		      WHERE oi.order_id = o.id AND pr.unit_usaha_id = $2
		  ))
		  AND ($3::date IS NULL OR o.created_at >= $3::date)
		  AND ($4::date IS NULL OR o.created_at < $4::date + INTERVAL '1 day')`
	if err := p.pool.QueryRow(ctx, ordersQ, partnerID, unitID, from, to).Scan(&metrics.TotalOrders); err != nil {
		return metrics, nil, err
	}

	productsQ := `SELECT COUNT(*)
		FROM products p
		WHERE p.is_active = TRUE
		  AND ($1::int IS NULL OR p.partner_id = $1)
		  AND ($2::int IS NULL OR p.unit_usaha_id = $2)`
	if err := p.pool.QueryRow(ctx, productsQ, partnerID, unitID).Scan(&metrics.ActiveProducts); err != nil {
		return metrics, nil, err
	}

	recentQ := `SELECT o.order_no,
		       COALESCE((
		          SELECT string_agg(DISTINCT bu.name, ', ')
		          FROM order_items oi
		          JOIN products pr ON pr.id = oi.product_id
		          JOIN business_units bu ON bu.id = pr.unit_usaha_id
		          WHERE oi.order_id = o.id
		       ), '') AS unit_name,
		       o.total_amount, o.status, o.created_at::text
		FROM orders o
		WHERE ($1::int IS NULL OR o.partner_id = $1)
		  AND ($2::int IS NULL OR EXISTS (
		      SELECT 1 FROM order_items oi
		      JOIN products pr ON pr.id = oi.product_id
		      WHERE oi.order_id = o.id AND pr.unit_usaha_id = $2
		  ))
		  AND ($3::date IS NULL OR o.created_at >= $3::date)
		  AND ($4::date IS NULL OR o.created_at < $4::date + INTERVAL '1 day')
		ORDER BY o.created_at DESC LIMIT 5`
	rows, err := p.pool.Query(ctx, recentQ, partnerID, unitID, from, to)
	if err != nil {
		return metrics, nil, err
	}
	defer rows.Close()
	recent := make([]m2dto.OrderSummaryItem, 0, 5)
	for rows.Next() {
		var item m2dto.OrderSummaryItem
		if err := rows.Scan(&item.OrderNo, &item.UnitName, &item.TotalAmount, &item.Status, &item.CreatedAt); err != nil {
			return metrics, nil, err
		}
		recent = append(recent, item)
	}
	if err := rows.Err(); err != nil {
		return metrics, nil, err
	}
	return metrics, recent, nil
}

func (r *InMemoryRepository) GetDashboardData(ctx context.Context, partnerID, unitID *int, from, to *time.Time) (m2dto.DashboardMetrics, []m2dto.OrderSummaryItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var metrics m2dto.DashboardMetrics
	matchesPartner := func(owner *int) bool {
		if partnerID == nil {
			return true
		}
		return owner != nil && *owner == *partnerID
	}
	inPeriod := func(created time.Time) bool {
		if from != nil && created.Before(*from) {
			return false
		}
		if to != nil && created.After(to.Add(23*time.Hour+59*time.Minute+59*time.Second)) {
			return false
		}
		return true
	}
	for _, product := range r.products {
		if product.IsActive && matchesPartner(product.PartnerID) && (unitID == nil || product.UnitUsahaID == *unitID) {
			metrics.ActiveProducts++
		}
	}
	for _, order := range r.orders {
		if !matchesPartner(order.PartnerID) || !inPeriod(order.CreatedAt) {
			continue
		}
		if unitID != nil {
			matchUnit := false
			for _, item := range order.Items {
				if product, ok := r.products[item.ProductID]; ok && product.UnitUsahaID == *unitID {
					matchUnit = true
					break
				}
			}
			if !matchUnit {
				continue
			}
		}
		metrics.TotalOrders++
	}
	for _, invoice := range r.invoices {
		if invoice.Status != domain.InvoicePaid || !matchesPartner(invoice.PartnerID) || !inPeriod(invoice.CreatedAt) {
			continue
		}
		if unitID != nil {
			matchUnit := false
			if invoice.QuotationID != nil {
				if quotation, ok := r.quotations[*invoice.QuotationID]; ok && quotation.UnitUsahaID == *unitID {
					matchUnit = true
				}
			}
			for _, item := range invoice.Items {
				if item.ProductID == nil {
					continue
				}
				if product, ok := r.products[*item.ProductID]; ok && product.UnitUsahaID == *unitID {
					matchUnit = true
					break
				}
			}
			if !matchUnit {
				continue
			}
		}
		metrics.TotalRevenue += invoice.TotalAmount
	}
	recent := make([]m2dto.OrderSummaryItem, 0, 5)
	for _, order := range r.orders {
		if !matchesPartner(order.PartnerID) || !inPeriod(order.CreatedAt) {
			continue
		}
		if unitID != nil {
			matchUnit := false
			for _, item := range order.Items {
				if product, ok := r.products[item.ProductID]; ok && product.UnitUsahaID == *unitID {
					matchUnit = true
					break
				}
			}
			if !matchUnit {
				continue
			}
		}
		recent = append(recent, m2dto.OrderSummaryItem{
			OrderNo: order.OrderNo, UnitName: "Unit usaha", TotalAmount: order.TotalAmount,
			Status: order.Status, CreatedAt: order.CreatedAt.Format(time.RFC3339),
		})
	}
	// Keep the in-memory dashboard deterministic and bounded like PostgreSQL.
	if len(recent) > 5 {
		recent = recent[:5]
	}
	return metrics, recent, nil
}
