package handler

import (
	"errors"
	"strconv"

	"araya-business-hub/be/internal/modules/m2/domain"
	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type DivisionHandler struct {
	svc *service.DivisionService
}

func NewDivisionHandler(svc *service.DivisionService) *DivisionHandler {
	return &DivisionHandler{svc: svc}
}

func parseDivisionID(c *gin.Context) (int, *response.AppError) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, response.BadRequest("ID divisi tidak valid")
	}
	return id, nil
}

// divisionError maps division domain errors to API responses.
func divisionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrDivisionNotFound):
		response.HandleAppError(c, response.NotFound("Divisi tidak ditemukan"))
	case errors.Is(err, domain.ErrDivisionInUse):
		response.HandleAppError(c, response.Conflict(err.Error()))
	case errors.Is(err, domain.ErrDuplicateDivision), errors.Is(err, domain.ErrNameRequired):
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
	default:
		response.HandleAppError(c, response.InternalError(err))
	}
}

func (h *DivisionHandler) CreateDivision(c *gin.Context) {
	var req dto.CreateDivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Nama divisi wajib diisi", nil))
		return
	}
	d, err := h.svc.CreateDivision(c.Request.Context(), req, getActor(c), mustScope(c))
	if err != nil {
		divisionError(c, err)
		return
	}
	response.Created(c, d)
}

func (h *DivisionHandler) GetDivision(c *gin.Context) {
	id, appErr := parseDivisionID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	d, err := h.svc.GetDivision(c.Request.Context(), id, mustScope(c))
	if err != nil {
		divisionError(c, err)
		return
	}
	response.OK(c, d)
}

func (h *DivisionHandler) ListDivisions(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	divisions, total, err := h.svc.ListDivisions(c.Request.Context(), p.Search, mustScope(c), p.PerPage, p.Offset())
	if err != nil {
		divisionError(c, err)
		return
	}
	response.List(c, divisions, p.Page, p.PerPage, total)
}

func (h *DivisionHandler) UpdateDivision(c *gin.Context) {
	id, appErr := parseDivisionID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.UpdateDivisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data tidak valid", nil))
		return
	}
	d, err := h.svc.UpdateDivision(c.Request.Context(), id, req, getActor(c), mustScope(c))
	if err != nil {
		divisionError(c, err)
		return
	}
	response.OK(c, d)
}

func (h *DivisionHandler) DeactivateDivision(c *gin.Context) {
	id, appErr := parseDivisionID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	if err := h.svc.DeactivateDivision(c.Request.Context(), id, getActor(c), mustScope(c)); err != nil {
		divisionError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Divisi berhasil dinonaktifkan"})
}

func (h *DivisionHandler) ActivateDivision(c *gin.Context) {
	id, appErr := parseDivisionID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	if err := h.svc.ActivateDivision(c.Request.Context(), id, getActor(c), mustScope(c)); err != nil {
		divisionError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Divisi berhasil diaktifkan kembali"})
}

func (h *DivisionHandler) DeleteDivision(c *gin.Context) {
	id, appErr := parseDivisionID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	if err := h.svc.DeleteDivision(c.Request.Context(), id, getActor(c), mustScope(c)); err != nil {
		divisionError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Divisi berhasil dihapus"})
}
