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

	// Validate key length strictly for AES-256
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256-GCM key must be exactly 32 bytes, got %d", len(key))
	}

	// Create a new AES cipher block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Wrap the AES block in Galois Counter Mode (GCM)
	// GCM provides both confidentiality (encryption) and integrity (authentication)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Return initialized AESGCM struct
	return &AESGCM{aead: gcm}, nil
}

// Encrypt any data → URL-safe base64 string (super fast)
func (c *AESGCM) Encrypt(data any) (string, error) {
	// Marshal the input data (struct/map/slice) to JSON bytes
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	// Create a random nonce (Number Used Once)
	// GCM standard requires a 12-byte nonce
	nonce := make([]byte, 12)
	// Read cryptographically secure random bytes into the nonce
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce generation failed: %w", err)
	}

	// Encrypt and authenticate
	// Seal appends result to the first argument (nonce) for efficiency
	// We prepend the nonce to the ciphertext so we can retrieve it during decryption
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)

	// Encode the combined [nonce + ciphertext] to URL-safe Base64 string.
	// This makes it safe to use in URLs (e.g., query params) or JSON
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt base64 string → original struct/map
func (c *AESGCM) Decrypt(encrypted string, target any) error {
	// Decode from URL-safe Base64
	data, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}

	// Validate min length (must at least contain nonce)
	if len(data) < 12 {
		return fmt.Errorf("ciphertext too short")
	}

	// Extract the nonce (first 12 bytes)
	nonce := data[:12]
	// Extract the actual ciphertext (remaining bytes)
	ciphertext := data[12:]

	// Decrypt and verify authentication tag
	// Open(dst, nonce, ciphertext, additionalData)
	// This also verifies the authentication tag (integrity check) automatically
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed (wrong key or tampered): %w", err)
	}

	// Unmarshal the decrypted JSON bytes back into the target struct
	return json.Unmarshal(plaintext, target)
}
