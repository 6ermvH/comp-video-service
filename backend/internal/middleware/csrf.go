package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireCSRF protects mutating admin endpoints with double-submit header token.
// Expected header: X-CSRF-Token, value must match csrf claim inside JWT.
func RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			c.Next()
			return
		}

		expected, _ := c.Get(CSRFTokenKey)
		expectedToken, _ := expected.(string)
		if strings.TrimSpace(expectedToken) == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token is missing in auth context"})
			return
		}

		provided := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if provided == "" || provided != expectedToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid csrf token"})
			return
		}

		c.Next()
	}
}
