package handler

import (
	"fmt"

	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc        *service.OrderService
	invoiceSvc *service.InvoiceService
}

func NewOrderHandler(svc *service.OrderService, invoiceSvc ...*service.InvoiceService) *OrderHandler {
	h := &OrderHandler{svc: svc}
	if len(invoiceSvc) > 0 {
		h.invoiceSvc = invoiceSvc[0]
	}
	return h
}

type publicOrderTrackRequest struct {
	OrderNo       string `json:"order_no" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
}

type publicOrderPaymentRequest struct {
	CustomerEmail   string `json:"customer_email" binding:"required,email"`
	Method          string `json:"method" binding:"required"`
	SimulateFailure bool   `json:"simulate_failure,omitempty"`
}

func (h *OrderHandler) TrackPublicOrder(c *gin.Context) {
	var req publicOrderTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Nomor pesanan dan email wajib diisi", nil))
		return
	}
	order, err := h.svc.GetPublicOrder(c.Request.Context(), req.OrderNo, req.CustomerEmail)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Pesanan tidak ditemukan. Periksa nomor pesanan dan email Anda."))
		return
	}
	var invoice any
	if order.InvoiceID != nil && h.invoiceSvc != nil {
		invoice, _ = h.invoiceSvc.GetInvoice(c.Request.Context(), *order.InvoiceID, nil)
	}
	response.OK(c, gin.H{"order": order, "invoice": invoice})
}

func (h *OrderHandler) PayPublicOrder(c *gin.Context) {
	if h.invoiceSvc == nil {
		response.HandleAppError(c, response.InternalError(fmt.Errorf("layanan pembayaran tidak tersedia")))
		return
	}
	var req publicOrderPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Email pelanggan dan metode pembayaran wajib diisi", nil))
		return
	}
	order, err := h.svc.GetPublicOrder(c.Request.Context(), c.Param("order_no"), req.CustomerEmail)
	if err != nil || order.InvoiceID == nil || order.Status == "dibatalkan" {
		response.HandleAppError(c, response.NotFound("Pesanan atau tagihan tidak ditemukan"))
		return
	}
	inv, err := h.invoiceSvc.GetInvoice(c.Request.Context(), *order.InvoiceID, nil)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Tagihan tidak ditemukan"))
		return
	}
	payment, err := h.invoiceSvc.PayInvoice(c.Request.Context(), inv.ID, dto.PayInvoiceRequest{
		Method: req.Method, Amount: inv.TotalAmount, SimulateFailure: req.SimulateFailure,
	}, "customer.portal", nil)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, gin.H{"payment": payment, "invoice_status": func() string {
		updated, e := h.invoiceSvc.GetInvoice(c.Request.Context(), inv.ID, nil)
		if e != nil {
			return inv.Status
		}
		return updated.Status
	}()})
}

// CreateOrder serves Kontrak K-04: POST /api/m4/v1/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data order tidak valid", nil))
		return
	}
	order, err := h.svc.CreateOrder(c.Request.Context(), req, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID order tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	order, err := h.svc.GetOrder(c.Request.Context(), id, mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.NotFound("Order tidak ditemukan"))
		return
	}
	response.OK(c, order)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	status := c.Query("status")
	list, total, err := h.svc.ListOrders(c.Request.Context(), status, mustScope(c), p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, list, p.Page, p.PerPage, total)
}

func (h *OrderHandler) ProcessOrder(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID order tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	order, err := h.svc.ProcessOrder(c.Request.Context(), id, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, order)
}

func (h *OrderHandler) CompleteOrder(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID order tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	order, err := h.svc.CompleteOrder(c.Request.Context(), id, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, order)
}

type cancelOrderReq struct {
	Reason string `json:"reason"`
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID order tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req cancelOrderReq
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "Dibatalkan oleh operasional/partner"
	}
	order, err := h.svc.CancelOrder(c.Request.Context(), id, req.Reason, getActor(c), mustScope(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, order)
}
