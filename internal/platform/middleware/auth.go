package middleware

import (
	"strings"

	"araya-business-hub/be/internal/platform/audit"
	"araya-business-hub/be/internal/platform/response"
	"araya-business-hub/be/internal/platform/security"
	"github.com/gin-gonic/gin"
)

const (
	ContextUserIDKey      = "auth_user_id"
	ContextUserNameKey    = "auth_name"
	ContextUserRoleKey    = "auth_role"
	ContextDivisionKey    = "auth_division"
	ContextUnitUsahaIDKey = "auth_unit_usaha_id"
	ContextPartnerIDKey   = "auth_partner_id"
	ContextClaimsKey      = "auth_claims"
)

// Authenticate returns a middleware checking the Authorization: Bearer <JWT> header.
func Authenticate(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.HandleAppError(c, response.Unauthorized("Header otorisasi tidak ditemukan"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.HandleAppError(c, response.Unauthorized("Format header otorisasi harus 'Bearer <token>'"))
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if tokenStr == "" {
			response.HandleAppError(c, response.Unauthorized("Token tidak boleh kosong"))
			c.Abort()
			return
		}

		claims, err := security.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			response.HandleAppError(c, response.Unauthorized(err.Error()))
			c.Abort()
			return
		}

		// Inject authenticated claims into Gin Context and tenant scope into
		// request context so audit events inherit the authenticated partner.
		c.Request = c.Request.WithContext(audit.WithPartnerID(c.Request.Context(), claims.PartnerID))
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUserNameKey, claims.Name)
		c.Set(ContextUserRoleKey, claims.Role)
		c.Set(ContextDivisionKey, claims.Division)
		c.Set(ContextUnitUsahaIDKey, claims.UnitUsahaID)
		c.Set(ContextPartnerIDKey, claims.PartnerID)
		c.Set(ContextClaimsKey, claims)

		c.Next()
	}
}

// AuthUser captures the context information of the authenticated user.
type AuthUser struct {
	UserID      int
	Name        string
	Role        string
	Division    string
	UnitUsahaID *int
	PartnerID   *int
}

// GetAuthUser extracts authenticated user info from the context.
func GetAuthUser(c *gin.Context) *AuthUser {
	uid, exists := c.Get(ContextUserIDKey)
	if !exists {
		return nil
	}

	userID, _ := uid.(int)
	name, _ := c.Get(ContextUserNameKey)
	role, _ := c.Get(ContextUserRoleKey)
	division, _ := c.Get(ContextDivisionKey)
	unitUsahaID, _ := c.Get(ContextUnitUsahaIDKey)

	partnerID, _ := c.Get(ContextPartnerIDKey)

	user := &AuthUser{
		UserID:   userID,
		Name:     name.(string),
		Role:     role.(string),
		Division: division.(string),
	}

	if unitUsahaID != nil {
		if idPtr, ok := unitUsahaID.(*int); ok {
			user.UnitUsahaID = idPtr
		}
	}
	if partnerID != nil {
		if idPtr, ok := partnerID.(*int); ok {
			user.PartnerID = idPtr
		}
	}

	return user
}
