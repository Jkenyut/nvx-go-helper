package response

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewMeta_UsesRequestIDFromContext(t *testing.T) {
	expectedID := uuid.New().String()
	ctx := context.WithValue(context.Background(), requestIDKey, expectedID)

	meta := NewMeta(ctx, true, "test", 200)

	assert.Equal(t, expectedID, meta.RequestID)
	assert.True(t, meta.Success)
	assert.Equal(t, "test", meta.Message)
	assert.Equal(t, 200, meta.StatusCode)
}

func TestNewMeta_GeneratesNewIDWhenMissing(t *testing.T) {
	ctx := context.Background()
	meta1 := NewMeta(ctx, true, "test", 200)
	meta2 := NewMeta(ctx, true, "test", 200)

	assert.NotEmpty(t, meta1.RequestID)
	assert.NotEmpty(t, meta2.RequestID)
	assert.NotEqual(t, meta1.RequestID, meta2.RequestID) // statistically unique
	assert.Len(t, meta1.RequestID, 36)                   // UUID format
}

func TestSuccessResponses(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "fixed-id-123")

	tests := []struct {
		name    string
		fn      func(context.Context) Response
		status  int
		message string
		hasData bool
	}{
		{"OK", func(c context.Context) Response { return OK(c, "all good", "data") }, 200, "all good", true},
		{"Created", func(c context.Context) Response { return Created(c, "created", nil) }, 201, "created", false},
		{"Accepted", func(c context.Context) Response { return Accepted(c, "accepted", nil) }, 202, "accepted", false},
		{"NoContent", func(c context.Context) Response { return NoContent(c) }, 204, "no content", false},
		{"Success format", func(c context.Context) Response { return Success(c, "payload") }, 200, "success", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.fn(ctx)
			assert.Equal(t, "fixed-id-123", resp.Meta.RequestID)
			assert.Equal(t, tt.status, resp.Meta.StatusCode)
			assert.Equal(t, tt.message, resp.Meta.Message)
			assert.Equal(t, tt.hasData, resp.Data != nil)
		})
	}
}

func TestErrorResponses(t *testing.T) {
	ctx := context.Background() // no request_id → auto generated

	errorFuncs := map[string]func(context.Context) Response{
		"BadRequest":          func(c context.Context) Response { return BadRequest(c, "invalid input") },
		"Unauthorized":        func(c context.Context) Response { return Unauthorized(c, "auth required") },
		"Forbidden":           func(c context.Context) Response { return Forbidden(c, "access denied") },
		"NotFound":            func(c context.Context) Response { return NotFound(c, "not found") },
		"Conflict":            func(c context.Context) Response { return Conflict(c, "already exists") },
		"UnprocessableEntity": func(c context.Context) Response { return UnprocessableEntity(c, "validation failed") },
		"TooManyRequests":     func(c context.Context) Response { return TooManyRequests(c, "rate limited") },
		"InternalError":       func(c context.Context) Response { return InternalError(c) },
	}

	for name, fn := range errorFuncs {
		t.Run(name, func(t *testing.T) {
			resp := fn(ctx)
			assert.False(t, resp.Meta.Success)
			assert.NotEmpty(t, resp.Meta.RequestID)
			assert.Nil(t, resp.Data)
		})
	}
}

func TestResponse_JSONSerialization(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "test-12345")
	resp := Created(ctx, "user registered", map[string]string{"name": "Budi"})

	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	assert.Contains(t, jsonStr, `"success":true`)
	assert.Contains(t, jsonStr, `"message":"user registered"`)
	assert.Contains(t, jsonStr, `"status_code":201`)
	assert.Contains(t, jsonStr, `"request_id":"test-12345"`)
	assert.Contains(t, jsonStr, `"data":{"name":"Budi"}`)
	assert.NotContains(t, jsonStr, "null") // omitempty works
}

func TestResponse_WithMessage(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "test-12345")
	resp := WithMessage(ctx, "user registered", 200)

	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	assert.Contains(t, jsonStr, `"success":true`)
	assert.Contains(t, jsonStr, `"message":"user registered"`)
}
