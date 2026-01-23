package activity

import (
	"context"
)

// key defines a custom type for context keys to avoid collisions.
type key int

// Context keys constants
const (
	TransactionID key = iota
	MerchantID
	RequestIDKey
	UserID
	UserType
	UserIP
)

func WithTransactionID(ctx context.Context, trxID string) context.Context {
	return context.WithValue(ctx, TransactionID, trxID)
}

// GetTransactionID retrieves the transaction ID from the context.
func GetTransactionID(ctx context.Context) (string, bool) {
	// Type assertion to ensure safety
	trxID, ok := ctx.Value(TransactionID).(string)
	return trxID, ok
}

func WithMerchantID(ctx context.Context, merchantID string) context.Context {
	return context.WithValue(ctx, MerchantID, merchantID)
}

func GetMerchantID(ctx context.Context) (string, bool) {
	merchantID, ok := ctx.Value(MerchantID).(string)
	return merchantID, ok
}

// WithRequestID adds a request ID to the context.
// Useful for distributed tracing.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

// GetFields collects all activity-related fields from the context into a map.
// Useful for structured logging.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserID, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserID).(string)
	return userID, ok
}

func WithUserType(ctx context.Context, userType string) context.Context {
	return context.WithValue(ctx, UserType, userType)
}

func GetUserType(ctx context.Context) (string, bool) {
	userType, ok := ctx.Value(UserType).(string)
	return userType, ok
}

func WithUserIP(ctx context.Context, userIP string) context.Context {
	return context.WithValue(ctx, UserIP, userIP)
}

func GetUserIP(ctx context.Context) (string, bool) {
	userIP, ok := ctx.Value(UserIP).(string)
	return userIP, ok
}

func WithCustomFields(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

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
	if merchantID, ok := GetMerchantID(ctx); ok {
		fields["nvx_merchant_id"] = merchantID // from client
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

func GetFieldValueFromContext[T any](ctx context.Context, key any) (T, bool) {
	u, ok := ctx.Value(key).(T)
	return u, ok
}
