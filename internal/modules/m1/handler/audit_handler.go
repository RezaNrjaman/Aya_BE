package handler

import (
	"araya-business-hub/be/internal/platform/audit"
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
	records, total, err := h.auditSvc.List(c.Request.Context(), p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, records, p.Page, p.PerPage, total)
}
