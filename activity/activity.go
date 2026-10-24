// Package activity provides context-based helpers for tracking request metadata.
//
// It is used to propagate:
//   - Request IDs (tracing)
//   - Correlation IDs (cross-service tracing)
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
	correlationID key = iota
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
	CorrelationID string `json:"correlation_id,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	UserIP        string `json:"user_ip,omitempty"`
	UserIPOrigin  string `json:"user_ip_origin,omitempty"`
}

// WithCorrelationID adds a correlation ID to the context.
// It returns a new context with the correlation ID.
// It creates a background context if ctx is nil.
func WithCorrelationID(ctx context.Context, trxID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, correlationID, trxID)
}

// GetCorrelationID retrieves the correlation ID from the context.
// It returns an empty string and false if ctx is nil or key is missing.
func GetCorrelationID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	trxID, ok := ctx.Value(correlationID).(string)
	return trxID, ok
}

// WithRequestID adds a request ID to the context.
// It creates a background context if ctx is nil.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestID, reqID)
}

// GetRequestID retrieves the request ID from the context.
// It returns an empty string and false if ctx is nil or key is missing.
func GetRequestID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(requestID).(string)
	return v, ok
}

// WithUserID adds a user ID to the context.
// It creates a background context if ctx is nil.
func WithUserID(ctx context.Context, uid string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userID, uid)
}

// GetUserID retrieves the user ID from the context.
// It returns an empty string and false if ctx is nil or key is missing.
func GetUserID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(userID).(string)
	return v, ok
}

// WithUserIP adds a user IP address to the context.
// It creates a background context if ctx is nil.
func WithUserIP(ctx context.Context, uip string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIP, uip)
}

// GetUserIP retrieves the user IP address from the context.
// It returns an empty string and false if ctx is nil or key is missing.
func GetUserIP(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(userIP).(string)
	return v, ok
}

// WithUserIPOrigin adds the original user IP to the context.
// It creates a background context if ctx is nil.
func WithUserIPOrigin(ctx context.Context, uip string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIPOrigin, uip)
}

// GetUserIPOrigin retrieves the original user IP address from the context.
// It returns an empty string and false if ctx is nil or key is missing.
func GetUserIPOrigin(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(userIPOrigin).(string)
	return v, ok
}

// WithActivity injects standard activity fields into the context.
// It creates a background context if ctx is nil.
func WithActivity(ctx context.Context, act Activity) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if act.CorrelationID != "" {
		ctx = WithCorrelationID(ctx, act.CorrelationID)
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

// FromContext extracts standard activity fields from context into Activity struct.
// It returns an empty struct if ctx is nil.
func FromContext(ctx context.Context) Activity {
	if ctx == nil {
		return Activity{}
	}
	var act Activity
	act.CorrelationID, _ = GetCorrelationID(ctx)
	act.RequestID, _ = GetRequestID(ctx)
	act.UserID, _ = GetUserID(ctx)
	act.UserIP, _ = GetUserIP(ctx)
	act.UserIPOrigin, _ = GetUserIPOrigin(ctx)
	return act
}

// WithCustomFields adds a key-value pair to the context.
// It creates a background context if ctx is nil.
func WithCustomFields(ctx context.Context, k string, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, customKey(k), value)
}

// GetCustomField retrieves a custom field value from the context.
// It returns nil and false if ctx is nil or key is missing.
func GetCustomField(ctx context.Context, k string) (any, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(customKey(k))
	if v == nil {
		return nil, false
	}
	return v, true
}

// WithMetadata attaches an arbitrary metadata map to the context.
// If metadata already exists in the context, new entries will be merged.
// It creates a background context if ctx is nil.
func WithMetadata(ctx context.Context, metadata map[string]any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
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
// It returns nil and false if ctx is nil or metadata is missing.
func GetMetadata(ctx context.Context) (map[string]any, bool) {
	if ctx == nil {
		return nil, false
	}
	v, ok := ctx.Value(metadataKey{}).(map[string]any)
	return v, ok
}

// GetAllFieldsFromContext collects standard activity fields and metadata into a map.
// It returns an empty map if ctx is nil.
func GetAllFieldsFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return make(map[string]any)
	}
	meta, hasMeta := GetMetadata(ctx)
	capHint := 5
	if hasMeta {
		capHint += len(meta)
	}
	fields := make(map[string]any, capHint)

	for k, v := range meta {
		fields[k] = v
	}

	if id, ok := GetCorrelationID(ctx); ok {
		fields["correlation_id"] = id
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

// ToSlogAttrs extracts all standard activity fields from context as slog attributes.
// It returns an empty slice if ctx is nil.
func ToSlogAttrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return make([]slog.Attr, 0)
	}
	attrs := make([]slog.Attr, 0, 5)

	if v, ok := GetCorrelationID(ctx); ok {
		attrs = append(attrs, slog.String("correlation_id", v))
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
// It returns zero value and false if ctx or k is nil.
func GetFieldValueFromContext[T any](ctx context.Context, k any) (T, bool) {
	var zero T
	if ctx == nil || k == nil {
		return zero, false
	}
	u, ok := ctx.Value(k).(T)
	return u, ok
}
