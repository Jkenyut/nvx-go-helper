// Package activity provides context-based helpers for tracking request metadata.
//
// It is used to propagate:
//   - Request IDs (tracing)
//   - Transaction IDs (business logic)
//   - User context (ID, type, IP)
//   - Merchant keys
//
// All values are stored in context.Context and are thread-safe.
package activity

import (
	"context"
)

// key defines a custom type for context keys to avoid collisions.
// Unexported to prevent external usage.
type key int

// Context keys constants
const (
	TransactionID key = iota
	APIKey
	RequestID
	UserID
	UserType
	UserIP
)

// WithTransactionID adds a transaction ID to the context.
// Used by middleware or when initiating a new business transaction.
func WithTransactionID(ctx context.Context, trxID string) context.Context {
	return context.WithValue(ctx, TransactionID, trxID)
}

// GetTransactionID retrieves the transaction ID from the context.
func GetTransactionID(ctx context.Context) (string, bool) {
	// Type assertion to ensure safety
	trxID, ok := ctx.Value(TransactionID).(string)
	return trxID, ok
}

// WithAPIKey adds a merchant key to the context.
func WithAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, APIKey, apiKey)
}

// GetAPIKey retrieves the merchant key from the context.
func GetAPIKey(ctx context.Context) (string, bool) {
	apiKey, ok := ctx.Value(APIKey).(string)
	return apiKey, ok
}

// WithRequestID adds a request ID to the context.
// Useful for distributed tracing across microservices.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestID, requestID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestID).(string)
	return requestID, ok
}

// WithUserID adds a user ID to the context.
// Typically set by authentication middleware.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserID, userID)
}

// GetUserID retrieves the user ID from the context.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserID).(string)
	return userID, ok
}

// WithUserType adds a user type (e.g., "admin", "customer") to the context.
func WithUserType(ctx context.Context, userType string) context.Context {
	return context.WithValue(ctx, UserType, userType)
}

// GetUserType retrieves the user type from the context.
func GetUserType(ctx context.Context) (string, bool) {
	userType, ok := ctx.Value(UserType).(string)
	return userType, ok
}

// WithUserIP adds a user IP address to the context.
func WithUserIP(ctx context.Context, userIP string) context.Context {
	return context.WithValue(ctx, UserIP, userIP)
}

// GetUserIP retrieves the user IP address from the context.
func GetUserIP(ctx context.Context) (string, bool) {
	userIP, ok := ctx.Value(UserIP).(string)
	return userIP, ok
}

// WithCustomFields adds any key-value pair to the context.
// Warning: Use specific functions above when possible to avoid key collisions.
func WithCustomFields(ctx context.Context, k string, value interface{}) context.Context {
	return context.WithValue(ctx, k, value)
}

// GetAllFieldsFromContext collects all standard activity fields into a map.
// Useful for structured logging setup (e.g. Logrus/Zap fields).
func GetAllFieldsFromContext(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// Add transaction_id if present
	if id, ok := GetTransactionID(ctx); ok {
		fields["nvx_transaction_id"] = id // generate by middleware
	}

	// Add request_id if present
	if requestID, ok := GetRequestID(ctx); ok {
		fields["nvx_request_id"] = requestID // from client
	}

	// Add client_id if present
	if apiKey, ok := GetAPIKey(ctx); ok {
		fields["nvx_api_key"] = apiKey // from client
	}

	if userID, ok := GetUserID(ctx); ok {
		// Add payload and result (can be nil)
		fields["nvx_user_id"] = userID // from token
	}

	if userType, ok := GetUserType(ctx); ok {
		fields["nvx_user_type"] = userType // from token
	}

	if userIP, ok := GetUserIP(ctx); ok {
		fields["nvx_user_ip"] = userIP // from client
	}

	return fields
}

// GetFieldValueFromContext is a generic helper to retrieve any value from context safely.
func GetFieldValueFromContext[T any](ctx context.Context, k any) (T, bool) {
	u, ok := ctx.Value(k).(T)
	return u, ok
}
