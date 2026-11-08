package repository

import (
	"context"

	"araya-business-hub/be/internal/modules/m4/domain"
)

type Repository interface {
	// Products
	CreateProduct(ctx context.Context, p *domain.Product) error
	GetProductByID(ctx context.Context, id int) (*domain.Product, error)
	UpdateProduct(ctx context.Context, p *domain.Product) error
	ListProducts(ctx context.Context, unitID *int, category, search string, activeOnly bool, limit, offset int) ([]domain.Product, int, error)

	// Customers
	CreateCustomer(ctx context.Context, c *domain.Customer) error
	GetCustomerByID(ctx context.Context, id int) (*domain.Customer, error)
	ListCustomers(ctx context.Context, search string, limit, offset int) ([]domain.Customer, int, error)

	// Quotations
	CreateQuotation(ctx context.Context, q *domain.Quotation) error
	GetQuotationByID(ctx context.Context, id int) (*domain.Quotation, error)
	UpdateQuotation(ctx context.Context, q *domain.Quotation) error
	ListQuotations(ctx context.Context, status string, limit, offset int) ([]domain.Quotation, int, error)

	// Invoices
	CreateInvoice(ctx context.Context, inv *domain.Invoice) error
	GetInvoiceByID(ctx context.Context, id int) (*domain.Invoice, error)
	UpdateInvoice(ctx context.Context, inv *domain.Invoice) error
	ListInvoices(ctx context.Context, status string, limit, offset int) ([]domain.Invoice, int, error)

	// Payments
	CreatePayment(ctx context.Context, p *domain.Payment) error
	GetPaymentByReference(ctx context.Context, ref string) (*domain.Payment, error)
	UpdatePayment(ctx context.Context, p *domain.Payment) error
	ListPayments(ctx context.Context, invoiceID int) ([]domain.Payment, error)

	// Orders
	CreateOrder(ctx context.Context, o *domain.Order) error
	GetOrderByID(ctx context.Context, id int) (*domain.Order, error)
	GetOrderByIdempotency(ctx context.Context, key string) (*domain.Order, error)
	UpdateOrder(ctx context.Context, o *domain.Order) error
	ListOrders(ctx context.Context, status string, limit, offset int) ([]domain.Order, int, error)
	AddOrderHistory(ctx context.Context, h *domain.OrderStatusHistory) error
	GetOrderHistories(ctx context.Context, orderID int) ([]domain.OrderStatusHistory, error)
}
