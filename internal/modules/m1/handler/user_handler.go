package handler

import (
	"errors"
	"strconv"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/dto"
	m1Repo "araya-business-hub/be/internal/modules/m1/repository"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSvc *service.UserService
	policy  *rbac.PolicyService
}

func NewUserHandler(svc *service.UserService, policy *rbac.PolicyService) *UserHandler {
	return &UserHandler{userSvc: svc, policy: policy}
}

// userError maps user-management errors to proper HTTP status codes.
func userError(err error) *response.AppError {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return response.NotFound("Pengguna tidak ditemukan")
	case errors.Is(err, service.ErrRoleChangeForbidden), errors.Is(err, service.ErrSelfAccessChange):
		return response.Forbidden(err.Error())
	case errors.Is(err, domain.ErrDuplicateEmail):
		return response.Conflict(err.Error())
	case errors.Is(err, service.ErrRoleUnavailable), errors.Is(err, m1Repo.ErrRoleNotAssignable),
		errors.Is(err, rbac.ErrRoleNotFound), errors.Is(err, rbac.ErrRoleInactive):
		return response.ValidationError("Role tidak ditemukan atau tidak aktif", nil)
	}
	return response.ValidationError(err.Error(), nil)
}

func getUserIDParam(c *gin.Context) (int, *response.AppError) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return 0, response.BadRequest("ID pengguna tidak valid")
	}
	return id, nil
}

func getActorName(c *gin.Context) string {
	if u := middleware.GetAuthUser(c); u != nil {
		return u.Name
	}
	return "system"
}

func (h *UserHandler) getActorContext(c *gin.Context) service.Actor {
	u := middleware.GetAuthUser(c)
	if u == nil {
		return service.Actor{Name: "system"}
	}
	actor := service.Actor{UserID: u.UserID, Name: u.Name, Role: u.Role, PartnerID: u.PartnerID}
	if h.policy != nil {
		actor.CanManageRoles = h.policy.CanRolePerform(c.Request.Context(), u.Role, service.PermRolesManage)
	}
	// Older tokens may not carry partner_id yet; resolve it once from the
	// database so partner-scoped users never fall back to global visibility.
	if actor.PartnerScoped() && actor.PartnerID == nil {
		actor.PartnerID = h.userSvc.ResolvePartnerID(c.Request.Context(), u.UserID)
	}
	return actor
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Data pengguna tidak lengkap", nil))
		return
	}
	user, err := h.userSvc.CreateUser(c.Request.Context(), req, h.getActorContext(c))
	if err != nil {
		response.HandleAppError(c, userError(err))
		return
	}
	response.Created(c, user)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, appErr := getUserIDParam(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	user, err := h.userSvc.GetUser(c.Request.Context(), id, h.getActorContext(c))
	if err != nil {
		response.HandleAppError(c, response.NotFound("Pengguna tidak ditemukan"))
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	p, appErr := query.ParsePagination(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	div := c.Query("division")
	users, total, err := h.userSvc.ListUsers(c.Request.Context(), p.Search, div, h.getActorContext(c), p.PerPage, p.Offset())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.List(c, users, p.Page, p.PerPage, total)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, appErr := getUserIDParam(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data tidak valid", nil))
		return
	}
	user, err := h.userSvc.UpdateUser(c.Request.Context(), id, req, h.getActorContext(c))
	if err != nil {
		response.HandleAppError(c, userError(err))
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) DeactivateUser(c *gin.Context) {
	id, appErr := getUserIDParam(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	if err := h.userSvc.DeactivateUser(c.Request.Context(), id, h.getActorContext(c)); err != nil {
		response.HandleAppError(c, userError(err))
		return
	}
	response.OK(c, gin.H{"message": "Pengguna berhasil dinonaktifkan"})
}

func (h *UserHandler) AssignRole(c *gin.Context) {
	id, appErr := getUserIDParam(c)
	if appErr != nil {
		response.HandleAppError(c, appErr)
		return
	}
	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Nama role wajib diisi", nil))
		return
	}
	if err := h.userSvc.AssignRole(c.Request.Context(), id, req, h.getActorContext(c)); err != nil {
		response.HandleAppError(c, userError(err))
		return
	}
	response.OK(c, gin.H{"message": "Role berhasil ditetapkan kepada pengguna"})
}
