package cryptoutil

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ============================================
// PARAMETERS FOR ARGON2ID
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
// BEST PRACTICE: SINGLE COLUMN (PHC FORMAT)
// ============================================

// HashPassword hashes a password using LOW RESOURCE settings (default).
// Returns a standard PHC format string containing the algorithm, version, parameters, salt, and hash.
// Example: $argon2id$v=19$m=32768,t=1,p=2$c2FsdA$aGFzaA
func HashPassword(password string) (string, error) {
	return hashPasswordParams(password, DefaultTime, DefaultMemory, DefaultThreads, DefaultKeyLen)
}

// HashPasswordMedium hashes a password using MEDIUM RESOURCE settings.
// Good for production servers.
func HashPasswordMedium(password string) (string, error) {
	return hashPasswordParams(password, MediumTime, MediumMemory, MediumThreads, MediumKeyLen)
}

// HashPasswordHigh hashes a password using HIGH RESOURCE settings.
// Good for high-security applications.
func HashPasswordHigh(password string) (string, error) {
	return hashPasswordParams(password, HighTime, HighMemory, HighThreads, HighKeyLen)
}

// HashPasswordCustom hashes a password using custom resource settings provided by the developer.
// Use this if you need to fine-tune the Argon2 parameters according to your server specs.
func HashPasswordCustom(password string, time, memory uint32, threads uint8, keyLen uint32) (string, error) {
	return hashPasswordParams(password, time, memory, threads, keyLen)
}

// VerifyPassword compares a plaintext password against a PHC formatted Argon2 hash string.
// It automatically extracts the correct salt, memory, time, and thread settings from the string
// ensuring that you don't need to guess which settings were used to create the hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Format: $argon2id$v=19$m=65536,t=2,p=1$c2FsdA$aGFzaA
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return false, errors.New("invalid hash format (must be PHC standard)")
	}

	if vals[1] != "argon2id" {
		return false, errors.New("incompatible variant, expected argon2id")
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil || version != argon2.Version {
		return false, errors.New("incompatible or unparseable version")
	}

	var memory, time uint32
	var threads uint8
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, errors.New("invalid parameter format")
	}

	saltBytes, err := base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return false, fmt.Errorf("error decoding salt: %w", err)
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return false, fmt.Errorf("error decoding hash: %w", err)
	}

	// Derive key with extracted parameters
	derivedKey := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, uint32(len(hashBytes)))

	// Secure comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(derivedKey, hashBytes) == 1 {
		return true, nil
	}
	return false, nil
}

// hashPasswordParams is the core implementation for generating PHC formatted strings.
func hashPasswordParams(password string, time, memory uint32, threads uint8, keyLen uint32) (string, error) {
	// Generate 16 byte salt (standard recommendation for Argon2)
	saltBytes, err := GenerateKeyRaw(16)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, keyLen)

	// Encode to base64 (Raw encoding without '=' padding is the standard for PHC format)
	b64Salt := base64.RawStdEncoding.EncodeToString(saltBytes)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Combine into a single string
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, time, threads, b64Salt, b64Hash)
	return encoded, nil
}
