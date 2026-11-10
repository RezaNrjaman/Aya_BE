package handler

import (
	"errors"
	"strconv"
	"strings"

	"araya-business-hub/be/internal/modules/m3/domain"
	"araya-business-hub/be/internal/modules/m3/dto"
	"araya-business-hub/be/internal/modules/m3/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type PartnerHandler struct {
	svc *service.PartnerService
}

func NewPartnerHandler(svc *service.PartnerService) *PartnerHandler {
	return &PartnerHandler{svc: svc}
}

func parsePartnerID(c *gin.Context) (int, *response.AppError) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, response.BadRequest("ID partner tidak valid")
	}
	return id, nil
}

func getActor(c *gin.Context) string {
	if u := middleware.GetAuthUser(c); u != nil {
		return u.Name
	}
	return "system"
}

// resolveScope returns the partner that the authenticated user is allowed to
// see. restricted is true for partner accounts and accounts linked to a partner,
// even when the linkage could not be resolved.
func (h *PartnerHandler) resolveScope(c *gin.Context) (*int, bool) {
	u := middleware.GetAuthUser(c)
	if u == nil {
		return nil, false
	}
	if u.PartnerID != nil {
		return u.PartnerID, true
	}
	if !strings.EqualFold(u.Role, "partner") {
		return nil, false
	}
	// Fallback for tokens issued before partner_id was included in claims.
	if p, err := h.svc.GetPartnerByUserID(c.Request.Context(), u.UserID); err == nil {
		id := p.ID
		return &id, true
	}
	return nil, true
}

func (h *PartnerHandler) Register(c *gin.Context) {
	var req dto.RegisterPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Data pendaftaran tidak lengkap", nil))
		return
	}
	partner, err := h.svc.Register(c.Request.Context(), req, getActor(c))
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyRegistered) {
			response.HandleAppError(c, response.Conflict(err.Error()))
			return
		}
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.Created(c, partner)
}

func (h *PartnerHandler) GetPartner(c *gin.Context) {
	id, appErr := parsePartnerID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	scope, restricted := h.resolveScope(c)
	detail, err := h.svc.GetPartnerDetailScoped(c.Request.Context(), id, scope, restricted)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Partner tidak ditemukan"))
		return
	}
	response.OK(c, detail)
}

func (h *PartnerHandler) ListPartners(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	status := c.Query("status")
	scope, restricted := h.resolveScope(c)
	partners, total, err := h.svc.ListPartnersScoped(c.Request.Context(), scope, restricted, status, p.Search, p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, partners, p.Page, p.PerPage, total)
}

func (h *PartnerHandler) ListVerifiedPartners(c *gin.Context) {
	views, err := h.svc.PublicVerifiedPartners(c.Request.Context())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, views)
}

// PublicPartnerStatus answers the public registration status page. It is
// addressed by the unguessable registration code and returns a minimal
// payload (ARY-002).
func (h *PartnerHandler) PublicPartnerStatus(c *gin.Context) {
	code := c.Query("code")
	detail, err := h.svc.GetPublicStatusByCode(c.Request.Context(), code)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Kode pelacakan tidak dikenali"))
		return
	}
	response.OK(c, detail)
}

func (h *PartnerHandler) Verify(c *gin.Context) {
	id, appErr := parsePartnerID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	scope, restricted := h.resolveScope(c)
	if err := h.svc.EnsurePartnerAccess(c.Request.Context(), id, scope, restricted); err != nil {
		response.HandleAppError(c, response.NotFound("Partner tidak ditemukan"))
		return
	}
	var req dto.VerifyPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data verifikasi tidak valid", nil))
		return
	}
	partner, err := h.svc.VerifyPartner(c.Request.Context(), id, req, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, partner)
}

func (h *PartnerHandler) Suspend(c *gin.Context) {
	id, appErr := parsePartnerID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	scope, restricted := h.resolveScope(c)
	if err := h.svc.EnsurePartnerAccess(c.Request.Context(), id, scope, restricted); err != nil {
		response.HandleAppError(c, response.NotFound("Partner tidak ditemukan"))
		return
	}
	var req dto.SuspendPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		response.HandleAppError(c, response.ValidationError("Alasan penangguhan wajib diisi", nil))
		return
	}
	partner, err := h.svc.SuspendPartner(c.Request.Context(), id, req.Reason, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, partner)
}

func (h *PartnerHandler) Reactivate(c *gin.Context) {
	id, appErr := parsePartnerID(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	scope, restricted := h.resolveScope(c)
	if err := h.svc.EnsurePartnerAccess(c.Request.Context(), id, scope, restricted); err != nil {
		response.HandleAppError(c, response.NotFound("Partner tidak ditemukan"))
		return
	}
	partner, err := h.svc.ReactivatePartner(c.Request.Context(), id, getActor(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, partner)
}
