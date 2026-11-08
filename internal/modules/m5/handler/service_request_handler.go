package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/service"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type ServiceRequestHandler struct {
	svc *service.ServiceRequestService
}

func NewServiceRequestHandler(svc *service.ServiceRequestService) *ServiceRequestHandler {
	return &ServiceRequestHandler{svc: svc}
}

func (h *ServiceRequestHandler) CreateRequest(c *gin.Context) {
	var req dto.CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data permintaan tidak valid", nil))
		return
	}
	sr, err := h.svc.CreateRequest(c.Request.Context(), req)
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, sr)
}

func (h *ServiceRequestHandler) GetRequest(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID permintaan tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	sr, err := h.svc.GetRequest(c.Request.Context(), id)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Permintaan layanan tidak ditemukan"))
		return
	}
	response.OK(c, sr)
}

func parseRequestFilter(c *gin.Context) (*int, string) {
	var uidPtr *int
	if uidStr := c.Query("unit_usaha_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			uidPtr = &uid
		}
	}
	return uidPtr, c.Query("status")
}

func (h *ServiceRequestHandler) ListRequests(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	uid, status := parseRequestFilter(c)
	list, total, err := h.svc.ListRequests(c.Request.Context(), uid, status, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, list, p.Page, p.PerPage, total)
}

func (h *ServiceRequestHandler) UpdateRequestStatus(c *gin.Context) {
	id, appErr := parseID(c, "id", "ID permintaan tidak valid")
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.UpdateServiceRequestStatus
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data status tidak valid", nil))
		return
	}
	sr, err := h.svc.UpdateRequestStatus(c.Request.Context(), id, req.Status, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, sr)
}
