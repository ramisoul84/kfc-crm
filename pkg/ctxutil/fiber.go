package ctxutil

import (
	"github.com/gofiber/fiber/v2"
)

// GetRequestIDFromFiber reads the request ID from Fiber locals.
func GetRequestIDFromFiber(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserIDFromFiber reads the user ID from Fiber locals.
func GetUserIDFromFiber(c *fiber.Ctx) string {
	if id, ok := c.Locals(UserIDKey).(string); ok {
		return id
	}
	return ""
}
