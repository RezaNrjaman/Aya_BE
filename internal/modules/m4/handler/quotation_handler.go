package handler

import (
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type QuotationHandler struct {
	svc *service.QuotationService
}

func NewQuotationHandler(svc *service.QuotationService) *QuotationHandler {
	return &QuotationHandler{svc: svc}
}

func (h *QuotationHandler) CreateQuotation(c *gin.Context) {
	var req dto.CreateQuotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data penawaran tidak valid", nil))
		return
	}
	quotation, err := h.svc.CreateQuotation(c.Request.Context(), req, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, quotation)
}

func (h *QuotationHandler) GetQuotation(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID quotation tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	q, err := h.svc.GetQuotation(c.Request.Context(), id)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Quotation tidak ditemukan"))
		return
	}
	response.OK(c, q)
}

func (h *QuotationHandler) ApproveQuotation(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID quotation tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	q, err := h.svc.ApproveQuotation(c.Request.Context(), id, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, q)
}

func (h *QuotationHandler) RejectQuotation(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID quotation tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	q, err := h.svc.RejectQuotation(c.Request.Context(), id, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, q)
}

func (h *QuotationHandler) ListQuotations(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	status := c.Query("status")
	list, total, err := h.svc.ListQuotations(c.Request.Context(), status, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, list, p.Page, p.PerPage, total)
}

