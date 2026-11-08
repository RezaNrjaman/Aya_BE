package handler

import (
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: svc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Email dan kata sandi wajib diisi", nil))
		return
	}
	res, err := h.authSvc.Login(c.Request.Context(), req)
	if err != nil {
		response.HandleAppError(c, response.Unauthorized(err.Error()))
		return
	}
	response.OK(c, res)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	user := middleware.GetAuthUser(c)
	if user == nil {
		response.HandleAppError(c, response.Unauthorized("Autentikasi diperlukan"))
		return
	}
	profile, err := h.authSvc.GetProfile(c.Request.Context(), user.UserID)
	if err != nil {
		response.HandleAppError(c, response.NotFound("Profil pengguna tidak ditemukan"))
		return
	}
	response.OK(c, profile)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user := middleware.GetAuthUser(c)
	if user == nil {
		response.HandleAppError(c, response.Unauthorized("Autentikasi diperlukan"))
		return
	}
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Kata sandi lama dan baru wajib diisi", nil))
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), user.UserID, req); err != nil {
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, gin.H{"message": "Kata sandi berhasil diperbarui"})
}
