package cryptoutil

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveKey(t *testing.T) {
	// Standard test params
	time := uint32(1)
	mem := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	t.Run("Generate 32-byte key (encoded as Base64)", func(t *testing.T) {
		key := DeriveKey("test-dummy-str", "salt", time, mem, threads, keyLen)

		// Decode back to check actual length
		decoded, err := base64.StdEncoding.DecodeString(key)
		assert.NoError(t, err)
		assert.Equal(t, int(keyLen), len(decoded))
	})

	t.Run("Consistency check", func(t *testing.T) {
		key1 := DeriveKey("test-dummy-str", "salt", time, mem, threads, keyLen)
		key2 := DeriveKey("test-dummy-str", "salt", time, mem, threads, keyLen)
		assert.Equal(t, key1, key2, "Same input should produce same output")
	})

	t.Run("Different inputs produce different keys", func(t *testing.T) {
		key1 := DeriveKey("test-dummy-str", "salt1", time, mem, threads, keyLen)
		key2 := DeriveKey("test-dummy-str", "salt2", time, mem, threads, keyLen)
		assert.NotEqual(t, key1, key2, "Different salts should produce different keys")
	})

	t.Run("Zero length", func(t *testing.T) {
		key := DeriveKey("test-dummy-str", "salt", time, mem, threads, 0)
		assert.Equal(t, "", key)
	})

	t.Run("Backward compatibility with non-base64 salt", func(t *testing.T) {
		key := DeriveKey("test-dummy-str", "raw-salt", time, mem, threads, keyLen)
		assert.NotEmpty(t, key)

		// Verify consistency
		key2 := DeriveKey("test-dummy-str", "raw-salt", time, mem, threads, keyLen)
		assert.Equal(t, key, key2)
	})
}

func TestDeriveKeyProfiles(t *testing.T) {
	testStr := "test-dummy-str"
	salt := "cmFuZG9tU2FsdA==" // base64

	t.Run("Default (Low)", func(t *testing.T) {
		key := DeriveKeyDefault(testStr, salt)
		assert.NotEmpty(t, key)

		match := CompareKeyDefault(testStr, salt, key)
		assert.True(t, match)
	})

	t.Run("Medium", func(t *testing.T) {
		key := DeriveKeyMedium(testStr, salt)
		assert.NotEmpty(t, key)

		match := CompareKeyMedium(testStr, salt, key)
		assert.True(t, match)
	})

	t.Run("High", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping High profile test in short mode")
		}
		key := DeriveKeyHigh(testStr, salt)
		assert.NotEmpty(t, key)

		match := CompareKeyHigh(testStr, salt, key)
		assert.True(t, match)
	})
}

func TestPasswordHelpers(t *testing.T) {
	testStr := "test-dummy-str"

	t.Run("HashPassword (Default)", func(t *testing.T) {
		salt, hash, err := HashPassword(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, salt)
		assert.NotEmpty(t, hash)

		match := VerifyPassword(testStr, salt, hash)
		assert.True(t, match)

		// Negative test
		match = VerifyPassword("wrong", salt, hash)
		assert.False(t, match)
	})

	t.Run("HashPasswordMedium", func(t *testing.T) {
		salt, hash, err := HashPasswordMedium(testStr)
		assert.NoError(t, err)

		match := VerifyPasswordMedium(testStr, salt, hash)
		assert.True(t, match)
	})

	t.Run("HashPasswordHigh", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping High profile test in short mode")
		}
		salt, hash, err := HashPasswordHigh(testStr)
		assert.NoError(t, err)

		match := VerifyPasswordHigh(testStr, salt, hash)
		assert.True(t, match)
	})
}

func TestCompareKey(t *testing.T) {
	// Standard test params
	time := uint32(1)
	mem := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	t.Run("Valid match", func(t *testing.T) {
		testStr := "test-dummy-str"
		salt := "cmFuZG9tU2FsdA==" // "randomSalt" in base64

		// Generate original key
		originalKey := DeriveKey(testStr, salt, time, mem, threads, keyLen)

		// Verify
		match := CompareKey(testStr, salt, originalKey, time, mem, threads, keyLen)
		assert.True(t, match, "Key should match with same parameters")
	})

	t.Run("Invalid password", func(t *testing.T) {
		testStr := "test-dummy-str"
		salt := "cmFuZG9tU2FsdA=="
		originalKey := DeriveKey(testStr, salt, time, mem, threads, keyLen)

		match := CompareKey("wrongTestStr", salt, originalKey, time, mem, threads, keyLen)
		assert.False(t, match, "Should not match with wrong password")
	})

	t.Run("Invalid salt", func(t *testing.T) {
		testStr := "test-dummy-str"
		salt := "cmFuZG9tU2FsdA=="
		originalKey := DeriveKey(testStr, salt, time, mem, threads, keyLen)

		match := CompareKey(testStr, "wrongSalt", originalKey, time, mem, threads, keyLen)
		assert.False(t, match, "Should not match with wrong salt")
	})

	t.Run("Invalid key format (not base64)", func(t *testing.T) {
		match := CompareKey("pass", "salt", "invalid-base64-!@#$", time, mem, threads, keyLen)
		assert.False(t, match)
	})
}
