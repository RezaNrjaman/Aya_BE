package handler

import (
	"strings"

	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	auditSvc *audit.Service
}

func NewAuditHandler(svc *audit.Service) *AuditHandler {
	return &AuditHandler{auditSvc: svc}
}

func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	user := middleware.GetAuthUser(c)
	var partnerID *int
	restricted := false
	if user != nil && !strings.EqualFold(user.Role, "admin") {
		partnerID = user.PartnerID
		restricted = true
	}
	records, total, err := h.auditSvc.ListScoped(c.Request.Context(), partnerID, restricted, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, records, p.Page, p.PerPage, total)
}
