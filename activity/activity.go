package activity

import (
	"context"

	"github.com/Jkenyut/nvx-go-helper/cryptoutil"
)

// key defines a custom type for context keys to avoid collisions.
type key int

// Context keys constants
const (
	TransactionID key = iota // Unique ID for the transaction
	Action                   // Action name
	ClientID                 // Client identifier
	Payload                  // Request payload
	Result                   // Response result
	RequestIDKey             // Request ID for tracing
)

// NewContext creates a new context with a generated transaction ID and action.
func NewContext(action string) context.Context {
	// Generate a new V7 UUID for the transaction
	trxID := cryptoutil.V7()
	// Inject TransactionID into context
	ctx := context.WithValue(context.Background(), TransactionID, trxID)
	// Inject Action and return the new context
	return context.WithValue(ctx, Action, action)
}

// GetTransactionID retrieves the transaction ID from the context.
func GetTransactionID(ctx context.Context) (string, bool) {
	// Type assertion to ensure safety
	trxID, ok := ctx.Value(TransactionID).(string)
	return trxID, ok
}

// WithAction adds an action string to the context.
func WithAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, Action, action)
}

// GetAction retrieves the action string from the context.
func GetAction(ctx context.Context) (string, bool) {
	action, ok := ctx.Value(Action).(string)
	return action, ok
}

// WithClientID adds a client ID to the context.
func WithClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, ClientID, clientID)
}

// GetClientID retrieves the client ID from the context.
func GetClientID(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(ClientID).(string)
	return clientID, ok
}

// WithPayload adds a payload object to the context.
func WithPayload(ctx context.Context, payload interface{}) context.Context {
	return context.WithValue(ctx, Payload, payload)
}

// GetPayload retrieves the payload object from the context.
// Returns nil if not found.
func GetPayload(ctx context.Context) interface{} {
	return ctx.Value(Payload)
}

// WithResult adds a result object to the context.
func WithResult(ctx context.Context, payload interface{}) context.Context {
	return context.WithValue(ctx, Result, payload)
}

// GetResult retrieves the result object from the context.
// Returns nil if not found.
func GetResult(ctx context.Context) interface{} {
	return ctx.Value(Result)
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
func GetFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// Add transaction_id if present
	if id, ok := GetTransactionID(ctx); ok {
		fields["transaction_id"] = id
	}

	// Add action if present
	if action, ok := GetAction(ctx); ok {
		fields["action"] = action
	}

	// Add request_id if present
	if requestID, ok := GetRequestID(ctx); ok {
		fields["request_id"] = requestID
	}

	// Add client_id if present
	if clientID, ok := GetClientID(ctx); ok {
		fields["client_id"] = clientID
	}

	// Add payload and result (can be nil)
	fields["payload"] = GetPayload(ctx)
	fields["result"] = GetResult(ctx)

	return fields
}
