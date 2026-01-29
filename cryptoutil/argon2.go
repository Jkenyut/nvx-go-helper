package cryptoutil

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// ============================================
// PARAMETERS FOR SMALL SERVERS
// ============================================

// Low Resource (Small Servers: 512MB-1GB RAM, 1-2 CPU cores)
const (
	DefaultTime    = 1         // 1 iteration (fast)
	DefaultMemory  = 32 * 1024 // 32 MB (low memory usage)
	DefaultThreads = 2         // 2 threads (for dual-core)
	DefaultKeyLen  = 32        // 256 bits (standard)
)

// Medium Resource (Medium Servers: 2-4GB RAM, 2-4 CPU cores)
const (
	MediumTime    = 2
	MediumMemory  = 64 * 1024 // 64 MB
	MediumThreads = 4
	MediumKeyLen  = 32
)

// High Resource (Large Servers: 8GB+ RAM, 4+ CPU cores)
const (
	HighTime    = 3
	HighMemory  = 256 * 1024 // 256 MB
	HighThreads = 8
	HighKeyLen  = 32
)

// ============================================
// CORE FUNCTIONS
// ============================================

// DeriveKey generates a cryptographically secure key using Argon2id.
func DeriveKey(password, salt string, time, memory uint32, threads uint8, keyLen uint32) string {
	if keyLen == 0 {
		return ""
	}

	// Decode salt from base64
	saltBytes, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		// If salt is not base64, use it as-is
		saltBytes = []byte(salt)
	}

	// Derive key
	key := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, keyLen)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(key)
}

// DeriveKeyDefault uses LOW RESOURCE parameters (for small servers).
//
// Parameters: time=1, memory=32MB, threads=2, keyLen=32
//
// Recommended for:
//   - Small VPS (512MB-1GB RAM)
//   - Shared hosting
//   - Development environments
//   - Low-traffic applications
func DeriveKeyDefault(password, salt string) string {
	return DeriveKey(password, salt, DefaultTime, DefaultMemory, DefaultThreads, DefaultKeyLen)
}

// DeriveKeyMedium uses MEDIUM RESOURCE parameters.
//
// Parameters: time=2, memory=64MB, threads=4, keyLen=32
//
// Recommended for:
//   - Medium VPS (2-4GB RAM)
//   - Production with moderate traffic
//   - Standard web applications
func DeriveKeyMedium(password, salt string) string {
	return DeriveKey(password, salt, MediumTime, MediumMemory, MediumThreads, MediumKeyLen)
}

// DeriveKeyHigh uses HIGH RESOURCE parameters (most secure).
//
// Parameters: time=3, memory=256MB, threads=8, keyLen=32
//
// Recommended for:
//   - Large servers (8GB+ RAM)
//   - High-security applications
//   - Financial/Healthcare systems
func DeriveKeyHigh(password, salt string) string {
	return DeriveKey(password, salt, HighTime, HighMemory, HighThreads, HighKeyLen)
}

// CompareKey verifies if a password+salt produces the expected key.
func CompareKey(password, salt, expectedKey string, time, memory uint32, threads uint8, keyLen uint32) bool {
	derivedKey := DeriveKey(password, salt, time, memory, threads, keyLen)

	derivedBytes, err1 := base64.StdEncoding.DecodeString(derivedKey)
	expectedBytes, err2 := base64.StdEncoding.DecodeString(expectedKey)

	if err1 != nil || err2 != nil {
		return false
	}

	return subtle.ConstantTimeCompare(derivedBytes, expectedBytes) == 1
}

// CompareKeyDefault uses LOW RESOURCE parameters (for small servers).
func CompareKeyDefault(password, salt, expectedKey string) bool {
	return CompareKey(password, salt, expectedKey, DefaultTime, DefaultMemory, DefaultThreads, DefaultKeyLen)
}

// CompareKeyMedium uses MEDIUM RESOURCE parameters.
func CompareKeyMedium(password, salt, expectedKey string) bool {
	return CompareKey(password, salt, expectedKey, MediumTime, MediumMemory, MediumThreads, MediumKeyLen)
}

// CompareKeyHigh uses HIGH RESOURCE parameters.
func CompareKeyHigh(password, salt, expectedKey string) bool {
	return CompareKey(password, salt, expectedKey, HighTime, HighMemory, HighThreads, HighKeyLen)
}

// ============================================
// HELPER FUNCTIONS
// ============================================

// HashPassword hashes a password using LOW RESOURCE settings (default).
//
// Good for small servers.
func HashPassword(password string) (salt, hash string, err error) {
	salt, err = GenerateKey(32)
	if err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}

	hash = DeriveKeyDefault(password, salt)
	return salt, hash, nil
}

// HashPasswordMedium hashes a password using MEDIUM RESOURCE settings.
//
// Good for production servers.
func HashPasswordMedium(password string) (salt, hash string, err error) {
	salt, err = GenerateKey(32)
	if err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}

	hash = DeriveKeyMedium(password, salt)
	return salt, hash, nil
}

// HashPasswordHigh hashes a password using HIGH RESOURCE settings.
//
// Good for high-security applications.
func HashPasswordHigh(password string) (salt, hash string, err error) {
	salt, err = GenerateKey(32)
	if err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}

	hash = DeriveKeyHigh(password, salt)
	return salt, hash, nil
}

// VerifyPassword verifies using LOW RESOURCE settings (default).
func VerifyPassword(password, salt, hash string) bool {
	return CompareKeyDefault(password, salt, hash)
}

// VerifyPasswordMedium verifies using MEDIUM RESOURCE settings.
func VerifyPasswordMedium(password, salt, hash string) bool {
	return CompareKeyMedium(password, salt, hash)
}

// VerifyPasswordHigh verifies using HIGH RESOURCE settings.
func VerifyPasswordHigh(password, salt, hash string) bool {
	return CompareKeyHigh(password, salt, hash)
}
