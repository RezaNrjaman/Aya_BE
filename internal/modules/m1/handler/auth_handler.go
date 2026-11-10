package handler

import (
	"context"
	"errors"

	"araya-business-hub/be/internal/modules/m1/domain"
	"araya-business-hub/be/internal/modules/m1/dto"
	"araya-business-hub/be/internal/modules/m1/service"
	"araya-business-hub/be/internal/platform/middleware"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

// AccountActivator consumes single-use activation invitation tokens (ARY-002).
type AccountActivator interface {
	ActivateAccount(ctx context.Context, token, password string) error
}

type AuthHandler struct {
	authSvc   *service.AuthService
	activator AccountActivator
}

func NewAuthHandler(svc *service.AuthService, activator AccountActivator) *AuthHandler {
	return &AuthHandler{authSvc: svc, activator: activator}
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
		response.HandleAppError(c, response.Unauthorized("Sesi tidak berlaku: akun tidak aktif atau tidak ditemukan"))
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

// ActivateAccount finishes partner onboarding: the invitation token (delivered
// out-of-band to the verified contact) is exchanged for a first strong
// password. ARY-002.
func (h *AuthHandler) ActivateAccount(c *gin.Context) {
	var req dto.ActivateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleAppError(c, response.ValidationError("Token dan kata sandi baru wajib diisi", nil))
		return
	}
	if h.activator == nil {
		response.HandleAppError(c, response.InternalError(errors.New("aktivasi tidak tersedia")))
		return
	}
	if err := h.activator.ActivateAccount(c.Request.Context(), req.Token, req.Password); err != nil {
		if errors.Is(err, domain.ErrActivationTokenInvalid) || errors.Is(err, domain.ErrActivationTokenExpired) {
			response.HandleAppError(c, response.BadRequest(err.Error()))
			return
		}
		response.HandleAppError(c, response.ValidationError(err.Error(), nil))
		return
	}
	response.OK(c, gin.H{"message": "Akun berhasil diaktivasi. Silakan masuk dengan kata sandi baru Anda."})
}
