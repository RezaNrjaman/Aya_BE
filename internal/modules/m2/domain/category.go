package domain

import (
	"errors"
	"strings"
	"time"
)

type BusinessCategory struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *BusinessCategory) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return errors.New("nama kategori wajib diisi")
	}
	return nil
}
