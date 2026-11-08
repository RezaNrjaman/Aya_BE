package query

import (
	"strconv"
	"strings"
	"time"

	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

// PaginationParams standardizes list query parameters across all modules.
type PaginationParams struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Search  string `json:"search,omitempty"`
	SortBy  string `json:"sort_by,omitempty"`
	SortDir string `json:"sort_dir,omitempty"` // "asc" or "desc"
}

// Offset calculates the database SQL OFFSET.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// ParsePagination extracts and validates page and per_page parameters from the request query.
func ParsePagination(c *gin.Context) (PaginationParams, *response.AppError) {
	params := PaginationParams{
		Page:    1,
		PerPage: 20,
		Search:  strings.TrimSpace(c.Query("search")),
		SortBy:  strings.TrimSpace(c.Query("sort_by")),
		SortDir: strings.ToLower(strings.TrimSpace(c.Query("sort_dir"))),
	}

	if params.SortDir == "" {
		params.SortDir = "desc"
	} else if params.SortDir != "asc" && params.SortDir != "desc" {
		return params, response.ValidationError("Parameter sort_dir harus 'asc' atau 'desc'", map[string]string{
			"sort_dir": "Nilai harus 'asc' atau 'desc'",
		})
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return params, response.ValidationError("Parameter page harus berupa bilangan bulat positif", map[string]string{
				"page": "Nomor halaman minimal 1",
			})
		}
		params.Page = page
	}

	if perPageStr := c.Query("per_page"); perPageStr != "" {
		perPage, err := strconv.Atoi(perPageStr)
		if err != nil || perPage < 1 {
			return params, response.ValidationError("Parameter per_page harus berupa bilangan bulat positif", map[string]string{
				"per_page": "Jumlah data per halaman minimal 1",
			})
		}
		if perPage > 100 {
			return params, response.ValidationError("Parameter per_page melebihi batas maksimum 100", map[string]string{
				"per_page": "Maksimum 100 item per halaman",
			})
		}
		params.PerPage = perPage
	}

	return params, nil
}

// DateRangeParams parses optional 'from' and 'to' date parameters in YYYY-MM-DD format.
type DateRangeParams struct {
	From *time.Time
	To   *time.Time
}

// ParseDateRange extracts and validates date range parameters.
func ParseDateRange(c *gin.Context) (DateRangeParams, *response.AppError) {
	var res DateRangeParams
	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))

	if fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return res, response.ValidationError("Format parameter 'from' harus YYYY-MM-DD", map[string]string{
				"from": "Gunakan format tanggal YYYY-MM-DD",
			})
		}
		res.From = &t
	}

	if toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return res, response.ValidationError("Format parameter 'to' harus YYYY-MM-DD", map[string]string{
				"to": "Gunakan format tanggal YYYY-MM-DD",
			})
		}
		res.To = &t
	}

	if res.From != nil && res.To != nil && res.From.After(*res.To) {
		return res, response.ValidationError("Tanggal 'from' tidak boleh lebih baru dari 'to'", map[string]string{
			"from": "Tanggal awal harus sebelum atau sama dengan tanggal akhir",
		})
	}

	return res, nil
}
