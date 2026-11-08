package test

import (
	"context"
	"testing"

	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
)

// MockM2Provider implements client.M2Client with explicit [MOCK] labeling.
type MockM2Provider struct {
	ActiveUnits []dto.BusinessUnitSummary
}

func (m *MockM2Provider) GetActiveBusinessUnits(_ context.Context) ([]dto.BusinessUnitSummary, error) {
	return m.ActiveUnits, nil
}

// MockM4Provider implements client.M4Client with explicit [MOCK] labeling.
type MockM4Provider struct {
	ActiveProducts []dto.ProductSummary
	CreatedOrderID int
	CreatedOrderNo string
}

func (m *MockM4Provider) GetActiveProducts(_ context.Context, _ *int, _, _ string) ([]dto.ProductSummary, error) {
	return m.ActiveProducts, nil
}

func (m *MockM4Provider) GetProductByID(_ context.Context, id int) (*dto.ProductSummary, error) {
	for _, p := range m.ActiveProducts {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *MockM4Provider) CreateOrderFromCart(_ context.Context, _ dto.CheckoutRequest, _ []domain.CartItem) (*dto.CheckoutResponse, error) {
	return &dto.CheckoutResponse{
		OrderID:        m.CreatedOrderID,
		OrderNo:        m.CreatedOrderNo,
		TotalAmount:    150000,
		Status:         "baru",
		CartCleared:    true,
		IdempotencyKey: "mock-idempotency-key",
	}, nil
}

func TestMock_M2Provider_ContractCompliance(t *testing.T) {
	mockM2 := &MockM2Provider{
		ActiveUnits: []dto.BusinessUnitSummary{
			{ID: 101, Name: "[MOCK] Unit Pertanian Araya", Category: "Agrikultur"},
		},
	}
	units, err := mockM2.GetActiveBusinessUnits(context.Background())
	if err != nil || len(units) != 1 || units[0].ID != 101 {
		t.Fatalf("mock M2 provider failed contract compliance: %v", err)
	}
}

func TestMock_M4Provider_OrderCreation(t *testing.T) {
	mockM4 := &MockM4Provider{
		CreatedOrderID: 501,
		CreatedOrderNo: "ORD-MOCK-2026-001",
	}
	res, err := mockM4.CreateOrderFromCart(context.Background(), dto.CheckoutRequest{CustomerID: 1}, nil)
	if err != nil || res.OrderID != 501 || res.OrderNo != "ORD-MOCK-2026-001" {
		t.Fatalf("mock M4 provider order creation failed: %v", err)
	}
}
