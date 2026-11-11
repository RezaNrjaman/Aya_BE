package handler

import (
	"net/http"
	"strconv"
	"strings"

	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func getSessionID(c *gin.Context) string {
	if s := c.GetHeader("X-Session-ID"); strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	if s := c.Query("session_id"); strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	if s, err := c.Cookie("session_id"); err == nil && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	sid := uuid.NewString()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_id", sid, 60*60*24*30, "/", "", c.Request.TLS != nil, true)
	return sid
}

func parseID(c *gin.Context, param, errMsg string) (int, *response.AppError) {
	val := c.Param(param)
	id, err := strconv.Atoi(val)
	if err != nil || id <= 0 {
		return 0, response.ValidationError(errMsg, map[string]string{param: "harus berupa bilangan bulat positif"})
	}
	return id, nil
}

func getActor(c *gin.Context) string {
	if email, ok := c.Get("user_email"); ok {
		return email.(string)
	}
	return "anonymous"
}
