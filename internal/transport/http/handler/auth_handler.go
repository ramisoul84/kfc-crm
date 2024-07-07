package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/service"
	"github.com/ramisoul84/kfc-crm/internal/transport/http/response"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/validator"
)

// Cookie name and path for the refresh token.
const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/v1/auth"
	refreshTokenTTL        = 7 * 24 * time.Hour
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService service.AuthService
	validator   *validator.Validator
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authService service.AuthService, v *validator.Validator) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   v,
	}
}

// ═══════════════════════════════════════════════════════════════════
// LOGIN
// ═══════════════════════════════════════════════════════════════════

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req domain.UserLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, domain.NewValidationError("invalid request body"))
	}

	if err := h.validator.Validate(&req); err != nil {
		return response.Error(c, err)
	}

	pair, err := h.authService.Login(c.UserContext(), &req)
	if err != nil {
		return response.Error(c, err)
	}

	h.setRefreshCookie(c, pair.RefreshToken)

	return response.Success(c, fiber.Map{
		"access_token":      pair.AccessToken,
		"expires_in":        pair.ExpiresIn,
		"token_type":        pair.TokenType,
		"profile_completed": pair.ProfileCompleted,
	})
}

// ═══════════════════════════════════════════════════════════════════
// REFRESH
// ═══════════════════════════════════════════════════════════════════

// RefreshToken handles POST /api/v1/auth/refresh.
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies(refreshTokenCookieName)
	if refreshToken == "" {
		return response.Error(c, domain.NewAuthenticationError("refresh token not found"))
	}

	pair, err := h.authService.RefreshToken(c.UserContext(), refreshToken)
	if err != nil {
		// Clear the invalid cookie so the client doesn't keep sending it.
		h.clearRefreshCookie(c)
		return response.Error(c, err)
	}

	h.setRefreshCookie(c, pair.RefreshToken)

	return response.Success(c, fiber.Map{
		"access_token":      pair.AccessToken,
		"expires_in":        pair.ExpiresIn,
		"token_type":        pair.TokenType,
		"profile_completed": pair.ProfileCompleted,
	})
}

// ═══════════════════════════════════════════════════════════════════
// LOGOUT
// ═══════════════════════════════════════════════════════════════════

// Logout handles POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	user := ctxutil.GetUserFromFiber(c)
	if user == nil {
		return response.Error(c, domain.NewAuthenticationError("not authenticated"))
	}

	accessToken := extractBearerToken(c)

	if err := h.authService.Logout(c.UserContext(), user.ID, accessToken); err != nil {
		return response.Error(c, err)
	}

	h.clearRefreshCookie(c)

	return response.Success(c, fiber.Map{"message": "logged out successfully"})
}

// ═══════════════════════════════════════════════════════════════════
// COOKIE HELPERS
// ═══════════════════════════════════════════════════════════════════

func (h *AuthHandler) setRefreshCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     refreshTokenCookiePath,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  time.Now().Add(refreshTokenTTL),
		MaxAge:   int(refreshTokenTTL.Seconds()),
	})
}

func (h *AuthHandler) clearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     refreshTokenCookiePath,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  time.Now().Add(-1 * time.Hour),
		MaxAge:   -1,
	})
}

// ═══════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════

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
