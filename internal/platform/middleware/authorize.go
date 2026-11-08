package middleware

import (
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

		if !policy.CanRolePerform(c.Request.Context(), user.Role, permissionKey) {
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
