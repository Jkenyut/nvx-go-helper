// Package aesgcm provides ultra-fast AES-256-GCM encryption for small servers.
// Zero external dependencies. Must be initialized once at startup with a 32-byte random key.
package crypto

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
	aead cipher.AEAD
}

// NewAESGCM MUST be called once at startup.
// Key must be EXACTLY 32 bytes (256 bit) → generate once and store safely!
func NewAESGCM(keys string) (*AESGCM, error) {
	key := []byte(keys)
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256-GCM key must be exactly 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AESGCM{aead: gcm}, nil
}

// Encrypt any data → URL-safe base64 string (super fast)
func (c *AESGCM) Encrypt(data any) (string, error) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	nonce := make([]byte, 12) // 12 bytes = GCM standard
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt base64 string → original struct/map
func (c *AESGCM) Decrypt(encrypted string, target any) error {
	data, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}

	if len(data) < 12 {
		return fmt.Errorf("ciphertext too short")
	}

	nonce := data[:12]
	ciphertext := data[12:]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed (wrong key or tampered): %w", err)
	}

	return json.Unmarshal(plaintext, target)
}
