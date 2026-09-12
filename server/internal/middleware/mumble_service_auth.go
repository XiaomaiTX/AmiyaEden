package middleware

import (
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireMumbleService authenticates only the configured service-to-service
// credential. Seat JWTs are intentionally not accepted at this boundary.
func RequireMumbleService() gin.HandlerFunc {
	return func(c *gin.Context) {
		configured := strings.TrimSpace(service.NewSysConfigService().GetMumbleConfig().ServiceToken)
		provided := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if configured == "" || provided == "" || subtle.ConstantTimeCompare([]byte(configured), []byte(provided)) != 1 {
			response.Fail(c, response.CodeUnauthorized, "服务身份验证失败")
			c.Abort()
			return
		}
		c.Next()
	}
}
