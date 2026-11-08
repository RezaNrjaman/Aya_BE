package client

import (
	"context"

	m2Service "araya-business-hub/be/internal/modules/m2/service"
	m4Dto "araya-business-hub/be/internal/modules/m4/dto"
	m4Service "araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
)

type InternalM2Adapter struct {
	unitSvc *m2Service.UnitService
}

func NewInternalM2Adapter(svc *m2Service.UnitService) *InternalM2Adapter {
	return &InternalM2Adapter{unitSvc: svc}
}

func (a *InternalM2Adapter) GetActiveBusinessUnits(ctx context.Context) ([]dto.BusinessUnitSummary, error) {
	units, err := a.unitSvc.ListActiveUnits(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]dto.BusinessUnitSummary, 0, len(units))
	for _, u := range units {
		res = append(res, dto.BusinessUnitSummary{
			ID: u.ID, Name: u.Name, Category: u.CategoryName,
			Description: u.Description, LogoURL: u.LogoURL,
		})
	}
	return res, nil
}

type InternalM4Adapter struct {
	prodSvc *m4Service.ProductService
	ordSvc  *m4Service.OrderService
}

func NewInternalM4Adapter(prodSvc *m4Service.ProductService, ordSvc *m4Service.OrderService) *InternalM4Adapter {
	return &InternalM4Adapter{prodSvc: prodSvc, ordSvc: ordSvc}
}

func (a *InternalM4Adapter) GetActiveProducts(ctx context.Context, unitID *int, cat, search string) ([]dto.ProductSummary, error) {
	prods, _, err := a.prodSvc.ListActiveProducts(ctx, unitID, cat, search, 100, 0)
	if err != nil {
		return nil, err
	}
	res := make([]dto.ProductSummary, 0, len(prods))
	for _, p := range prods {
		res = append(res, dto.ProductSummary{
			ID: p.ID, UnitUsahaID: p.UnitUsahaID, Name: p.Name,
			Category: p.Category, Price: p.Price, Stock: p.Stock, IsActive: p.IsActive,
		})
	}
	return res, nil
}

func (a *InternalM4Adapter) GetProductByID(ctx context.Context, id int) (*dto.ProductSummary, error) {
	p, err := a.prodSvc.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	return &dto.ProductSummary{
		ID: p.ID, UnitUsahaID: p.UnitUsahaID, Name: p.Name,
		Category: p.Category, Price: p.Price, Stock: p.Stock, IsActive: p.IsActive,
	}, nil
}

func mapCartItemsToM4(items []domain.CartItem) []m4Dto.OrderItemInput {
	m4Items := make([]m4Dto.OrderItemInput, 0, len(items))
	for _, it := range items {
		m4Items = append(m4Items, m4Dto.OrderItemInput{
			ProductID: it.ProductID, Quantity: it.Quantity,
		})
	}
	return m4Items
}

func (a *InternalM4Adapter) CreateOrderFromCart(ctx context.Context, req dto.CheckoutRequest, items []domain.CartItem) (*dto.CheckoutResponse, error) {
	m4Req := m4Dto.CreateOrderRequest{
		CustomerID:     req.CustomerID,
		IdempotencyKey: req.IdempotencyKey,
		Items:          mapCartItemsToM4(items),
	}
	ord, err := a.ordSvc.CreateOrder(ctx, m4Req, "customer.portal")
	if err != nil {
		return nil, err
	}
	return &dto.CheckoutResponse{
		OrderID: ord.ID, OrderNo: ord.OrderNo,
		TotalAmount: ord.TotalAmount, Status: ord.Status,
		CartCleared: true, IdempotencyKey: ord.IdempotencyKey,
	}, nil
}
