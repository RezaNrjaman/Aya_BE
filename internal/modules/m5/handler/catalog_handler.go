package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/service"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type CatalogHandler struct {
	svc *service.CatalogService
}

func NewCatalogHandler(svc *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) GetBusinessUnits(c *gin.Context) {
	units, err := h.svc.GetBusinessUnits(c.Request.Context())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, units)
}

func parseCatalogFilter(c *gin.Context) dto.CatalogFilter {
	var f dto.CatalogFilter
	if uidStr := c.Query("unit_usaha_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			f.UnitUsahaID = &uid
		}
	}
	f.Category = c.Query("category")
	f.Search = c.Query("search")
	return f
}

func (h *CatalogHandler) GetProducts(c *gin.Context) {
	f := parseCatalogFilter(c)
	prods, err := h.svc.GetProducts(c.Request.Context(), f)
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, prods)
}

func (h *CatalogHandler) GetCatalog(c *gin.Context) {
	f := parseCatalogFilter(c)
	cat, err := h.svc.GetCatalog(c.Request.Context(), f)
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, cat)
}

