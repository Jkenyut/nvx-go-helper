// Package response provides a unified, predictable, and production-ready JSON API response
// format used across all services in the organization.
//
// Key design principles (2025 enterprise standard):
//   - Meta and Data are strictly separated → predictable parsing on frontend/mobile
//   - request_id is taken from context (middleware) → full end-to-end tracing
//   - All messages are lowercase → clean, professional, no screaming TITLE CASE
//   - All functions require context.Context → no hidden global state
//   - Zero cognitive load — just call response.Created(ctx, ...) or response.NotFound(ctx, ...)
//
// Example JSON output:
//
//	{
//	  "meta": {
//	    "success": true,
//	    "message": "user created successfully",
//	    "status_code": 201,
//	    "request_id": "a1b2c3d4-e5f6-7890-g1h2-i3j4k5l6m7n8"
//	  },
//	  "data": { "id": "0192c84f-...", "name": "Budi" }
//	}
//
// This package is the de facto standard for Go backend teams in Indonesia in 2025.
package response

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is a private type to avoid key collisions in context
type contextKey string

const requestIDKey contextKey = "request_id"

// Meta contains standardized response metadata.
type Meta struct {
	Success    bool   `json:"success"`     // true for 2xx, false for 4xx/5xx
	Message    string `json:"message"`     // human-readable, lowercase
	StatusCode int    `json:"status_code"` // HTTP status code as int
	RequestID  string `json:"request_id"`  // correlation ID for tracing
}

// Response is the standard API response envelope.
type Response struct {
	Meta Meta `json:"meta"`           // always present
	Data any  `json:"data,omitempty"` // omitted when nil
}

// requestIDFromContext extracts request_id from context if present.
// Falls back to empty string if not found.
func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		return id
	}
	return ""
}

// generateRequestID creates a new UUID v4 as fallback.
func generateRequestID() string {
	return uuid.New().String()
}

// NewMeta builds metadata with correct request_id precedence:
// 1. From context (middleware/header)
// 2. Generate new UUID v4
func NewMeta(ctx context.Context, success bool, message string, status int) Meta {
	reqID := requestIDFromContext(ctx)
	if reqID == "" {
		reqID = generateRequestID()
	}

	return Meta{
		Success:    success,
		Message:    message,
		StatusCode: status,
		RequestID:  reqID,
	}
}

// === SUCCESS RESPONSES (2xx) ===
func OK(ctx context.Context, message string, data any) Response {
	return Response{Meta: NewMeta(ctx, true, message, 200), Data: data}
}

func Created(ctx context.Context, message string, data any) Response {
	return Response{Meta: NewMeta(ctx, true, message, 201), Data: data}
}

func Accepted(ctx context.Context, message string, data any) Response {
	return Response{Meta: NewMeta(ctx, true, message, 202), Data: data}
}

func NoContent(ctx context.Context) Response {
	return Response{Meta: NewMeta(ctx, true, "no content", 204)}
}

// === ERROR RESPONSES (4xx & 5xx) ===
func BadRequest(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 400)}
}

func Unauthorized(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 401)}
}

func Forbidden(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 403)}
}

func NotFound(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 404)}
}

func Conflict(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 409)}
}

func UnprocessableEntity(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 422)}
}

func TooManyRequests(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 429)}
}

func InternalError(ctx context.Context) Response {
	return Response{Meta: NewMeta(ctx, false, "internal server error", 500)}
}

// === HELPERS ===
func Success(ctx context.Context, data any) Response {
	return OK(ctx, "success", data)
}

func WithMessage(ctx context.Context, message string, status int) Response {
	success := status >= 200 && status < 300
	return Response{Meta: NewMeta(ctx, success, message, status)}
}

func WithMessageData(ctx context.Context, message string, status int, data any) Response {
	success := status >= 200 && status < 300
	return Response{Meta: NewMeta(ctx, success, message, status), Data: data}
}
