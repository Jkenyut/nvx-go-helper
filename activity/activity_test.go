package activity

import (
	"context"
	"log/slog"
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

	t.Run("UserIPOrigin", func(t *testing.T) {
		uipOrigin := "10.0.0.1"
		ctx = WithUserIPOrigin(ctx, uipOrigin)
		got, ok := GetUserIPOrigin(ctx)
		assert.True(t, ok)
		assert.Equal(t, uipOrigin, got)
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

	t.Run("Metadata", func(t *testing.T) {
		emptyCtx := context.Background()
		// Test empty metadata no-op
		noOpCtx := WithMetadata(emptyCtx, nil)
		_, hasMeta := GetMetadata(noOpCtx)
		assert.False(t, hasMeta)

		meta := map[string]any{"source": "web", "version": "v1.0"}
		ctxMeta := WithMetadata(emptyCtx, meta)
		gotMeta, ok := GetMetadata(ctxMeta)
		assert.True(t, ok)
		assert.Equal(t, "web", gotMeta["source"])
		assert.Equal(t, "v1.0", gotMeta["version"])

		// Test merge
		ctxMerged := WithMetadata(ctxMeta, map[string]any{"env": "prod"})
		gotMerged, okMerged := GetMetadata(ctxMerged)
		assert.True(t, okMerged)
		assert.Equal(t, "web", gotMerged["source"])
		assert.Equal(t, "prod", gotMerged["env"])
	})

	t.Run("GetAllFieldsFromContext", func(t *testing.T) {
		fields := GetAllFieldsFromContext(ctx)
		assert.Equal(t, "trx-123", fields["transaction_id"])
		assert.Equal(t, "req-789", fields["request_id"])
		assert.Equal(t, "user-001", fields["user_id"])
		assert.Equal(t, "127.0.0.1", fields["user_ip"])
		assert.Equal(t, "10.0.0.1", fields["user_ip_origin"])
	})

	t.Run("GetAllFieldsFromContext_WithMetadata", func(t *testing.T) {
		ctxWithMeta := WithMetadata(ctx, map[string]any{"module": "payment"})
		fields := GetAllFieldsFromContext(ctxWithMeta)
		assert.Equal(t, "payment", fields["module"])
		assert.Equal(t, "trx-123", fields["transaction_id"])
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

	t.Run("UnsetFields", func(t *testing.T) {
		emptyCtx := context.Background()
		_, okReq := GetRequestID(emptyCtx)
		assert.False(t, okReq)
		_, okUID := GetUserID(emptyCtx)
		assert.False(t, okUID)
		_, okIP := GetUserIP(emptyCtx)
		assert.False(t, okIP)
		_, okIPOrigin := GetUserIPOrigin(emptyCtx)
		assert.False(t, okIPOrigin)
	})
}

func TestActivityStruct(t *testing.T) {
	orig := Activity{
		TransactionID: "trx-batch-1",
		RequestID:     "req-batch-2",
		UserID:        "user-batch-3",
		UserIP:        "192.168.1.1",
		UserIPOrigin:  "10.10.10.10",
	}

	ctx := WithActivity(context.Background(), orig)

	// Verify standard getters work with values injected via WithActivity
	reqID, okReq := GetRequestID(ctx)
	assert.True(t, okReq)
	assert.Equal(t, "req-batch-2", reqID)

	uid, okUID := GetUserID(ctx)
	assert.True(t, okUID)
	assert.Equal(t, "user-batch-3", uid)

	// Verify FromContext
	extracted := FromContext(ctx)
	assert.Equal(t, orig, extracted)
}

func TestToSlogAttrs(t *testing.T) {
	t.Run("Populated", func(t *testing.T) {
		act := Activity{
			TransactionID: "trx-slog",
			RequestID:     "req-slog",
			UserID:        "user-slog",
			UserIP:        "127.0.0.1",
			UserIPOrigin:  "10.0.0.1",
		}
		ctx := WithActivity(context.Background(), act)
		attrs := ToSlogAttrs(ctx)

		assert.Len(t, attrs, 5)

		attrMap := make(map[string]string, len(attrs))
		for _, a := range attrs {
			assert.Equal(t, slog.KindString, a.Value.Kind())
			attrMap[a.Key] = a.Value.String()
		}

		assert.Equal(t, "trx-slog", attrMap["transaction_id"])
		assert.Equal(t, "req-slog", attrMap["request_id"])
		assert.Equal(t, "user-slog", attrMap["user_id"])
		assert.Equal(t, "127.0.0.1", attrMap["user_ip"])
		assert.Equal(t, "10.0.0.1", attrMap["user_ip_origin"])
	})

	t.Run("Empty", func(t *testing.T) {
		attrs := ToSlogAttrs(context.Background())
		assert.Empty(t, attrs)
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
