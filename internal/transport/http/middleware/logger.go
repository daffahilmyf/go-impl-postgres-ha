package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()

		requestID := c.GetString(RequestIDKey)
		if requestID == "" {
			requestID = c.GetHeader("X-Request-ID")
		}
		entry := log.WithFields(logrus.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"ip":         c.ClientIP(),
			"latency":    time.Since(start).String(),
			"error":      c.Errors.ByType(gin.ErrorTypePrivate).String(),
			"user_agent": c.Request.UserAgent(),
			"request_id": requestID,
		})

		if c.Writer.Status() >= 500 {
			entry.Error("request failed")
			return
		}
		if c.Writer.Status() >= 400 {
			entry.Warn("request client error")
			return
		}
		entry.Info("request handled")
	}
}
