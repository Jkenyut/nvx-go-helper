package cryptoutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAESGCM(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes
	aes, err := NewAESGCM(key)
	assert.NoError(t, err)
	assert.NotNil(t, aes)

	t.Run("Encrypt and Decrypt String", func(t *testing.T) {
		original := "secret message"
		encrypted, err := aes.Encrypt(original)
		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, original, encrypted)

		var decrypted string
		err = aes.Decrypt(encrypted, &decrypted)
		assert.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("Encrypt and Decrypt Map", func(t *testing.T) {
		original := map[string]int{"a": 1, "b": 2}
		encrypted, err := aes.Encrypt(original)
		assert.NoError(t, err)

		var decrypted map[string]int
		err = aes.Decrypt(encrypted, &decrypted)
		assert.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("Invalid Key Length", func(t *testing.T) {
		_, err := NewAESGCM("short")
		assert.Error(t, err)
	})

	t.Run("Decrypt Invalid Data", func(t *testing.T) {
		var target string
		err := aes.Decrypt("invalid-base64", &target)
		assert.Error(t, err)
	})
}
