package domain

import (
	"errors"
	"time"
)

var (
	ErrCartEmpty        = errors.New("keranjang belanja kosong")
	ErrCartItemNotFound = errors.New("item tidak ditemukan di keranjang")
	ErrInvalidQuantity  = errors.New("jumlah item minimal 1")
)

type CartItem struct {
	ID        int       `json:"id"`
	CartID    int       `json:"cart_id"`
	ProductID int       `json:"product_id"`
	ItemName  string    `json:"item_name"`
	Quantity  int       `json:"quantity"`
	UnitPrice int64     `json:"unit_price"`
	Subtotal  int64     `json:"subtotal"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Cart struct {
	ID         int        `json:"id"`
	SessionID  string     `json:"session_id"`
	CustomerID *int       `json:"customer_id,omitempty"`
	Items      []CartItem `json:"items"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (c *Cart) AddItem(productID int, name string, unitPrice int64, qty int) {
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			c.Items[i].Quantity += qty
			c.Items[i].Subtotal = int64(c.Items[i].Quantity) * c.Items[i].UnitPrice
			c.Items[i].UpdatedAt = time.Now().UTC()
			c.UpdatedAt = time.Now().UTC()
			return
		}
	}
	now := time.Now().UTC()
	c.Items = append(c.Items, CartItem{
		CartID: c.ID, ProductID: productID, ItemName: name,
		Quantity: qty, UnitPrice: unitPrice, Subtotal: int64(qty) * unitPrice,
		CreatedAt: now, UpdatedAt: now,
	})
	c.UpdatedAt = now
}

func (c *Cart) UpdateQuantity(productID int, qty int) error {
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			c.Items[i].Quantity = qty
			c.Items[i].Subtotal = int64(qty) * c.Items[i].UnitPrice
			c.Items[i].UpdatedAt = time.Now().UTC()
			c.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrCartItemNotFound
}

func (c *Cart) RemoveItem(productID int) bool {
	for i := range c.Items {
		if c.Items[i].ProductID == productID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.UpdatedAt = time.Now().UTC()
			return true
		}
	}
	return false
}

func (c *Cart) Clear() {
	c.Items = []CartItem{}
	c.UpdatedAt = time.Now().UTC()
}

func (c *Cart) TotalAmount() int64 {
	var total int64
	for _, it := range c.Items {
		total += it.Subtotal
	}
	return total
}

func (c *Cart) TotalQuantity() int {
	var count int
	for _, it := range c.Items {
		count += it.Quantity
	}
	return count
}
