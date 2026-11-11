package handler

import (
	"strconv"
	"strings"

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

// getScope resolves the tenant restriction of the authenticated actor
// (ARY-001/ARY-005). It returns the partner id the actor is locked to, or nil
// for platform-wide actors. A partner-role user whose tenant linkage cannot be
// resolved is locked out entirely (ok == false).
func getScope(c *gin.Context) (*int, bool) {
	u := middleware.GetAuthUser(c)
	if u == nil {
		return nil, true // public flow: no internal scope
	}
	if u.PartnerID != nil {
		return u.PartnerID, true
	}
	if !strings.EqualFold(u.Role, "admin") {
		return nil, false
	}
	return nil, true
}

// mustScope is a variant of getScope for protected endpoints: an unresolvable
// partner account is locked out with an impossible scope instead of receiving
// platform-wide access.
func mustScope(c *gin.Context) *int {
	scope, ok := getScope(c)
	if !ok {
		denied := -1
		return &denied
	}
	return scope
}

func (h *UnitHandler) CreateUnit(c *gin.Context) {
	var req dto.CreateUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Nama unit usaha wajib diisi", nil))
		return
	}
	unit, err := h.svc.CreateUnit(c.Request.Context(), req, getActor(c), mustScope(c))
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
	unit, err := h.svc.GetUnit(c.Request.Context(), id, mustScope(c))
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
	units, total, err := h.svc.ListUnits(c.Request.Context(), p.Search, cat, activeOnly, mustScope(c), p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	// JSON list responses must always contain an array. PostgreSQL returns a
	// nil Go slice for zero rows, which otherwise serializes to data: null and
	// can break frontend consumers that iterate over the list.
	if units == nil {
		units = []domain.BusinessUnit{}
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
	unit, err := h.svc.UpdateUnit(c.Request.Context(), id, req, getActor(c), mustScope(c))
	if err != nil {
		if err == domain.ErrUnitNotFound {
			response.HandleAppError(c, response.NotFound("Unit usaha tidak ditemukan"))
			return
		}
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
	if err := h.svc.DeactivateUnit(c.Request.Context(), id, getActor(c), mustScope(c)); err != nil {
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
	if err := h.svc.ActivateUnit(c.Request.Context(), id, getActor(c), mustScope(c)); err != nil {
		response.HandleAppError(c, response.NotFound(err.Error()))
		return
	}
	response.OK(c, gin.H{"message": "Unit usaha berhasil diaktifkan kembali"})
}
