package activity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivityContext(t *testing.T) {
	t.Run("NewContext", func(t *testing.T) {
		ctx := NewContext("login")

		id, ok := GetTransactionID(ctx)
		assert.True(t, ok)
		assert.NotEmpty(t, id)

		action, ok := GetAction(ctx)
		assert.True(t, ok)
		assert.Equal(t, "login", action)
	})

	t.Run("WithAction", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithAction(ctx, "update")

		action, ok := GetAction(ctx)
		assert.True(t, ok)
		assert.Equal(t, "update", action)
	})

	t.Run("WithClientID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithClientID(ctx, "client-123")

		clientID, ok := GetClientID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "client-123", clientID)
	})

	t.Run("WithPayload", func(t *testing.T) {
		payload := map[string]string{"foo": "bar"}
		ctx := context.Background()
		ctx = WithPayload(ctx, payload)

		p := GetPayload(ctx)
		assert.Equal(t, payload, p)
	})

	t.Run("WithResult", func(t *testing.T) {
		result := map[string]string{"status": "ok"}
		ctx := context.Background()
		ctx = WithResult(ctx, result)

		r := GetResult(ctx)
		assert.Equal(t, result, r)
	})

	t.Run("WithRequestID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithRequestID(ctx, "req-xyz")

		reqID, ok := GetRequestID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "req-xyz", reqID)
	})

	t.Run("GetTransactionID_Missing", func(t *testing.T) {
		ctx := context.Background()
		_, ok := GetTransactionID(ctx)
		assert.False(t, ok)
	})
}

func TestGetFields(t *testing.T) {
	t.Run("All fields present", func(t *testing.T) {
		ctx := NewContext("create_user")
		ctx = WithClientID(ctx, "client-001")
		ctx = WithRequestID(ctx, "req-001")
		ctx = WithPayload(ctx, "test-payload")
		ctx = WithResult(ctx, "test-result")

		fields := GetFields(ctx)

		assert.NotEmpty(t, fields["transaction_id"])
		assert.Equal(t, "create_user", fields["action"])
		assert.Equal(t, "client-001", fields["client_id"])
		assert.Equal(t, "req-001", fields["request_id"])
		assert.Equal(t, "test-payload", fields["payload"])
		assert.Equal(t, "test-result", fields["result"])
	})

	t.Run("Empty context", func(t *testing.T) {
		ctx := context.Background()
		fields := GetFields(ctx)

		assert.Len(t, fields, 2) // payload and result are always retrieved (nil if missing)
		assert.Nil(t, fields["payload"])
		assert.Nil(t, fields["result"])
		assert.Nil(t, fields["transaction_id"])
		assert.Nil(t, fields["action"])
	})
}
