package handler

import (
	"errors"
	"strconv"
	"strings"

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

// getScope resolves the tenant restriction of the authenticated actor (ARY-001).
// It returns the partner id the actor is locked to, or nil for platform-wide
// actors (super admin and internal staff). A partner-role user whose tenant
// linkage cannot be resolved is locked out entirely (ok == false).
func getScope(c *gin.Context) (*int, bool) {
	u := middleware.GetAuthUser(c)
	if u == nil {
		return nil, true // public flow: no internal scope
	}
	if u.PartnerID != nil {
		return u.PartnerID, true
	}
	if strings.EqualFold(u.Role, "partner") {
		return nil, false
	}
	return nil, true
}

// mustScope is a variant of getScope for protected endpoints: an unresolvable
// partner account is locked out with an impossible scope instead of receiving
// platform-wide access (ARY-001).
func mustScope(c *gin.Context) *int {
	scope, ok := getScope(c)
	if !ok {
		denied := -1
		return &denied
	}
	return scope
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
	scope, ok := getScope(c)
	if !ok {
		response.HandleAppError(c, response.Forbidden("Akun mitra belum terhubung ke data partner"))
		return
	}
	product, err := h.svc.CreateProduct(c.Request.Context(), req, getActor(c), scope)
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
	scope, _ := getScope(c)
	p, err := h.svc.GetProduct(c.Request.Context(), id, scope)
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
	scope, ok := getScope(c)
	if !ok {
		response.HandleAppError(c, response.Forbidden("Akun mitra belum terhubung ke data partner"))
		return
	}
	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data produk tidak valid", nil))
		return
	}
	p, err := h.svc.UpdateProduct(c.Request.Context(), id, req, getActor(c), scope)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			response.HandleAppError(c, response.NotFound("Produk tidak ditemukan"))
			return
		}
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
	scope, ok := getScope(c)
	if !ok {
		response.HandleAppError(c, response.Forbidden("Akun mitra belum terhubung ke data partner"))
		return
	}
	unitID := parseUnitQuery(c)
	cat := c.Query("category")
	activeOnly := c.Query("active_only") == "true"
	products, total, err := h.svc.ListProducts(c.Request.Context(), unitID, cat, p.Search, activeOnly, scope, p.PerPage, p.Offset())
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
