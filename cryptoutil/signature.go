package cryptoutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"io"
)

// Signature creates a secure HMAC-SHA256 signature for the given data using a secret key.
// It encodes length-prefixes to eliminate delimiter canonicalization collision vulnerabilities.
// It returns a hex-encoded string relative to the input.
func Signature(secret string, data ...string) string {
	// Initialize HMAC with SHA256 and the secret key
	h := hmac.New(sha256.New, []byte(secret))
	lengthBuf := make([]byte, 8)

	// Write each length-prefixed data part to the HMAC hasher
	for _, v := range data {
		binary.BigEndian.PutUint64(lengthBuf, uint64(len(v)))
		_, _ = h.Write(lengthBuf)
		_, _ = io.WriteString(h, v)
	}

	// Calculate the final hash (Sum) and return it as a Hex string
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature compares an expected hex signature against computed signature in constant time.
func VerifySignature(secret string, expectedHex string, data ...string) bool {
	actualHex := Signature(secret, data...)
	return subtle.ConstantTimeCompare([]byte(actualHex), []byte(expectedHex)) == 1
}
