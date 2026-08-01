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

func TestGenerateAESKeyAndNewFromHex(t *testing.T) {
	// 1. Generate key
	hexKey, err := GenerateAESKey()
	assert.NoError(t, err)
	
	// Hex string of 32 bytes should be 64 characters long
	assert.Len(t, hexKey, 64)

	// 2. Initialize from hex
	aesGCM, err := NewAESGCMFromHex(hexKey)
	assert.NoError(t, err)
	assert.NotNil(t, aesGCM)

	// 3. Test Encrypt/Decrypt
	original := "secret message with generated key"
	encrypted, err := aesGCM.Encrypt(original)
	assert.NoError(t, err)

	var decrypted string
	err = aesGCM.Decrypt(encrypted, &decrypted)
	assert.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestNewAESGCMFromHex_Invalid(t *testing.T) {
	// Invalid hex string
	_, err := NewAESGCMFromHex("not a hex string!")
	assert.Error(t, err)

	// Valid hex but wrong length (16 bytes = 32 hex chars)
	_, err = NewAESGCMFromHex("0123456789abcdef0123456789abcdef")
	assert.Error(t, err)
}
