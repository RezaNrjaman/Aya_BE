package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m2/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func parseUnitFilter(c *gin.Context) *int {
	unitStr := c.Query("unit_usaha_id")
	if unitStr == "" {
		return nil
	}
	id, err := strconv.Atoi(unitStr)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func resolveUnitFilter(c *gin.Context) (*int, *response.AppError) {
	reqID := parseUnitFilter(c)
	user := middleware.GetAuthUser(c)
	if user != nil && user.UnitUsahaID != nil {
		if reqID != nil && *reqID != *user.UnitUsahaID {
			return nil, response.Forbidden("Akses ditolak untuk unit usaha di luar kewenangan")
		}
		return user.UnitUsahaID, nil
	}
	return reqID, nil
}

// GetSummary serves Kontrak K-03: GET /api/m2/v1/dashboard-summary
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	dr, appErr := query.ParseDateRange(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	unitID, scopeErr := resolveUnitFilter(c)
	if scopeErr != nil {
		response.HandleAppError(c, scopeErr)
		return
	}
	summary, err := h.svc.GetSummary(c.Request.Context(), unitID, dr.From, dr.To)
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, summary)
}
