package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m4/domain"
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func parseID(c *gin.Context, paramName, errMsg string) (int, *response.AppError) {
	id, err := strconv.Atoi(c.Param(paramName))
	if err != nil || id <= 0 {
		return 0, response.BadRequest(errMsg)
	}
	return id, nil
}

func getActor(c *gin.Context) string {
	if u := middleware.GetAuthUser(c); u != nil {
		return u.Name
	}
	return "system"
}

func parseUnitQuery(c *gin.Context) *int {
	uStr := c.Query("unit_usaha_id")
	if uStr == "" {
		return nil
	}
	id, err := strconv.Atoi(uStr)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data produk tidak valid", nil))
		return
	}
	product, err := h.svc.CreateProduct(c.Request.Context(), req, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, product)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID produk tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	p, err := h.svc.GetProduct(c.Request.Context(), id)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Produk tidak ditemukan"))
		return
	}
	response.OK(c, p)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID produk tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data produk tidak valid", nil))
		return
	}
	p, err := h.svc.UpdateProduct(c.Request.Context(), id, req, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, p)
}

func (h *ProductHandler) ListAdminProducts(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	unitID := parseUnitQuery(c)
	cat := c.Query("category")
	activeOnly := c.Query("active_only") == "true"
	products, total, err := h.svc.ListProducts(c.Request.Context(), unitID, cat, p.Search, activeOnly, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, products, p.Page, p.PerPage, total)
}

// ListActiveProducts serves Kontrak K-05: GET /api/m4/v1/products
func (h *ProductHandler) ListActiveProducts(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	unitID := parseUnitQuery(c)
	cat := c.Query("category")
	products, total, err := h.svc.ListActiveProducts(c.Request.Context(), unitID, cat, p.Search, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	if products == nil {
		products = []domain.Product{}
	}
	response.List(c, products, p.Page, p.PerPage, total)
}
