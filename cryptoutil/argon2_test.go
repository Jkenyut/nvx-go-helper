package cryptoutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashAndVerifyPassword(t *testing.T) {
	testStr := "test-dummy-str"

	t.Run("Default (Low)", func(t *testing.T) {
		encoded, err := HashPassword(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, encoded)
		// 32 * 1024 = 32768
		assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=32768,t=1,p=2$"))

		match, err := VerifyPassword(testStr, encoded)
		assert.NoError(t, err)
		assert.True(t, match)

		// Negative test
		match, err = VerifyPassword("wrong", encoded)
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("Medium", func(t *testing.T) {
		encoded, err := HashPasswordMedium(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, encoded)

		match, err := VerifyPassword(testStr, encoded)
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("High", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping High profile test in short mode")
		}
		encoded, err := HashPasswordHigh(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, encoded)

		match, err := VerifyPassword(testStr, encoded)
		assert.NoError(t, err)
		assert.True(t, match)
	})
}

func TestVerifyPassword_Errors(t *testing.T) {
	_, err := VerifyPassword("pass", "invalid-hash-string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hash format")

	_, err = VerifyPassword("pass", "$wrongalgo$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible variant")

	_, err = VerifyPassword("pass", "$argon2id$v=99$m=1,t=1,p=1$c2FsdA$aGFzaA")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible or unparseable version")
}

func TestHashPasswordCustom(t *testing.T) {
	testStr := "test-dummy-custom"
	// Ultra-low settings just for fast testing
	time := uint32(1)
	mem := uint32(16 * 1024)
	threads := uint8(1)
	keyLen := uint32(16)

	encoded, err := HashPasswordCustom(testStr, time, mem, threads, keyLen)
	assert.NoError(t, err)
	assert.NotEmpty(t, encoded)
	// format check
	assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=16384,t=1,p=1$"))

	// Verify
	match, err := VerifyPassword(testStr, encoded)
	assert.NoError(t, err)
	assert.True(t, match)
}
