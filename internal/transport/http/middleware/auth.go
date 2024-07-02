package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/jwt"
)

// Auth validates the access token and populates the request context
// with the authenticated user's identity.
func Auth(tokenManager *jwt.TokenManager, tokenRepo repository.TokenRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"type":    "authentication",
					"message": "missing access token",
				},
			})
		}

		// Reject blacklisted tokens (logged-out sessions).
		blacklisted, err := tokenRepo.IsAccessTokenBlacklisted(c.UserContext(), token)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"type":    "internal",
					"message": "internal server error",
				},
			})
		}
		if blacklisted {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"type":    "authentication",
					"message": "token revoked",
				},
			})
		}

		// Validate the token signature and expiry.
		claims, err := tokenManager.ValidateAccessToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"type":    "authentication",
					"message": "invalid or expired token",
				},
			})
		}

		// Populate the standard context for services.
		ctx := ctxutil.WithUserID(c.UserContext(), claims.UserID.String())
		c.SetUserContext(ctx)

		// Populate Fiber locals for handlers and other middleware.
		c.Locals(ctxutil.UserIDKey, claims.UserID.String())
		c.Locals(ctxutil.EmailKey, claims.Email)
		c.Locals(ctxutil.RoleKey, string(claims.Role))

		return c.Next()
	}
}

// extractBearerToken extracts the bearer token from the Authorization header.
func extractBearerToken(c *fiber.Ctx) string {
	header := c.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
