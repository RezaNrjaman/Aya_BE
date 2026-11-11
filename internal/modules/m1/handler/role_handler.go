package handler

import (
	"context"
	"errors"
	"strings"

	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleSvc         *service.RoleService
	partnerResolver func(context.Context, int) *int
}

func NewRoleHandler(svc *service.RoleService, resolver ...func(context.Context, int) *int) *RoleHandler {
	var resolve func(context.Context, int) *int
	if len(resolver) > 0 {
		resolve = resolver[0]
	}
	return &RoleHandler{roleSvc: svc, partnerResolver: resolve}
}

func (h *RoleHandler) getRoleActor(c *gin.Context) service.RoleActor {
	if u := middleware.GetAuthUser(c); u != nil {
		actor := service.RoleActor{Name: u.Name, Role: u.Role, PartnerID: u.PartnerID}
		isPartnerActor := strings.EqualFold(u.Role, "partner") || strings.HasPrefix(strings.ToLower(u.Role), "p")
		if actor.PartnerID == nil && isPartnerActor && h.partnerResolver != nil {
			actor.PartnerID = h.partnerResolver(c.Request.Context(), u.UserID)
		}
		return actor
	}
	return service.RoleActor{Name: "system"}
}

// roleError maps RBAC domain errors to the standard API error envelope.
func roleError(err error) *response.AppError {
	switch {
	case errors.Is(err, rbac.ErrRoleNotFound):
		return response.NotFound("Role tidak ditemukan")
	case errors.Is(err, rbac.ErrDuplicateRole):
		return response.Conflict("Kode role sudah digunakan")
	case errors.Is(err, service.ErrAccessLockout), errors.Is(err, service.ErrPlatformRolesOnly),
		errors.Is(err, service.ErrDelegationForbidden), errors.Is(err, rbac.ErrSystemRole):
		return response.Forbidden(err.Error())
	case errors.Is(err, rbac.ErrRoleInactive):
		return response.ValidationError("Role tidak aktif; aktifkan role terlebih dahulu", nil)
	case errors.Is(err, service.ErrUnknownPermission), errors.Is(err, service.ErrRoleRequiresPermission), errors.Is(err, rbac.ErrPermissionNotFound),
		errors.Is(err, rbac.ErrInvalidRoleName):
		return response.ValidationError(err.Error(), nil)
	}
	var appErr *response.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return response.ValidationError(err.Error(), nil)
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roleSvc.ListRoles(c.Request.Context(), h.getRoleActor(c))
	if err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.OK(c, roles)
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	permissions, err := h.roleSvc.ListPermissionsForActor(c.Request.Context(), h.getRoleActor(c))
	if err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.OK(c, permissions)
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Kode role dan nama tampilan wajib diisi", nil))
		return
	}
	if err := h.roleSvc.CreateRole(c.Request.Context(), req, h.getRoleActor(c)); err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.Created(c, gin.H{"message": "Role berhasil dibuat"})
}

func (h *RoleHandler) UpdateRole(c *gin.Context) {
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data tidak valid", nil))
		return
	}
	if err := h.roleSvc.UpdateRole(c.Request.Context(), c.Param("name"), req, h.getRoleActor(c)); err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.OK(c, gin.H{"message": "Role berhasil diperbarui"})
}

func (h *RoleHandler) AssignPermission(c *gin.Context) {
	var req dto.AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("permission_key wajib diisi", nil))
		return
	}
	if err := h.roleSvc.AssignPermission(c.Request.Context(), c.Param("name"), req.PermissionKey, h.getRoleActor(c)); err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.OK(c, gin.H{"message": "Permission berhasil dipetakan ke role"})
}

func (h *RoleHandler) RevokePermission(c *gin.Context) {
	if err := h.roleSvc.RevokePermission(c.Request.Context(), c.Param("name"), c.Param("key"), h.getRoleActor(c)); err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.OK(c, gin.H{"message": "Permission berhasil dicabut dari role"})
}

// SetPermissions replaces the full permission set of a role (permission matrix).
func (h *RoleHandler) SetPermissions(c *gin.Context) {
	var req dto.SetRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("permission_keys harus berupa daftar", nil))
		return
	}
	role, err := h.roleSvc.SetPermissions(c.Request.Context(), c.Param("name"), req.PermissionKeys, h.getRoleActor(c))
	if err != nil {
		response.HandleAppError(c, roleError(err))
		return
	}
	response.OK(c, role)
}
