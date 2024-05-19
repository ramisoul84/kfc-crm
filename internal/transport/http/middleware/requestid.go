package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
)

// RequestIDHeader is the header name used for the request ID.
const RequestIDHeader = "X-Request-ID"

// RequestID generates a unique ID for each request (or reuses the incoming
// one if present), stores it in the Fiber context and locals, and echoes it
// back in the response header.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		// Store in the standard context so services can read it.
		ctx := ctxutil.WithRequestID(c.UserContext(), id)
		c.SetUserContext(ctx)

		// Store in Fiber locals for handlers/middleware.
		c.Locals(ctxutil.RequestIDKey, id)

		// Echo back to the client.
		c.Set(RequestIDHeader, id)

		return c.Next()
	}
}
