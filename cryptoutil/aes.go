// Package cryptoutil provides ultra-fast AES-256-GCM encryption for small servers.
// Zero external dependencies. Must be initialized once at startup with a 32-byte random key.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// AESGCM is the only struct you will ever use.
// Internal fields are unexported → must use NewAESGCM()
type AESGCM struct {
	aead cipher.AEAD // Authenticated Encryption with Associated Data
}

// NewAESGCM MUST be called once at startup.
// Key must be EXACTLY 32 bytes (256 bit) → generate once and store safely!
func NewAESGCM(keys string) (*AESGCM, error) {
	// Convert key string to bytes
	key := []byte(keys)
	// Validate key length (AES-256 requires 32 bytes)
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256-GCM key must be exactly 32 bytes, got %d", len(key))
	}

	// Create new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Wrap block in GCM mode (Galois/Counter Mode)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Return initialized AESGCM struct
	return &AESGCM{aead: gcm}, nil
}

// Encrypt any data → URL-safe base64 string (super fast)
func (c *AESGCM) Encrypt(data any) (string, error) {
	// Serialize data to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	// Generate random nonce (Number used ONCE)
	nonce := make([]byte, 12) // 12 bytes = GCM standard ideal size
	// Read random bytes into nonce
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}

	// Encrypt and authenticate
	// Seal appends result to the first argument (nonce) for efficiency
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	// Return result as URL-safe Base64 string
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt base64 string → original struct/map
func (c *AESGCM) Decrypt(encrypted string, target any) error {
	// Decode Base64 string
	data, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}

	// Verify minimum length (nonce + minimal tag)
	if len(data) < 12 {
		return fmt.Errorf("ciphertext too short")
	}

	// Split nonce and ciphertext
	nonce := data[:12]
	ciphertext := data[12:]

	// Decrypt and verify authentication tag
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed (wrong key or tampered): %w", err)
	}

	// Unmarshal JSON back to target struct
	return json.Unmarshal(plaintext, target)
}
