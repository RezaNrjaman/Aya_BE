package handler

import (
	"encoding/json"

	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	svc *service.InvoiceService
}

func NewInvoiceHandler(svc *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	var req dto.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data tagihan tidak valid", nil))
		return
	}
	invoice, err := h.svc.CreateInvoice(c.Request.Context(), req, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, invoice)
}

func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID invoice tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	inv, err := h.svc.GetInvoice(c.Request.Context(), id, mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.NotFound("Invoice tidak ditemukan"))
		return
	}
	response.OK(c, inv)
}

func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	status := c.Query("status")
	list, total, err := h.svc.ListInvoices(c.Request.Context(), status, mustScope(c), p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, list, p.Page, p.PerPage, total)
}

func (h *InvoiceHandler) PayInvoice(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID invoice tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.PayInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data pembayaran tidak valid", nil))
		return
	}
	payment, err := h.svc.PayInvoice(c.Request.Context(), id, req, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, payment)
}

func (h *InvoiceHandler) HandlePaymentCallback(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil {
		response.HandleAppError(c, response.BadRequest("Gagal membaca body callback"))
		return
	}
	sig := c.GetHeader("X-Callback-Signature")
	if !h.svc.VerifySignature(raw, sig) {
		response.HandleAppError(c, response.Unauthorized("Signature webhook pembayaran tidak valid"))
		return
	}
	var req dto.PaymentCallbackRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data callback tidak valid", nil))
		return
	}
	payment, err := h.svc.HandlePaymentCallback(c.Request.Context(), req)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, payment)
}
