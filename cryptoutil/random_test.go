package cryptoutil

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomStrings(t *testing.T) {
	t.Run("String (Upper + Numbers)", func(t *testing.T) {
		l := 10
		s1 := String(l)
		s2 := String(l)

		assert.Len(t, s1, l)
		assert.NotEqual(t, s1, s2)
		assert.Regexp(t, "^[A-Z0-9]+$", s1)
	})

	t.Run("StringLower (Lower + Numbers)", func(t *testing.T) {
		l := 12
		s := StringLower(l)
		assert.Len(t, s, l)
		assert.Regexp(t, "^[a-z0-9]+$", s)
	})

	t.Run("StringMixed (Upper + Lower + Numbers)", func(t *testing.T) {
		l := 32
		s := StringMixed(l)
		assert.Len(t, s, l)
		assert.Regexp(t, "^[a-zA-Z0-9]+$", s)
	})

	t.Run("Numbers (Digits only)", func(t *testing.T) {
		l := 6
		s := Numbers(l)
		assert.Len(t, s, l)
		assert.Regexp(t, "^[0-9]+$", s)
	})

	t.Run("Zero Length", func(t *testing.T) {
		assert.Empty(t, String(0))
		assert.Empty(t, String(-5))
	})

	t.Run("High Concurrency / Uniqueness", func(t *testing.T) {
		count := 1000
		set := make(map[string]bool)
		for i := 0; i < count; i++ {
			s := String(16)
			if set[s] {
				t.Fatalf("Collision detected: %s", s)
			}
			set[s] = true
		}
	})
}

// Private function test (whitebox testing)
func TestStringWithCharset(t *testing.T) {
	charset := "ABC"
	s := stringWithCharset(100, charset)
	assert.Len(t, s, 100)

	// Ensure ONLY characters from charset are used
	for _, r := range s {
		assert.True(t, strings.ContainsRune(charset, r), "Character %c not in charset", r)
	}
}

func TestGenerateKey(t *testing.T) {
	t.Run("GenerateKey (Base64)", func(t *testing.T) {
		lenBytes := uint32(32)
		key, err := GenerateKey(lenBytes)
		assert.NoError(t, err)

		decoded, err := base64.StdEncoding.DecodeString(key)
		assert.NoError(t, err)
		assert.Len(t, decoded, int(lenBytes))
	})

	t.Run("GenerateKey Invalid Length", func(t *testing.T) {
		_, err := GenerateKey(0)
		assert.Error(t, err)
	})
}

func TestGenerateKeyHex(t *testing.T) {
	t.Run("GenerateKeyHex", func(t *testing.T) {
		lenBytes := uint32(32)
		key, err := GenerateKeyHex(lenBytes)
		assert.NoError(t, err)

		decoded, err := hex.DecodeString(key)
		assert.NoError(t, err)
		assert.Len(t, decoded, int(lenBytes))
		assert.Len(t, key, int(lenBytes)*2) // Hex string is 2x length
	})

	t.Run("GenerateKeyHex Invalid Length", func(t *testing.T) {
		_, err := GenerateKeyHex(0)
		assert.Error(t, err)
	})
}

func TestGenerateKeyRaw(t *testing.T) {
	t.Run("GenerateKeyRaw", func(t *testing.T) {
		lenBytes := uint32(32)
		key, err := GenerateKeyRaw(lenBytes)
		assert.NoError(t, err)
		assert.Len(t, key, int(lenBytes))
	})

	t.Run("GenerateKeyRaw Invalid Length", func(t *testing.T) {
		_, err := GenerateKeyRaw(0)
		assert.Error(t, err)
	})
}
