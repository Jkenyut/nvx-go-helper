package activity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivityContext(t *testing.T) {
	ctx := context.Background()

	t.Run("TransactionID", func(t *testing.T) {
		trxID := "trx-123"
		ctx = WithTransactionID(ctx, trxID)
		got, ok := GetTransactionID(ctx)
		assert.True(t, ok)
		assert.Equal(t, trxID, got)
	})

	t.Run("APIKey", func(t *testing.T) {
		key := "api-456"
		ctx = WithAPIKey(ctx, key)
		got, ok := GetAPIKey(ctx)
		assert.True(t, ok)
		assert.Equal(t, key, got)
	})

	t.Run("RequestID", func(t *testing.T) {
		reqID := "req-789"
		ctx = WithRequestID(ctx, reqID)
		got, ok := GetRequestID(ctx)
		assert.True(t, ok)
		assert.Equal(t, reqID, got)
	})

	t.Run("UserID", func(t *testing.T) {
		uid := "user-001"
		ctx = WithUserID(ctx, uid)
		got, ok := GetUserID(ctx)
		assert.True(t, ok)
		assert.Equal(t, uid, got)
	})

	t.Run("UserIP", func(t *testing.T) {
		uip := "127.0.0.1"
		ctx = WithUserIP(ctx, uip)
		got, ok := GetUserIP(ctx)
		assert.True(t, ok)
		assert.Equal(t, uip, got)
	})

	t.Run("WithCustomFields", func(t *testing.T) {
		k := "custom-key"
		val := "custom-value"
		ctx = WithCustomFields(ctx, k, val)

		// Verify with GetCustomField
		got, ok := GetCustomField(ctx, k)
		assert.True(t, ok)
		assert.Equal(t, val, got)
	})

	t.Run("GetCustomField_NotFound", func(t *testing.T) {
		got, ok := GetCustomField(ctx, "nonexistent")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("CustomFields_NoCollision_With_RawString", func(t *testing.T) {
		// Ensure customKey("foo") does NOT collide with raw string "foo"
		rawKey := "collision-test"
		ctx2 := context.WithValue(context.Background(), rawKey, "raw-value")
		ctx2 = WithCustomFields(ctx2, rawKey, "custom-value")

		// Raw string key should still return its own value
		rawVal := ctx2.Value(rawKey)
		assert.Equal(t, "raw-value", rawVal)

		// Custom field should return its own value
		customVal, ok := GetCustomField(ctx2, rawKey)
		assert.True(t, ok)
		assert.Equal(t, "custom-value", customVal)
	})

	t.Run("GetAllFieldsFromContext", func(t *testing.T) {
		fields := GetAllFieldsFromContext(ctx)
		assert.Equal(t, "trx-123", fields["transaction_id"])
		assert.Equal(t, "api-456", fields["api_key"])
		assert.Equal(t, "req-789", fields["request_id"])
		assert.Equal(t, "user-001", fields["user_id"])
		assert.Equal(t, "admin", fields["user_type"])
		assert.Equal(t, "127.0.0.1", fields["user_ip"])
	})

	t.Run("GetAllFieldsFromContext_Empty", func(t *testing.T) {
		emptyCtx := context.Background()
		fields := GetAllFieldsFromContext(emptyCtx)
		assert.Empty(t, fields)
	})

	t.Run("GetTransactionID_NotSet", func(t *testing.T) {
		emptyCtx := context.Background()
		_, ok := GetTransactionID(emptyCtx)
		assert.False(t, ok)
	})
}

func TestGetFieldValueFromContext(t *testing.T) {
	ctx := context.Background()

	// Test with internal key via WithTransactionID
	trxID := "trx-generic-123"
	ctx = WithTransactionID(ctx, trxID)

	got, ok := GetFieldValueFromContext[string](ctx, transactionID)
	assert.True(t, ok)
	assert.Equal(t, trxID, got)

	// Test with string key
	keyStr := "my-string-key"
	valStr := "my-value"
	ctx = context.WithValue(ctx, keyStr, valStr)

	gotStr, okStr := GetFieldValueFromContext[string](ctx, keyStr)
	assert.True(t, okStr)
	assert.Equal(t, valStr, gotStr)

	// Test with explicit mismatched type
	gotInt, okInt := GetFieldValueFromContext[int](ctx, keyStr)
	assert.False(t, okInt)
	assert.Equal(t, 0, gotInt)
}
