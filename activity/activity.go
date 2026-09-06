// Package activity provides context-based helpers for tracking request metadata.
//
// It is used to propagate:
//   - Request IDs (tracing)
//   - Transaction IDs (business logic)
//   - User context (ID, IP)
//   - Arbitrary metadata
//
// All values are stored in context.Context and are thread-safe.
package activity

import (
	"context"
	"log/slog"
)

// key defines a custom type for context keys to avoid collisions.
// Unexported to prevent external usage.
type key int

// Context keys constants — unexported to prevent external access.
// Use the With*/Get* functions to interact with these values.
const (
	transactionID key = iota
	requestID
	userID
	userIP
	userIPOrigin
)

// customKey is a typed key for custom fields, preventing collisions
// with other packages that also use string-based context keys.
type customKey string

// metadataKey is a typed key for custom metadata map.
type metadataKey struct{}

// Activity encapsulates standard contextual tracking metadata.
type Activity struct {
	TransactionID string `json:"transaction_id,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	UserIP        string `json:"user_ip,omitempty"`
	UserIPOrigin  string `json:"user_ip_origin,omitempty"`
}

// WithTransactionID adds a transaction ID to the context.
// Used by middleware or when initiating a new business transaction.
func WithTransactionID(ctx context.Context, trxID string) context.Context {
	return context.WithValue(ctx, transactionID, trxID)
}

// GetTransactionID retrieves the transaction ID from the context.
func GetTransactionID(ctx context.Context) (string, bool) {
	trxID, ok := ctx.Value(transactionID).(string)
	return trxID, ok
}

// WithRequestID adds a request ID to the context.
// Useful for tracing per-hop HTTP requests.
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

// WithUserIP adds a user IP address to the context.
func WithUserIP(ctx context.Context, uip string) context.Context {
	return context.WithValue(ctx, userIP, uip)
}

// GetUserIP retrieves the user IP address from the context.
func GetUserIP(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIP).(string)
	return v, ok
}

// WithUserIPOrigin adds the original user IP (from proxy) to the context.
func WithUserIPOrigin(ctx context.Context, uip string) context.Context {
	return context.WithValue(ctx, userIPOrigin, uip)
}

// GetUserIPOrigin retrieves the user IP address from the context.
func GetUserIPOrigin(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIPOrigin).(string)
	return v, ok
}

// WithActivity injects all non-empty fields of an Activity struct into the context.
func WithActivity(ctx context.Context, act Activity) context.Context {
	if act.TransactionID != "" {
		ctx = WithTransactionID(ctx, act.TransactionID)
	}
	if act.RequestID != "" {
		ctx = WithRequestID(ctx, act.RequestID)
	}
	if act.UserID != "" {
		ctx = WithUserID(ctx, act.UserID)
	}
	if act.UserIP != "" {
		ctx = WithUserIP(ctx, act.UserIP)
	}
	if act.UserIPOrigin != "" {
		ctx = WithUserIPOrigin(ctx, act.UserIPOrigin)
	}
	return ctx
}

// FromContext extracts all standard activity fields from context into an Activity struct.
func FromContext(ctx context.Context) Activity {
	var act Activity
	act.TransactionID, _ = GetTransactionID(ctx)
	act.RequestID, _ = GetRequestID(ctx)
	act.UserID, _ = GetUserID(ctx)
	act.UserIP, _ = GetUserIP(ctx)
	act.UserIPOrigin, _ = GetUserIPOrigin(ctx)
	return act
}

// WithCustomFields adds any key-value pair to the context using a typed key
// to prevent collisions with other packages.
// Use specific functions above when possible for standard fields.
func WithCustomFields(ctx context.Context, k string, value any) context.Context {
	return context.WithValue(ctx, customKey(k), value)
}

// GetCustomField retrieves a custom field value from the context.
// The key must match the one used in WithCustomFields.
func GetCustomField(ctx context.Context, k string) (any, bool) {
	v := ctx.Value(customKey(k))
	if v == nil {
		return nil, false
	}
	return v, true
}

// WithMetadata attaches an arbitrary metadata map to the context.
// If metadata already exists in the context, new entries will be merged.
func WithMetadata(ctx context.Context, metadata map[string]any) context.Context {
	if len(metadata) == 0 {
		return ctx
	}
	existing, _ := ctx.Value(metadataKey{}).(map[string]any)
	merged := make(map[string]any, len(existing)+len(metadata))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range metadata {
		merged[k] = v
	}
	return context.WithValue(ctx, metadataKey{}, merged)
}

// GetMetadata retrieves the custom metadata map from context.
func GetMetadata(ctx context.Context) (map[string]any, bool) {
	v, ok := ctx.Value(metadataKey{}).(map[string]any)
	return v, ok
}

// GetAllFieldsFromContext collects all standard activity fields and attached metadata into a map.
// Useful for structured logging setup (e.g. Zerolog Fields, Logrus/Zap fields).
func GetAllFieldsFromContext(ctx context.Context) map[string]any {
	meta, hasMeta := GetMetadata(ctx)
	capHint := 5
	if hasMeta {
		capHint += len(meta)
	}
	fields := make(map[string]any, capHint)

	// Copy custom metadata if present
	for k, v := range meta {
		fields[k] = v
	}

	if id, ok := GetTransactionID(ctx); ok {
		fields["transaction_id"] = id
	}
	if v, ok := GetRequestID(ctx); ok {
		fields["request_id"] = v
	}
	if v, ok := GetUserID(ctx); ok {
		fields["user_id"] = v
	}
	if v, ok := GetUserIP(ctx); ok {
		fields["user_ip"] = v
	}
	if v, ok := GetUserIPOrigin(ctx); ok {
		fields["user_ip_origin"] = v
	}

	return fields
}

// ToSlogAttrs extracts all standard activity fields from context as a slice of slog.Attr.
// It enables zero-allocation, high-performance integration with Go's log/slog package.
func ToSlogAttrs(ctx context.Context) []slog.Attr {
	attrs := make([]slog.Attr, 0, 5)

	if v, ok := GetTransactionID(ctx); ok {
		attrs = append(attrs, slog.String("transaction_id", v))
	}
	if v, ok := GetRequestID(ctx); ok {
		attrs = append(attrs, slog.String("request_id", v))
	}
	if v, ok := GetUserID(ctx); ok {
		attrs = append(attrs, slog.String("user_id", v))
	}
	if v, ok := GetUserIP(ctx); ok {
		attrs = append(attrs, slog.String("user_ip", v))
	}
	if v, ok := GetUserIPOrigin(ctx); ok {
		attrs = append(attrs, slog.String("user_ip_origin", v))
	}

	return attrs
}

// GetFieldValueFromContext is a generic helper to retrieve any value from context safely.
func GetFieldValueFromContext[T any](ctx context.Context, k any) (T, bool) {
	u, ok := ctx.Value(k).(T)
	return u, ok
}
