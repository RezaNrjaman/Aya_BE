package handler

import (
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/service"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type CartHandler struct {
	svc *service.CartService
}

func NewCartHandler(svc *service.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

func (h *CartHandler) GetCart(c *gin.Context) {
	sessionID := getSessionID(c)
	cart, err := h.svc.GetCart(c.Request.Context(), sessionID)
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, cart)
}

func (h *CartHandler) AddItem(c *gin.Context) {
	sessionID := getSessionID(c)
	var req dto.AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data item keranjang tidak valid", nil))
		return
	}
	cart, err := h.svc.AddItem(c.Request.Context(), sessionID, req)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, cart)
}

func (h *CartHandler) UpdateItem(c *gin.Context) {
	sessionID := getSessionID(c)
	productID, appErr := parseID(c, "product_id", "ID produk tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format jumlah item tidak valid", nil))
		return
	}
	cart, err := h.svc.UpdateItem(c.Request.Context(), sessionID, productID, req.Quantity)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, cart)
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	sessionID := getSessionID(c)
	productID, appErr := parseID(c, "product_id", "ID produk tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	cart, err := h.svc.RemoveItem(c.Request.Context(), sessionID, productID)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Item tidak ditemukan di keranjang"))
		return
	}
	response.OK(c, cart)
}

func (h *CartHandler) ClearCart(c *gin.Context) {
	sessionID := getSessionID(c)
	if err := h.svc.ClearCart(c.Request.Context(), sessionID); err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, map[string]string{"message": "Keranjang belanja berhasil dikosongkan"})
}

