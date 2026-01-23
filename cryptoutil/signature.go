package cryptoutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Signature creates a secure HMAC-SHA256 signature for the given data using a secret key.
// It returns a hex-encoded string relative to the input.
func Signature(secret string, data ...string) string {
	// Initialize HMAC with SHA256 and the secret key
	// hmac.New ensures that the key is handled correctly (padding, hashing if too long)
	h := hmac.New(sha256.New, []byte(secret))

	// Write each data part to the HMAC hasher
	// Order matters! Signature(a, b) != Signature(b, a)
	for _, v := range data {
		h.Write([]byte(v))
	}

	// Calculate the final hash (Sum) and return it as a Hex string
	// Hex is safer for transport (URL/JSON) than raw bytes or Base64 (no URL issues)
	return hex.EncodeToString(h.Sum(nil))
}
