package handler

import (
	"araya-business-hub/be/internal/modules/m4/dto"
	"araya-business-hub/be/internal/modules/m4/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	svc *service.CustomerService
}

func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req dto.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data customer tidak valid", nil))
		return
	}
	customer, err := h.svc.CreateCustomer(c.Request.Context(), req)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, customer)
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID customer tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	cust, err := h.svc.GetCustomer(c.Request.Context(), id)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Customer tidak ditemukan"))
		return
	}
	response.OK(c, cust)
}

func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	customers, total, err := h.svc.ListCustomers(c.Request.Context(), p.Search, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, customers, p.Page, p.PerPage, total)
}

