package handler

import (
	"errors"
	"strconv"
	"strings"

	"araya-business-hub/be/internal/modules/m5/domain"
	"araya-business-hub/be/internal/modules/m5/dto"
	"araya-business-hub/be/internal/modules/m5/service"
	"araya-business-hub/be/internal/platform/middleware"
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

// partnerScope locks every authenticated non-platform account to its own tenant.
// A missing tenant linkage is fail-closed, never interpreted as global access.
func partnerScope(c *gin.Context) (*int, bool) {
	u := middleware.GetAuthUser(c)
	if u == nil {
		return nil, true
	}
	if strings.EqualFold(u.Role, "admin") {
		return nil, false
	}
	return u.PartnerID, true
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
	partnerID, restricted := partnerScope(c)
	sr, err := h.svc.GetRequest(c.Request.Context(), id, partnerID, restricted)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Permintaan layanan tidak ditemukan"))
		return
	}
	response.OK(c, sr)
}

func parseRequestFilter(c *gin.Context) (*int, string) {
	var uidPtr *int
	if uidStr := c.Query("unit_usaha_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil && uid > 0 {
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
	partnerID, restricted := partnerScope(c)
	list, total, err := h.svc.ListRequests(c.Request.Context(), uid, status, partnerID, restricted, p.PerPage, p.Offset())
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
	u := middleware.GetAuthUser(c)
	actor := "partner"
	if u != nil {
		actor = u.Name
	}
	partnerID, restricted := partnerScope(c)
	sr, err := h.svc.UpdateRequestStatus(c.Request.Context(), id, req.Status, actor, partnerID, restricted)
	if err != nil {
		if errors.Is(err, domain.ErrRequestNotFound) {
			response.HandleAppError(c, response.NotFound("Permintaan layanan tidak ditemukan"))
		} else {
			response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		}
		return
	}
	response.OK(c, sr)
}
