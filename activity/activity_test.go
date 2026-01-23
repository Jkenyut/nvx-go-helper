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

	t.Run("MerchantID", func(t *testing.T) {
		merchantID := "merc-456"
		ctx = WithMerchantID(ctx, merchantID)
		got, ok := GetMerchantID(ctx)
		assert.True(t, ok)
		assert.Equal(t, merchantID, got)
	})

	t.Run("RequestID", func(t *testing.T) {
		reqID := "req-789"
		ctx = WithRequestID(ctx, reqID)
		got, ok := GetRequestID(ctx)
		assert.True(t, ok)
		assert.Equal(t, reqID, got)
	})

	t.Run("UserID", func(t *testing.T) {
		userID := "user-001"
		ctx = WithUserID(ctx, userID)
		got, ok := GetUserID(ctx)
		assert.True(t, ok)
		assert.Equal(t, userID, got)
	})

	t.Run("UserType", func(t *testing.T) {
		userType := "admin"
		ctx = WithUserType(ctx, userType)
		got, ok := GetUserType(ctx)
		assert.True(t, ok)
		assert.Equal(t, userType, got)
	})

	t.Run("UserIP", func(t *testing.T) {
		userIP := "127.0.0.1"
		ctx = WithUserIP(ctx, userIP)
		got, ok := GetUserIP(ctx)
		assert.True(t, ok)
		assert.Equal(t, userIP, got)
	})

	t.Run("WithCustomFields", func(t *testing.T) {
		key := "custom-key"
		val := "custom-value"
		ctx = WithCustomFields(ctx, key, val)

		// Verify with GetFieldValueFromContext
		got, ok := GetFieldValueFromContext[string](ctx, key)
		assert.True(t, ok)
		assert.Equal(t, val, got)
	})

	t.Run("GetAllFieldsFromContext", func(t *testing.T) {
		fields := GetAllFieldsFromContext(ctx)
		assert.Equal(t, "trx-123", fields["nvx_transaction_id"])
		assert.Equal(t, "merc-456", fields["nvx_merchant_id"])
		assert.Equal(t, "req-789", fields["nvx_request_id"])
		assert.Equal(t, "user-001", fields["nvx_user_id"])
		assert.Equal(t, "admin", fields["nvx_user_type"])
		assert.Equal(t, "127.0.0.1", fields["nvx_user_ip"])
	})
}

func TestGetFieldValueFromContext(t *testing.T) {
	ctx := context.Background()

	// Test with internal key (TransactionID is of type activity.key)
	trxID := "trx-generic-123"
	ctx = WithTransactionID(ctx, trxID)

	got, ok := GetFieldValueFromContext[string](ctx, TransactionID)
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
