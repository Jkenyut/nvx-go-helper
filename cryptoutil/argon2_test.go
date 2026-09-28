package cryptoutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashAndVerifyPassword(t *testing.T) {
	testStr := "test-dummy-str"

	t.Run("Default (RFC 9106)", func(t *testing.T) {
		encoded, err := HashPassword(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, encoded)
		// 64 * 1024 = 65536, t=1, p=4
		assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=1,p=4$"))

		match, err := VerifyPassword(testStr, encoded)
		assert.NoError(t, err)
		assert.True(t, match)

		// Negative test
		match, err = VerifyPassword("wrong", encoded)
		assert.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("Low (OWASP Constrained)", func(t *testing.T) {
		encoded, err := HashPasswordLow(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, encoded)
		// 19 * 1024 = 19456, t=2, p=1
		assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$"))

		match, err := VerifyPassword(testStr, encoded)
		assert.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("High (High Security)", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping High profile test in short mode")
		}
		encoded, err := HashPasswordHigh(testStr)
		assert.NoError(t, err)
		assert.NotEmpty(t, encoded)
		// 64 * 1024 = 65536, t=3, p=4
		assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=4$"))

		match, err := VerifyPassword(testStr, encoded)
		assert.NoError(t, err)
		assert.True(t, match)
	})
}

func TestVerifyPassword_Errors(t *testing.T) {
	_, err := VerifyPassword("pass", "invalid-hash-string")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidHash)

	_, err = VerifyPassword("pass", "$wrongalgo$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrIncompatibleVariant)

	_, err = VerifyPassword("pass", "$argon2id$v=99$m=1,t=1,p=1$c2FsdA$aGFzaA")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrIncompatibleVersion)

	// Out of bounds / invalid params
	_, err = VerifyPassword("pass", "$argon2id$v=19$m=2097152,t=1,p=1$c2FsdA$aGFzaA")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidParams)
}

func TestHashPasswordCustom(t *testing.T) {
	testStr := "test-dummy-custom"
	customParams := Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	encoded, err := HashPasswordCustom(testStr, customParams)
	assert.NoError(t, err)
	assert.NotEmpty(t, encoded)
	assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=16384,t=1,p=1$"))

	// Verify
	match, err := VerifyPassword(testStr, encoded)
	assert.NoError(t, err)
	assert.True(t, match)
}

func TestNeedsRehash(t *testing.T) {
	testStr := "secure-password"

	// Hashed with LowResourceParams
	lowHash, err := HashPasswordLow(testStr)
	assert.NoError(t, err)

	// Check against DefaultParams: should need rehash
	needs, err := NeedsRehash(lowHash, DefaultParams)
	assert.NoError(t, err)
	assert.True(t, needs)

	// Check against LowResourceParams: should NOT need rehash
	needs, err = NeedsRehash(lowHash, LowResourceParams)
	assert.NoError(t, err)
	assert.False(t, needs)

	// Invalid hash check
	_, err = NeedsRehash("invalid-hash", DefaultParams)
	assert.Error(t, err)
}
