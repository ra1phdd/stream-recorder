package noCache

import (
	"github.com/gin-gonic/gin"
	"time"
)

func NoCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", time.Now().Add(-1*time.Second).Format(time.RFC1123))
		c.Next()
	}
}
