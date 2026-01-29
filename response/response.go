// Package response provides a unified, predictable, and production-ready JSON API response
// format used across all services in the organization.
//
// Key design principles (enterprise standard):
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
// This package provides a consistent JSON structure for API responses.
package response

import (
	"context"
	"encoding/json"

	"github.com/Jkenyut/nvx-go-helper/activity"
	"github.com/Jkenyut/nvx-go-helper/cryptoutil"
)

// Meta holds the metadata for the API response.
// It contains status information, messages, and tracing IDs.
type Meta struct {
	Success    bool   `json:"success"`     // true for 2xx, false for 4xx/5xx
	Message    string `json:"message"`     // human-readable, lowercase
	StatusCode int    `json:"status_code"` // HTTP status code as int
	RequestID  string `json:"request_id"`  // correlation ID for tracing
}

// Response is the standard top-level JSON structure.
// All API endpoints must return this structure.
type Response struct {
	Meta Meta `json:"meta"`           // always present
	Data any  `json:"data,omitempty"` // omitted when nil
}

// NewMeta builds metadata with correct request_id precedence:
// 1. From context (middleware/header)
// 2. Generate new UUID v7
func NewMeta(ctx context.Context, success bool, message string, status int) Meta {
	// Try to get request ID from context
	reqID, _ := activity.GetRequestID(ctx)
	// If not found, generate a new random UUID v4
	if reqID == "" {
		reqID = cryptoutil.V7()
	}

	// Return the constructed Meta struct
	return Meta{
		Success:    success, // Success status
		Message:    message, // Message string
		StatusCode: status,  // HTTP status code
		RequestID:  reqID,   // Tracing ID
	}
}

// === SUCCESS RESPONSES (2xx) ===

// OK sends a 200 OK response with data.
func OK(ctx context.Context, message string, data any) Response {
	return Response{Meta: NewMeta(ctx, true, message, 200), Data: data}
}

// Created sends a 201 Created response with data.
func Created(ctx context.Context, message string, data any) Response {
	return Response{Meta: NewMeta(ctx, true, message, 201), Data: data}
}

// Accepted sends a 202 Accepted response with data.
func Accepted(ctx context.Context, message string, data any) Response {
	return Response{Meta: NewMeta(ctx, true, message, 202), Data: data}
}

// NoContent sends a 204 No Content response.
func NoContent(ctx context.Context) Response {
	return Response{Meta: NewMeta(ctx, true, "no content", 204)}
}

// === ERROR RESPONSES (4xx & 5xx) ===

// BadRequest sends a 400 Bad Request response.
func BadRequest(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 400)}
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 401)}
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 403)}
}

// NotFound sends a 404 Not Found response.
func NotFound(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 404)}
}

// Conflict sends a 409 Conflict response.
func Conflict(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 409)}
}

// UnprocessableEntity sends a 422 Unprocessable Entity response.
func UnprocessableEntity(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 422)}
}

// TooManyRequests sends a 429 Too Many Requests response.
func TooManyRequests(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 429)}
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(ctx context.Context) Response {
	return Response{Meta: NewMeta(ctx, false, "internal server error", 500)}
}

// MethodNotAllowed sends a 405 Method Not Allowed response.
func MethodNotAllowed(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 405)}
}

// NotAcceptable sends a 406 Not Acceptable response.
func NotAcceptable(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 406)}
}

// RequestTimeout sends a 408 Request Timeout response.
func RequestTimeout(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 408)}
}

// Gone sends a 410 Gone response.
func Gone(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 410)}
}

// PreconditionFailed sends a 412 Precondition Failed response.
func PreconditionFailed(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 412)}
}

// PayloadTooLarge sends a 413 Payload Too Large response.
func PayloadTooLarge(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 413)}
}

// UnsupportedMediaType sends a 415 Unsupported Media Type response.
func UnsupportedMediaType(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 415)}
}

// NotImplemented sends a 501 Not Implemented response.
func NotImplemented(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 501)}
}

// BadGateway sends a 502 Bad Gateway response.
func BadGateway(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 502)}
}

// ServiceUnavailable sends a 503 Service Unavailable response.
func ServiceUnavailable(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 503)}
}

// GatewayTimeout sends a 504 Gateway Timeout response.
func GatewayTimeout(ctx context.Context, message string) Response {
	return Response{Meta: NewMeta(ctx, false, message, 504)}
}

// === HELPERS ===

// Success is a shortcut for OK(ctx, "success", data).
func Success(ctx context.Context, data any) Response {
	return OK(ctx, "success", data)
}

// WithMessage sends a response with a custom message and status code (no data).
func WithMessage(ctx context.Context, message string, status int) Response {
	// Determine success based on status code range
	success := status >= 200 && status < 300
	return Response{Meta: NewMeta(ctx, success, message, status)}
}

// WithMessageData sends a response with a custom message, status code, and data.
func WithMessageData(ctx context.Context, message string, status int, data any) Response {
	// Determine success based on status code range
	success := status >= 200 && status < 300
	return Response{Meta: NewMeta(ctx, success, message, status), Data: data}
}

func (r *Response) JSONMarshal() []byte {
	if r.Meta.StatusCode == 0 {
		r.Meta.StatusCode = 200
	}

	resp, _ := json.Marshal(r)
	return resp
}
