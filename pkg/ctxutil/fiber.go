package ctxutil

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/kfc-crm/internal/domain"
)

// ─────────────────────────────────────────────────────────────────
// Fiber locals
// ─────────────────────────────────────────────────────────────────

// GetRequestIDFromFiber reads the request ID from Fiber locals.
func GetRequestIDFromFiber(c *fiber.Ctx) string {
	if id, ok := c.Locals(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserFromFiber returns the authenticated user stored by the auth middleware.
// Returns nil when the request is not authenticated.
func GetUserFromFiber(c *fiber.Ctx) *domain.User {
	if u, ok := c.Locals(UserKey).(*domain.User); ok {
		return u
	}
	return nil
}
