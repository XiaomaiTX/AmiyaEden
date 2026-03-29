package middleware

import (
	"amiya-eden/global"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Cors 跨域中间件
// 当 server.cors_origins 配置为空时，仅在 debug 模式下允许所有来源；
// 否则只允许配置中声明的来源。
func Cors() gin.HandlerFunc {
	allowed := buildAllowedOrigins()

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowedOrigin := matchOrigin(origin, allowed); allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// buildAllowedOrigins returns the set of allowed origins from config.
// An empty set in non-debug mode means no origin is allowed.
func buildAllowedOrigins() map[string]struct{} {
	cfg := global.Config
	origins := cfg.Server.CORSOrigins
	m := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		if o != "" {
			m[o] = struct{}{}
		}
	}
	return m
}

// matchOrigin returns the origin to set in the header, or "" to deny.
func matchOrigin(origin string, allowed map[string]struct{}) string {
	if len(allowed) > 0 {
		if _, ok := allowed[origin]; ok {
			return origin
		}
		return ""
	}
	// No explicit allowlist: only permit all origins in debug mode
	if global.Config.Server.Mode == "debug" {
		return "*"
	}
	return ""
}
