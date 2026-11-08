package handler

import (
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/service"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type CheckoutHandler struct {
	svc *service.CheckoutService
}

func NewCheckoutHandler(svc *service.CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{svc: svc}
}

func (h *CheckoutHandler) Checkout(c *gin.Context) {
	sessionID := getSessionID(c)
	var req dto.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data checkout tidak valid", nil))
		return
	}
	resp, err := h.svc.Checkout(c.Request.Context(), sessionID, req)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, resp)
}
