package env

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnv(t *testing.T) {
	// Setup
	os.Setenv("TEST_STR", "hello")
	os.Setenv("TEST_INT", "123")
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "0")
	os.Setenv("TEST_BOOL_YES", "yes")
	os.Setenv("TEST_BOOL_ON", "on")
	os.Setenv("TEST_BOOL_NO", "no")
	os.Setenv("TEST_BOOL_OFF", "off")
	os.Setenv("TEST_BOOL_INVALID", "maybe")
	os.Setenv("TEST_DURATION", "10s")
	os.Setenv("TEST_BAD_INT", "abc")
	os.Setenv("TEST_BAD_DURATION", "xyz")
	os.Setenv("TEST_FLOAT", "3.14")
	os.Setenv("TEST_BAD_FLOAT", "not-a-float")
	os.Setenv("TEST_SLICE", "a, b, c")
	os.Setenv("TEST_SLICE_EMPTY", "  ,  ,  ")
	os.Setenv("TEST_MUST", "required_value")

	defer func() {
		os.Unsetenv("TEST_STR")
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_YES")
		os.Unsetenv("TEST_BOOL_ON")
		os.Unsetenv("TEST_BOOL_NO")
		os.Unsetenv("TEST_BOOL_OFF")
		os.Unsetenv("TEST_BOOL_INVALID")
		os.Unsetenv("TEST_DURATION")
		os.Unsetenv("TEST_BAD_INT")
		os.Unsetenv("TEST_BAD_DURATION")
		os.Unsetenv("TEST_FLOAT")
		os.Unsetenv("TEST_BAD_FLOAT")
		os.Unsetenv("TEST_SLICE")
		os.Unsetenv("TEST_SLICE_EMPTY")
		os.Unsetenv("TEST_MUST")
	}()

	t.Run("GetString", func(t *testing.T) {
		assert.Equal(t, "hello", GetString("TEST_STR", "default"))
		assert.Equal(t, "default", GetString("MISSING", "default"))
	})

	t.Run("GetInt", func(t *testing.T) {
		assert.Equal(t, 123, GetInt("TEST_INT", 1))
		assert.Equal(t, 1, GetInt("MISSING", 1))
		assert.Equal(t, 1, GetInt("TEST_BAD_INT", 1))
	})

	t.Run("GetFloat64", func(t *testing.T) {
		assert.InDelta(t, 3.14, GetFloat64("TEST_FLOAT", 0.0), 0.001)
		assert.InDelta(t, 0.0, GetFloat64("MISSING", 0.0), 0.001)
		assert.InDelta(t, 1.5, GetFloat64("TEST_BAD_FLOAT", 1.5), 0.001)
	})

	t.Run("GetBool", func(t *testing.T) {
		assert.True(t, GetBool("TEST_BOOL_TRUE", false))
		assert.False(t, GetBool("TEST_BOOL_FALSE", true))
		assert.True(t, GetBool("TEST_BOOL_YES", false))
		assert.True(t, GetBool("TEST_BOOL_ON", false))
		assert.False(t, GetBool("TEST_BOOL_NO", true))
		assert.False(t, GetBool("TEST_BOOL_OFF", true))
		assert.True(t, GetBool("TEST_BOOL_INVALID", true)) // fallback
		assert.True(t, GetBool("MISSING", true))
	})

	t.Run("GetDuration", func(t *testing.T) {
		assert.Equal(t, 10*time.Second, GetDuration("TEST_DURATION", 1*time.Second))
		assert.Equal(t, 1*time.Second, GetDuration("MISSING", 1*time.Second))
		assert.Equal(t, 1*time.Second, GetDuration("TEST_BAD_DURATION", 1*time.Second))
	})

	t.Run("GetStringSlice", func(t *testing.T) {
		result := GetStringSlice("TEST_SLICE", ",", nil)
		assert.Equal(t, []string{"a", "b", "c"}, result)

		// Missing key
		result = GetStringSlice("MISSING", ",", []string{"default"})
		assert.Equal(t, []string{"default"}, result)

		// All-whitespace entries fallback
		result = GetStringSlice("TEST_SLICE_EMPTY", ",", []string{"fb"})
		assert.Equal(t, []string{"fb"}, result)
	})

	t.Run("MustGet_Exists", func(t *testing.T) {
		assert.Equal(t, "required_value", MustGet("TEST_MUST"))
	})

	t.Run("MustGet_Panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustGet("MISSING_REQUIRED_VAR")
		})
	})
}
