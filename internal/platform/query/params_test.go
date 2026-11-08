package query_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/platform/query"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestParsePagination_Defaults(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodGet, "/items", nil)

	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		t.Fatalf("unexpected error: %v", appErr)
	}

	if p.Page != 1 || p.PerPage != 20 || p.SortDir != "desc" {
		t.Fatalf("expected defaults (1, 20, desc), got %+v", p)
	}
}

func TestParsePagination_ExceedMaxLimit(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodGet, "/items?per_page=150", nil)

	_, appErr := query.ParsePagination(c)
	if appErr == nil {
		t.Fatalf("expected error when per_page exceeds 100, got nil")
	}

	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", appErr.StatusCode)
	}
}

func TestParsePagination_InvalidPage(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodGet, "/items?page=0", nil)

	_, appErr := query.ParsePagination(c)
	if appErr == nil {
		t.Fatalf("expected error when page is 0, got nil")
	}
}

func TestParseDateRange_ValidAndInvalid(t *testing.T) {
	t.Run("Valid date range", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest(http.MethodGet, "/items?from=2026-10-01&to=2026-10-31", nil)

		dr, appErr := query.ParseDateRange(c)
		if appErr != nil {
			t.Fatalf("unexpected error: %v", appErr)
		}
		if dr.From == nil || dr.To == nil {
			t.Fatalf("expected non-nil dates")
		}
	})

	t.Run("From after To", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest(http.MethodGet, "/items?from=2026-11-01&to=2026-10-01", nil)

		_, appErr := query.ParseDateRange(c)
		if appErr == nil {
			t.Fatalf("expected error when from > to")
		}
	})
}
