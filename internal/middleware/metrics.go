package middleware

import (
	"time"

	"github.com/Nysonn/unibuzz-api/internal/config"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()

		config.HTTPRequests.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Inc()

		config.HTTPDuration.WithLabelValues(
			c.FullPath(),
		).Observe(duration)
	}
}
