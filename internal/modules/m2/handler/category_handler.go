package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m2/dto"
	"araya-business-hub/be/internal/modules/m2/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct{ svc *service.CategoryService }

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) List(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var unitID *int
	if raw := c.Query("unit_usaha_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			response.HandleAppError(c, response.BadRequest("ID unit usaha tidak valid"))
			return
		}
		unitID = &id
	}
	partnerID := mustScope(c)
	items, total, err := h.svc.List(c.Request.Context(), unitID, partnerID, p.Search, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, items, p.Page, p.PerPage, total)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Unit usaha dan nama kategori wajib diisi", nil))
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, item)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.HandleAppError(c, response.BadRequest("ID kategori tidak valid"))
		return
	}
	var req dto.UpdateCategoryRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		response.HandleAppError(c, response.ValidationError("Format data tidak valid", nil))
		return
	}
	item, e := h.svc.Update(c.Request.Context(), id, req, getActor(c), mustScope(c))
	if e != nil {
		response.HandleAppError(c, response.ValidationError(e.Error(), nil))
		return
	}
	response.OK(c, item)
}

func (h *CategoryHandler) Deactivate(c *gin.Context) { h.setStatus(c, false) }
func (h *CategoryHandler) Activate(c *gin.Context)   { h.setStatus(c, true) }
func (h *CategoryHandler) setStatus(c *gin.Context, active bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.HandleAppError(c, response.BadRequest("ID kategori tidak valid"))
		return
	}
	if e := h.svc.SetStatus(c.Request.Context(), id, active, getActor(c), mustScope(c)); e != nil {
		response.HandleAppError(c, response.NotFound(e.Error()))
		return
	}
	response.OK(c, gin.H{"message": "Status kategori berhasil diperbarui"})
}
