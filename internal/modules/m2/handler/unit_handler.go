package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type UnitHandler struct {
	svc *service.UnitService
}

func NewUnitHandler(svc *service.UnitService) *UnitHandler {
	return &UnitHandler{svc: svc}
}

func parseUnitID(c *gin.Context) (int, *response.AppError) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, response.BadRequest("ID unit usaha tidak valid")
	}
	return id, nil
}

func getActor(c *gin.Context) string {
	if u := middleware.GetAuthUser(c); u != nil {
		return u.Name
	}
	return "system"
}

func (h *UnitHandler) CreateUnit(c *gin.Context) {
	var req dto.CreateUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Nama unit usaha wajib diisi", nil))
		return
	}
	unit, err := h.svc.CreateUnit(c.Request.Context(), req, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, unit)
}

func (h *UnitHandler) GetUnit(c *gin.Context) {
	id, appErr := parseUnitID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	unit, err := h.svc.GetUnit(c.Request.Context(), id)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Unit usaha tidak ditemukan"))
		return
	}
	response.OK(c, unit)
}

func (h *UnitHandler) ListAdminUnits(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	cat := c.Query("category")
	activeOnly := c.Query("active_only") == "true"
	units, total, err := h.svc.ListUnits(c.Request.Context(), p.Search, cat, activeOnly, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, units, p.Page, p.PerPage, total)
}

// ListActiveUnits serves Kontrak K-02: GET /api/m2/v1/business-units
func (h *UnitHandler) ListActiveUnits(c *gin.Context) {
	units, err := h.svc.ListActiveUnits(c.Request.Context())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	if units == nil {
		units = []domain.BusinessUnit{}
	}
	response.OK(c, units)
}

func (h *UnitHandler) UpdateUnit(c *gin.Context) {
	id, appErr := parseUnitID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.UpdateUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data tidak valid", nil))
		return
	}
	unit, err := h.svc.UpdateUnit(c.Request.Context(), id, req, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, unit)
}

func (h *UnitHandler) DeactivateUnit(c *gin.Context) {
	id, appErr := parseUnitID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	if err := h.svc.DeactivateUnit(c.Request.Context(), id, getActor(c)); err != nil {
		response.HandleAppError(c, response.NotFound(err.Error()))
		return
	}
	response.OK(c, gin.H{"message": "Unit usaha berhasil dinonaktifkan"})
}

func (h *UnitHandler) ActivateUnit(c *gin.Context) {
	id, appErr := parseUnitID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	if err := h.svc.ActivateUnit(c.Request.Context(), id, getActor(c)); err != nil {
		response.HandleAppError(c, response.NotFound(err.Error()))
		return
	}
	response.OK(c, gin.H{"message": "Unit usaha berhasil diaktifkan kembali"})
}
