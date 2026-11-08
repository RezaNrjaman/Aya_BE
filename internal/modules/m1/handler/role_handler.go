package handler

import (
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleSvc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{roleSvc: svc}
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roleSvc.ListRoles(c.Request.Context())
	if err != nil {
		response.HandleAppError(c, response.InternalError(err))
		return
	}
	response.OK(c, roles)
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Nama role dan display_name wajib diisi", nil))
		return
	}
	if err := h.roleSvc.CreateRole(c.Request.Context(), req, getActorName(c)); err != nil {
		response.HandleAppError(c, response.Conflict(err.Error()))
		return
	}
	response.Created(c, gin.H{"message": "Role berhasil dibuat"})
}

func (h *RoleHandler) UpdateRole(c *gin.Context) {
	roleName := c.Param("name")
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Format data tidak valid", nil))
		return
	}
	if err := h.roleSvc.UpdateRole(c.Request.Context(), roleName, req, getActorName(c)); err != nil {
		response.HandleAppError(c, response.NotFound(err.Error()))
		return
	}
	response.OK(c, gin.H{"message": "Role berhasil diperbarui"})
}

func (h *RoleHandler) AssignPermission(c *gin.Context) {
	roleName := c.Param("name")
	var req dto.AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("permission_key wajib diisi", nil))
		return
	}
	if err := h.roleSvc.AssignPermission(c.Request.Context(), roleName, req.PermissionKey, getActorName(c)); err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, gin.H{"message": "Permission berhasil dipetakan ke role"})
}

func (h *RoleHandler) RevokePermission(c *gin.Context) {
	roleName := c.Param("name")
	permKey := c.Param("key")
	if err := h.roleSvc.RevokePermission(c.Request.Context(), roleName, permKey, getActorName(c)); err != nil {
		response.HandleAppError(c, response.NotFound(err.Error()))
		return
	}
	response.OK(c, gin.H{"message": "Permission berhasil dicabut dari role"})
}
