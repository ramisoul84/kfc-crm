package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/jwt"
)

// Auth validates the access token, loads the user from the database,
// and stores the fully populated *domain.User in Fiber locals.
func Auth(
	tokenManager *jwt.TokenManager,
	tokenRepo repository.TokenRepository,
	userRepo repository.UserRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extract token
		token := extractBearerToken(c)
		if token == "" {
			return unauthorized(c, "missing access token")
		}

		// 2. Reject blacklisted tokens
		blacklisted, err := tokenRepo.IsAccessTokenBlacklisted(c.UserContext(), token)
		if err != nil {
			return internalError(c)
		}
		if blacklisted {
			return unauthorized(c, "token revoked")
		}

		// 3. Validate signature and expiry
		claims, err := tokenManager.ValidateAccessToken(token)
		if err != nil {
			return unauthorized(c, "invalid or expired token")
		}

		// 4. Load the user fresh from the database
		user, err := userRepo.GetByID(c.UserContext(), claims.UserID)
		if err != nil {
			return unauthorized(c, "user not found")
		}

		// 5. Reject inactive users immediately
		if !user.IsActive {
			return forbidden(c, "user account is inactive")
		}

		// 6. Store the full user in locals
		c.Locals(ctxutil.UserKey, user)

		// 7. Also store the user ID in the standard context
		//    so services can read it via ctxutil.GetUserID(ctx).
		ctx := ctxutil.WithUserID(c.UserContext(), user.ID.String())
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// ─────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────

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

func unauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"type":    "authentication",
			"message": message,
		},
	})
}

func forbidden(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"type":    "authorization",
			"message": message,
		},
	})
}

func internalError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"type":    "internal",
			"message": "internal server error",
		},
	})
}
