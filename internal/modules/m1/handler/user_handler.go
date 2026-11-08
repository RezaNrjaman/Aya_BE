package handler

import (
	"strconv"

	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/query"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userSvc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: svc}
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

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Data pengguna tidak lengkap", nil))
		return
	}
	user, err := h.userSvc.CreateUser(c.Request.Context(), req, getActorName(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
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
	user, err := h.userSvc.GetUser(c.Request.Context(), id)
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
	users, total, err := h.userSvc.ListUsers(c.Request.Context(), p.Search, div, p.PerPage, p.Offset())
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
	user, err := h.userSvc.UpdateUser(c.Request.Context(), id, req, getActorName(c))
	if err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
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
	if err := h.userSvc.DeactivateUser(c.Request.Context(), id, getActorName(c)); err != nil {
		response.HandleAppError(c, response.NotFound(err.Error()))
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
	if err := h.userSvc.AssignRole(c.Request.Context(), id, req, getActorName(c)); err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, gin.H{"message": "Role berhasil ditetapkan kepada pengguna"})
}
