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

// Context keys constants — unexported to prevent external access.
// Use the With*/Get* functions to interact with these values.
const (
	transactionID key = iota
	apiKey
	requestID
	userID
	userType
	userIP
)

// customKey is a typed key for custom fields, preventing collisions
// with other packages that also use string-based context keys.
type customKey string

// WithTransactionID adds a transaction ID to the context.
// Used by middleware or when initiating a new business transaction.
func WithTransactionID(ctx context.Context, trxID string) context.Context {
	return context.WithValue(ctx, transactionID, trxID)
}

// GetTransactionID retrieves the transaction ID from the context.
func GetTransactionID(ctx context.Context) (string, bool) {
	// Type assertion to ensure safety
	trxID, ok := ctx.Value(transactionID).(string)
	return trxID, ok
}

// WithAPIKey adds a merchant key to the context.
func WithAPIKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, apiKey, key)
}

// GetAPIKey retrieves the merchant key from the context.
func GetAPIKey(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(apiKey).(string)
	return v, ok
}

// WithRequestID adds a request ID to the context.
// Useful for distributed tracing across microservices.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestID, reqID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestID).(string)
	return v, ok
}

// WithUserID adds a user ID to the context.
// Typically set by authentication middleware.
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, userID, uid)
}

// GetUserID retrieves the user ID from the context.
func GetUserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userID).(string)
	return v, ok
}

// WithUserType adds a user type (e.g., "admin", "customer") to the context.
func WithUserType(ctx context.Context, utype string) context.Context {
	return context.WithValue(ctx, userType, utype)
}

// GetUserType retrieves the user type from the context.
func GetUserType(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userType).(string)
	return v, ok
}

// WithUserIP adds a user IP address to the context.
func WithUserIP(ctx context.Context, uip string) context.Context {
	return context.WithValue(ctx, userIP, uip)
}

// GetUserIP retrieves the user IP address from the context.
func GetUserIP(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIP).(string)
	return v, ok
}

// WithCustomFields adds any key-value pair to the context using a typed key
// to prevent collisions with other packages.
// Use specific functions above when possible for standard fields.
func WithCustomFields(ctx context.Context, k string, value interface{}) context.Context {
	return context.WithValue(ctx, customKey(k), value)
}

// GetCustomField retrieves a custom field value from the context.
// The key must match the one used in WithCustomFields.
func GetCustomField(ctx context.Context, k string) (interface{}, bool) {
	v := ctx.Value(customKey(k))
	if v == nil {
		return nil, false
	}
	return v, true
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
	if v, ok := GetRequestID(ctx); ok {
		fields["nvx_request_id"] = v // from client
	}

	// Add client_id if present
	if v, ok := GetAPIKey(ctx); ok {
		fields["nvx_api_key"] = v // from client
	}

	if v, ok := GetUserID(ctx); ok {
		// Add payload and result (can be nil)
		fields["nvx_user_id"] = v // from token
	}

	if v, ok := GetUserType(ctx); ok {
		fields["nvx_user_type"] = v // from token
	}

	if v, ok := GetUserIP(ctx); ok {
		fields["nvx_user_ip"] = v // from client
	}

	return fields
}

// GetFieldValueFromContext is a generic helper to retrieve any value from context safely.
func GetFieldValueFromContext[T any](ctx context.Context, k any) (T, bool) {
	u, ok := ctx.Value(k).(T)
	return u, ok
}
