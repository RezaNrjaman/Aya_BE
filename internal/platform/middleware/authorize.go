package middleware

import (
	"errors"

	"araya-business-hub/be/internal/platform/rbac"
	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

// RequirePermission ensures the authenticated user's role has the requested permission key.
func RequirePermission(policy *rbac.PolicyService, permissionKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetAuthUser(c)
		if user == nil {
			response.HandleAppError(c, response.Unauthorized("Autentikasi diperlukan"))
			c.Abort()
			return
		}

		role, ok := resolveCurrentRole(c, policy, user)
		if !ok {
			return
		}

		if !policy.CanRolePerform(c.Request.Context(), role, permissionKey) {
			response.HandleAppError(c, response.Forbidden("Anda tidak memiliki hak akses untuk tindakan ini ("+permissionKey+")"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireUnitScope validates that the user is authorized to operate on the specified unit_usaha_id.
func RequireUnitScope(policy *rbac.PolicyService, targetUnitUsahaID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetAuthUser(c)
		if user == nil {
			response.HandleAppError(c, response.Unauthorized("Autentikasi diperlukan"))
			c.Abort()
			return
		}

		if !policy.IsScopePermitted(user.Role, user.UnitUsahaID, targetUnitUsahaID) {
			response.HandleAppError(c, response.Forbidden("Akses ditolak: Anda tidak memiliki izin untuk unit usaha ini"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// resolveCurrentRole replaces the role from the token with the user's current
// role (when M1 registered a resolver) so that role changes and deactivation
// apply to sessions that are already open. It aborts the request on failure.
func resolveCurrentRole(c *gin.Context, policy *rbac.PolicyService, user *AuthUser) (string, bool) {
	role, err := policy.CurrentRole(c.Request.Context(), user.UserID, user.Role)
	if err == nil {
		c.Set(ContextUserRoleKey, role)
		return role, true
	}
	if errors.Is(err, rbac.ErrAccessRevoked) {
		response.HandleAppError(c, response.Unauthorized("Akun tidak aktif atau tidak ditemukan, silakan login kembali"))
	} else {
		response.HandleAppError(c, response.InternalError(err))
	}
	c.Abort()
	return "", false
}
