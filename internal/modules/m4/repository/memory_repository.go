package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"araya-business-hub/be/internal/modules/m4/domain"
)

type InMemoryRepository struct {
	mu             sync.RWMutex
	products       map[int]domain.Product
	customers      map[int]domain.Customer
	quotations     map[int]domain.Quotation
	invoices       map[int]domain.Invoice
	payments       map[int]domain.Payment
	orders         map[int]domain.Order
	orderHistories []domain.OrderStatusHistory

	nextProdID int
	nextCustID int
	nextQuotID int
	nextInvID  int
	nextPayID  int
	nextOrdID  int
	nextHistID int
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		products:   make(map[int]domain.Product),
		customers:  make(map[int]domain.Customer),
		quotations: make(map[int]domain.Quotation),
		invoices:   make(map[int]domain.Invoice),
		payments:   make(map[int]domain.Payment),
		orders:     make(map[int]domain.Order),
		nextProdID: 1, nextCustID: 1, nextQuotID: 1,
		nextInvID: 1, nextPayID: 1, nextOrdID: 1, nextHistID: 1,
	}
}

func (r *InMemoryRepository) CreateProduct(ctx context.Context, p *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.nextProdID
	r.nextProdID++
	now := time.Now().UTC()
	p.CreatedAt, p.UpdatedAt = now, now
	r.products[p.ID] = *p
	return nil
}

func (r *InMemoryRepository) GetProductByID(ctx context.Context, id int) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return &p, nil
}

func (r *InMemoryRepository) UpdateProduct(ctx context.Context, p *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	curr, ok := r.products[p.ID]
	if !ok {
		return domain.ErrProductNotFound
	}
	p.CreatedAt = curr.CreatedAt
	p.UpdatedAt = time.Now().UTC()
	r.products[p.ID] = *p
	return nil
}

func matchesProduct(p domain.Product, unitID *int, category, search string, activeOnly bool, partnerID *int) bool {
	if activeOnly && !p.IsActive {
		return false
	}
	if unitID != nil && p.UnitUsahaID != *unitID {
		return false
	}
	if partnerID != nil && (p.PartnerID == nil || *p.PartnerID != *partnerID) {
		return false
	}
	if category != "" && !strings.EqualFold(p.Category, category) {
		return false
	}
	if search == "" {
		return true
	}
	s := strings.ToLower(search)
	return strings.Contains(strings.ToLower(p.Name), s) || strings.Contains(strings.ToLower(p.Description), s)
}

func (r *InMemoryRepository) ValidateUnitOwnership(ctx context.Context, unitID int, partnerID *int) error {
	// The in-memory M4 fixture does not own M2 unit records. PostgreSQL is the
	// authoritative production validator; service tests exercise scope on IDs.
	return nil
}

func (r *InMemoryRepository) ListProducts(ctx context.Context, unitID *int, category, search string, activeOnly bool, partnerID *int, limit, offset int) ([]domain.Product, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Product, 0)
	for _, p := range r.products {
		if matchesProduct(p, unitID, category, search, activeOnly, partnerID) {
			res = append(res, p)
		}
	}
	total := len(res)
	if offset >= total {
		return []domain.Product{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return res[offset:end], total, nil
}

func (r *InMemoryRepository) CreateCustomer(ctx context.Context, c *domain.Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c.ID = r.nextCustID
	r.nextCustID++
	now := time.Now().UTC()
	c.CreatedAt, c.UpdatedAt = now, now
	r.customers[c.ID] = *c
	return nil
}

func (r *InMemoryRepository) GetCustomerByID(ctx context.Context, id int) (*domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return &c, nil
}

func (r *InMemoryRepository) GetCustomerByEmail(ctx context.Context, email string, partnerID *int) (*domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, customer := range r.customers {
		if !strings.EqualFold(strings.TrimSpace(customer.Email), strings.TrimSpace(email)) {
			continue
		}
		if partnerID == nil && customer.PartnerID != nil {
			continue
		}
		if partnerID != nil && (customer.PartnerID == nil || *customer.PartnerID != *partnerID) {
			continue
		}
		copy := customer
		return &copy, nil
	}
	return nil, domain.ErrCustomerNotFound
}

func (r *InMemoryRepository) ListCustomers(ctx context.Context, search string, partnerID *int, limit, offset int) ([]domain.Customer, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Customer, 0)
	s := strings.ToLower(search)
	for _, c := range r.customers {
		if partnerID != nil && (c.PartnerID == nil || *c.PartnerID != *partnerID) {
			continue
		}
		if s == "" || strings.Contains(strings.ToLower(c.Name), s) || strings.Contains(strings.ToLower(c.Email), s) {
			res = append(res, c)
		}
	}
	total := len(res)
	if offset >= total {
		return []domain.Customer{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return res[offset:end], total, nil
}

func (r *InMemoryRepository) CreateQuotation(ctx context.Context, q *domain.Quotation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	q.ID = r.nextQuotID
	r.nextQuotID++
	now := time.Now().UTC()
	q.CreatedAt, q.UpdatedAt = now, now
	r.quotations[q.ID] = *q
	return nil
}

func (r *InMemoryRepository) GetQuotationByID(ctx context.Context, id int) (*domain.Quotation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q, ok := r.quotations[id]
	if !ok {
		return nil, domain.ErrQuotationNotFound
	}
	return &q, nil
}

func (r *InMemoryRepository) UpdateQuotation(ctx context.Context, q *domain.Quotation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	curr, ok := r.quotations[q.ID]
	if !ok {
		return domain.ErrQuotationNotFound
	}
	q.CreatedAt = curr.CreatedAt
	q.UpdatedAt = time.Now().UTC()
	r.quotations[q.ID] = *q
	return nil
}

func (r *InMemoryRepository) ListQuotations(ctx context.Context, status string, partnerID *int, limit, offset int) ([]domain.Quotation, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Quotation, 0)
	for _, q := range r.quotations {
		if partnerID != nil && (q.PartnerID == nil || *q.PartnerID != *partnerID) {
			continue
		}
		if status == "" || strings.EqualFold(q.Status, status) {
			res = append(res, q)
		}
	}
	total := len(res)
	if offset >= total {
		return []domain.Quotation{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return res[offset:end], total, nil
}

func (r *InMemoryRepository) CreateInvoice(ctx context.Context, inv *domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv.ID = r.nextInvID
	r.nextInvID++
	now := time.Now().UTC()
	inv.CreatedAt, inv.UpdatedAt = now, now
	r.invoices[inv.ID] = *inv
	return nil
}

func (r *InMemoryRepository) GetInvoiceByID(ctx context.Context, id int) (*domain.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inv, ok := r.invoices[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	return &inv, nil
}

func (r *InMemoryRepository) UpdateInvoice(ctx context.Context, inv *domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	curr, ok := r.invoices[inv.ID]
	if !ok {
		return domain.ErrInvoiceNotFound
	}
	inv.CreatedAt = curr.CreatedAt
	inv.UpdatedAt = time.Now().UTC()
	r.invoices[inv.ID] = *inv
	return nil
}

func (r *InMemoryRepository) ListInvoices(ctx context.Context, status string, partnerID *int, limit, offset int) ([]domain.Invoice, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Invoice, 0)
	for _, inv := range r.invoices {
		if partnerID != nil && (inv.PartnerID == nil || *inv.PartnerID != *partnerID) {
			continue
		}
		if status == "" || strings.EqualFold(inv.Status, status) {
			res = append(res, inv)
		}
	}
	total := len(res)
	if offset >= total {
		return []domain.Invoice{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return res[offset:end], total, nil
}

func (r *InMemoryRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.nextPayID
	r.nextPayID++
	now := time.Now().UTC()
	p.CreatedAt, p.UpdatedAt = now, now
	r.payments[p.ID] = *p
	return nil
}

func (r *InMemoryRepository) GetPaymentByReference(ctx context.Context, ref string) (*domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.payments {
		if p.PaymentReference == ref {
			return &p, nil
		}
	}
	return nil, errors.New("pembayaran tidak ditemukan")
}

func (r *InMemoryRepository) UpdatePayment(ctx context.Context, p *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.UpdatedAt = time.Now().UTC()
	r.payments[p.ID] = *p
	return nil
}

func (r *InMemoryRepository) ListPayments(ctx context.Context, invoiceID int) ([]domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Payment, 0)
	for _, p := range r.payments {
		if p.InvoiceID == invoiceID {
			res = append(res, p)
		}
	}
	return res, nil
}

func (r *InMemoryRepository) CreateOrder(ctx context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o.ID = r.nextOrdID
	r.nextOrdID++
	now := time.Now().UTC()
	o.CreatedAt, o.UpdatedAt = now, now
	r.orders[o.ID] = *o
	return nil
}

func (r *InMemoryRepository) CreateOrderWithInvoice(ctx context.Context, o *domain.Order, inv *domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range o.Items {
		prod, ok := r.products[item.ProductID]
		if !ok || !prod.IsActive {
			return domain.ErrProductNotFound
		}
		if prod.ProductType != "jasa" {
			if prod.Stock < item.Quantity {
				return errors.New("stok produk tidak mencukupi")
			}
			prod.Stock -= item.Quantity
			r.products[prod.ID] = prod
		}
	}
	now := time.Now().UTC()
	o.ID = r.nextOrdID
	r.nextOrdID++
	o.CreatedAt, o.UpdatedAt = now, now
	inv.ID = r.nextInvID
	r.nextInvID++
	inv.CreatedAt, inv.UpdatedAt = now, now
	o.InvoiceID = &inv.ID
	r.orders[o.ID] = *o
	r.invoices[inv.ID] = *inv
	return nil
}

func (r *InMemoryRepository) GetOrderByID(ctx context.Context, id int) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return &o, nil
}

func (r *InMemoryRepository) GetOrderByOrderNoAndEmail(ctx context.Context, orderNo, email string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.orders {
		if o.OrderNo != orderNo {
			continue
		}
		customer, ok := r.customers[o.CustomerID]
		if ok && strings.EqualFold(strings.TrimSpace(customer.Email), strings.TrimSpace(email)) {
			return &o, nil
		}
	}
	return nil, domain.ErrOrderNotFound
}

func (r *InMemoryRepository) GetOrderByIdempotency(ctx context.Context, key string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if key == "" {
		return nil, nil
	}
	for _, o := range r.orders {
		if o.IdempotencyKey == key {
			return &o, nil
		}
	}
	return nil, nil
}

func (r *InMemoryRepository) UpdateOrder(ctx context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	curr, ok := r.orders[o.ID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.CreatedAt = curr.CreatedAt
	o.UpdatedAt = time.Now().UTC()
	r.orders[o.ID] = *o
	return nil
}

func (r *InMemoryRepository) ListOrders(ctx context.Context, status string, partnerID *int, limit, offset int) ([]domain.Order, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Order, 0)
	for _, o := range r.orders {
		if partnerID != nil && (o.PartnerID == nil || *o.PartnerID != *partnerID) {
			continue
		}
		if status == "" || strings.EqualFold(o.Status, status) {
			res = append(res, o)
		}
	}
	total := len(res)
	if offset >= total {
		return []domain.Order{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return res[offset:end], total, nil
}

func (r *InMemoryRepository) AddOrderHistory(ctx context.Context, h *domain.OrderStatusHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	h.ID = r.nextHistID
	r.nextHistID++
	h.CreatedAt = time.Now().UTC()
	r.orderHistories = append(r.orderHistories, *h)
	return nil
}

func (r *InMemoryRepository) GetOrderHistories(ctx context.Context, orderID int) ([]domain.OrderStatusHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.OrderStatusHistory, 0)
	for _, h := range r.orderHistories {
		if h.OrderID == orderID {
			res = append(res, h)
		}
	}
	return res, nil
}
