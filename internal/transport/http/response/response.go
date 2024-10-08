package response

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/validator"
)

// Response is the standard API response envelope.
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// ErrorInfo holds error details.
type ErrorInfo struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════
// SUCCESS RESPONSES
// ═══════════════════════════════════════════════════════════════════

// Success sends a 200 OK response.
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(Response{
		Success:   true,
		Data:      data,
		RequestID: ctxutil.GetRequestIDFromFiber(c),
		Timestamp: time.Now().Unix(),
	})
}

// Created sends a 201 Created response.
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Data:      data,
		RequestID: ctxutil.GetRequestIDFromFiber(c),
		Timestamp: time.Now().Unix(),
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════
// ERROR RESPONSES
// ═══════════════════════════════════════════════════════════════════

// Error sends an error response with the correct HTTP status.
func Error(c *fiber.Ctx, err error) error {
	requestID := ctxutil.GetRequestIDFromFiber(c)

	// Validation errors → 400 with details
	if ve, ok := validator.AsValidationErrors(err); ok {
		return c.Status(fiber.StatusBadRequest).JSON(Response{
			Success: false,
			Error: &ErrorInfo{
				Type:    "validation",
				Message: "validation failed",
				Details: ve.Errors,
			},
			RequestID: requestID,
			Timestamp: time.Now().Unix(),
		})
	}

	// Domain errors → their status
	if appErr, ok := domain.AsAppError(err); ok {
		return c.Status(appErr.StatusCode).JSON(Response{
			Success: false,
			Error: &ErrorInfo{
				Type:    string(appErr.Type),
				Message: appErr.Message,
				Details: appErr.Details,
			},
			RequestID: requestID,
			Timestamp: time.Now().Unix(),
		})
	}

	// Fallback → 500
	return c.Status(fiber.StatusInternalServerError).JSON(Response{
		Success: false,
		Error: &ErrorInfo{
			Type:    "internal",
			Message: "internal server error",
		},
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	})
}

// ErrorWithStatus sends an error response with an explicit status code.
// Useful for middleware that isn't using domain.AppError.
func ErrorWithStatus(c *fiber.Ctx, status int, errType string, message string) error {
	return c.Status(status).JSON(Response{
		Success: false,
		Error: &ErrorInfo{
			Type:    errType,
			Message: message,
		},
		RequestID: ctxutil.GetRequestIDFromFiber(c),
		Timestamp: time.Now().Unix(),
	})
}
