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
	os.Setenv("TEST_DURATION", "10s")
	os.Setenv("TEST_BAD_INT", "abc")
	os.Setenv("TEST_BAD_DURATION", "xyz")

	defer func() {
		os.Unsetenv("TEST_STR")
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_DURATION")
		os.Unsetenv("TEST_BAD_INT")
		os.Unsetenv("TEST_BAD_DURATION")
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

	t.Run("GetBool", func(t *testing.T) {
		assert.True(t, GetBool("TEST_BOOL_TRUE", false))
		assert.False(t, GetBool("TEST_BOOL_FALSE", true))
		assert.True(t, GetBool("MISSING", true))
	})

	t.Run("GetDuration", func(t *testing.T) {
		assert.Equal(t, 10*time.Second, GetDuration("TEST_DURATION", 1*time.Second))
		assert.Equal(t, 1*time.Second, GetDuration("MISSING", 1*time.Second))
		assert.Equal(t, 1*time.Second, GetDuration("TEST_BAD_DURATION", 1*time.Second))
	})
}
