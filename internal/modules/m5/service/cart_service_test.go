package service

import (
	"testing"

	"araya-business-hub/be/internal/modules/m5/domain"
)

func TestSamePartner(t *testing.T) {
	partnerA, partnerB := 10, 20
	tests := []struct {
		name string
		a, b *int
		want bool
	}{
		{name: "same partner", a: &partnerA, b: &partnerA, want: true},
		{name: "different partners", a: &partnerA, b: &partnerB, want: false},
		{name: "both platform", a: nil, b: nil, want: true},
		{name: "platform and partner", a: nil, b: &partnerA, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := samePartner(tt.a, tt.b); got != tt.want {
				t.Fatalf("samePartner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToCartResponseAlwaysReturnsItemsArray(t *testing.T) {
	got := toCartResponse(&domain.Cart{ID: 1, SessionID: "session"})
	if got.Items == nil {
		t.Fatal("expected empty non-nil items slice")
	}
	if len(got.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(got.Items))
	}
}
