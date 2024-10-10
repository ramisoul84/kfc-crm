package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// Logging logs every HTTP request with method, path, status, and duration.
// Health checks are skipped to reduce noise.
func Logging(log *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		if c.Path() == "/health" {
			return err
		}

		requestID := ctxutil.GetRequestIDFromFiber(c)
		user := ctxutil.GetUserFromFiber(c)

		fields := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
		}

		if user != nil {
			fields = append(fields, "user_id", user.ID.String())
		}

		reqLog := log.WithRequestID(requestID)

		status := c.Response().StatusCode()
		switch {
		case status >= 500:
			reqLog.Error("request completed", fields...)
		case status >= 400:
			reqLog.Warn("request completed", fields...)
		default:
			reqLog.Info("request completed", fields...)
		}

		return err
	}
}
