package handler

import (
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// CreateOrder serves Kontrak K-04: POST /api/m4/v1/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data order tidak valid", nil))
		return
	}
	order, err := h.svc.CreateOrder(c.Request.Context(), req, getActor(c))
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
	order, err := h.svc.GetOrder(c.Request.Context(), id)
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
	list, total, err := h.svc.ListOrders(c.Request.Context(), status, p.PerPage, p.Offset())
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
	order, err := h.svc.ProcessOrder(c.Request.Context(), id, getActor(c))
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
	order, err := h.svc.CompleteOrder(c.Request.Context(), id, getActor(c))
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
	order, err := h.svc.CancelOrder(c.Request.Context(), id, req.Reason, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, order)
}
