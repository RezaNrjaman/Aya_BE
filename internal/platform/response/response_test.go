package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := map[string]any{"id": 125, "name": "Unit Usaha Araya"}
	response.OK(c, payload)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res response.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected success true, got false")
	}

	dataMap, ok := res.Data.(map[string]any)
	if !ok || dataMap["id"] != float64(125) {
		t.Fatalf("expected data.id = 125, got %v", res.Data)
	}
}

func TestListResponseWithPagination(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []map[string]any{
		{"id": 1, "title": "Produk 1"},
		{"id": 2, "title": "Produk 2"},
	}
	response.List(c, items, 1, 20, 45)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res response.ListSuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected success true")
	}

	if res.Meta.Total != 45 || res.Meta.TotalPages != 3 || res.Meta.Page != 1 || res.Meta.PerPage != 20 {
		t.Fatalf("unexpected meta values: %+v", res.Meta)
	}
}

func TestListResponseEmptyNilGuaranteedSlice(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// nil items must render as [] in json, not null
	response.List(c, nil, 1, 20, 0)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res response.ListSuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	slice, ok := res.Data.([]any)
	if !ok {
		t.Fatalf("expected data to be slice, got %T", res.Data)
	}
	if len(slice) != 0 {
		t.Fatalf("expected empty slice, got len %d", len(slice))
	}
}

func TestStandardErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		errFunc    func(c *gin.Context)
		wantStatus int
		wantCode   string
	}{
		{
			name: "Validation Error",
			errFunc: func(c *gin.Context) {
				fields := map[string]string{"name": "Nama wajib diisi"}
				response.HandleAppError(c, response.ValidationError("Validasi gagal", fields))
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   response.CodeValidationError,
		},
		{
			name: "Unauthorized Error",
			errFunc: func(c *gin.Context) {
				response.HandleAppError(c, response.Unauthorized(""))
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   response.CodeUnauthorized,
		},
		{
			name: "Forbidden Error",
			errFunc: func(c *gin.Context) {
				response.HandleAppError(c, response.Forbidden(""))
			},
			wantStatus: http.StatusForbidden,
			wantCode:   response.CodeForbidden,
		},
		{
			name: "Not Found Error",
			errFunc: func(c *gin.Context) {
				response.HandleAppError(c, response.NotFound(""))
			},
			wantStatus: http.StatusNotFound,
			wantCode:   response.CodeNotFound,
		},
		{
			name: "Conflict Error",
			errFunc: func(c *gin.Context) {
				response.HandleAppError(c, response.Conflict(""))
			},
			wantStatus: http.StatusConflict,
			wantCode:   response.CodeConflict,
		},
		{
			name: "Internal Server Error",
			errFunc: func(c *gin.Context) {
				response.HandleAppError(c, response.InternalError(errors.New("db connection failure")))
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   response.CodeInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			tt.errFunc(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var res response.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
				t.Fatalf("failed to parse json: %v", err)
			}

			if res.Success {
				t.Fatalf("expected success false, got true")
			}
			if res.Error.Code != tt.wantCode {
				t.Fatalf("expected code %s, got %s", tt.wantCode, res.Error.Code)
			}
			if res.Error.Message == "" {
				t.Fatalf("expected non-empty message")
			}
		})
	}
}
