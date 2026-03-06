package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func TenantExtractor() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-Id")
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "MISSING_TENANT",
					"message": "X-Tenant-Id header is required",
				},
			})
			return
		}

		ctx := context.WithValue(c.Request.Context(), "tenant_id", tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("tenant_id", tenantID)

		userID := c.GetHeader("X-User-Id")
		if userID != "" {
			ctx = context.WithValue(c.Request.Context(), "user_id", userID)
			c.Request = c.Request.WithContext(ctx)
			c.Set("user_id", userID)
		}

		c.Next()
	}
}
