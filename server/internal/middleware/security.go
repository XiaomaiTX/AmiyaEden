package middleware

import "github.com/gin-gonic/gin"

// SecureHeaders adds standard security headers to all responses.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "0") // modern browsers use CSP; disable legacy filter
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
