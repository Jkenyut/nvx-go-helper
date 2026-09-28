package cryptoutil

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Common errors returned when validating Argon2 hashes.
var (
	// ErrInvalidHash indicates that the provided hash string is not a valid Argon2 PHC formatted string.
	ErrInvalidHash = errors.New("cryptoutil: invalid Argon2 PHC string format")
	// ErrIncompatibleVariant indicates that the hash does not use the argon2id algorithm variant.
	ErrIncompatibleVariant = errors.New("cryptoutil: incompatible variant, expected argon2id")
	// ErrIncompatibleVersion indicates that the hash version is not supported by this library.
	ErrIncompatibleVersion = errors.New("cryptoutil: incompatible or unparseable argon2 version")
	// ErrInvalidParams indicates that one or more Argon2 parameters are invalid or out of safe bounds.
	ErrInvalidParams = errors.New("cryptoutil: invalid memory, time, or parallelism parameters")
)

// Params defines the configuration parameters used by Argon2id.
type Params struct {
	Memory      uint32 // Memory usage in KiB
	Iterations  uint32 // Number of passes over memory (Time)
	Parallelism uint8  // Number of parallel threads/lanes
	SaltLength  uint32 // Salt length in bytes (standard: 16)
	KeyLength   uint32 // Derived tag length in bytes (standard: 32)
}

// Industry Standard Presets (RFC 9106 & OWASP)
var (
	// DefaultParams (RFC 9106 First Recommended / OWASP Primary)
	// Recommended standard for production web applications and APIs.
	DefaultParams = Params{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  1,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}

	// LowResourceParams (OWASP Constrained)
	// Lower bound for micro-VMs, AWS Lambda, or containers with <= 1GB RAM.
	LowResourceParams = Params{
		Memory:      19 * 1024, // 19 MiB
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	// HighSecurityParams (OWASP High-Security Preset)
	// For dedicated authentication servers targeting higher compute cost (~250-500ms).
	HighSecurityParams = Params{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
)

// HashPassword hashes a plaintext password using the default production parameters (RFC 9106).
func HashPassword(password string) (string, error) {
	return HashPasswordCustom(password, DefaultParams)
}

// HashPasswordLow hashes a plaintext password using low-resource parameters (e.g. for micro-VMs or AWS Lambda).
func HashPasswordLow(password string) (string, error) {
	return HashPasswordCustom(password, LowResourceParams)
}

// HashPasswordHigh hashes a plaintext password using high-security parameters.
func HashPasswordHigh(password string) (string, error) {
	return HashPasswordCustom(password, HighSecurityParams)
}

// HashPasswordCustom hashes a plaintext password using specific Argon2id parameters.
func HashPasswordCustom(password string, p Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("cryptoutil: failed to generate secure salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	// Encode to unpadded base64 (RawStdEncoding per PHC string specification)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism, b64Salt, b64Hash), nil
}

// VerifyPassword securely compares a plaintext password against a PHC-formatted Argon2 hash.
// It extracts the original salt and cost parameters from the string, preventing mismatch.
func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodePHCHash(encodedHash)
	if err != nil {
		return false, err
	}

	derivedKey := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(hash)),
	)

	// Constant-time comparison prevents timing side-channel attacks
	if subtle.ConstantTimeCompare(derivedKey, hash) == 1 {
		return true, nil
	}

	return false, nil
}

// NeedsRehash checks whether the encoded hash was created using different parameters
// than target, signaling that the password should be re-hashed on the next successful authentication.
func NeedsRehash(encodedHash string, target Params) (bool, error) {
	params, _, _, err := decodePHCHash(encodedHash)
	if err != nil {
		return false, err
	}

	if params.Memory != target.Memory ||
		params.Iterations != target.Iterations ||
		params.Parallelism != target.Parallelism ||
		params.KeyLength != target.KeyLength {
		return true, nil
	}

	return false, nil
}

// decodePHCHash parses a PHC formatted string into its components.
func decodePHCHash(encodedHash string) (p Params, salt, hash []byte, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return p, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return p, nil, nil, ErrIncompatibleVariant
	}

	var version int
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil || version != argon2.Version {
		return p, nil, nil, ErrIncompatibleVersion
	}

	// Parse parameters: m=65536,t=1,p=4
	paramPairs := strings.Split(parts[3], ",")
	if len(paramPairs) != 3 {
		return p, nil, nil, ErrInvalidParams
	}

	for _, pair := range paramPairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return p, nil, nil, ErrInvalidParams
		}

		val, parseErr := strconv.ParseUint(kv[1], 10, 32)
		if parseErr != nil {
			return p, nil, nil, ErrInvalidParams
		}

		switch kv[0] {
		case "m":
			p.Memory = uint32(val)
		case "t":
			p.Iterations = uint32(val)
		case "p":
			p.Parallelism = uint8(val)
		default:
			return p, nil, nil, ErrInvalidParams
		}
	}

	// Sanity bounds check to protect against DoS / memory exhaustion
	if p.Memory < 8 || p.Memory > 1024*1024 || p.Iterations < 1 || p.Iterations > 100 || p.Parallelism < 1 {
		return p, nil, nil, ErrInvalidParams
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return p, nil, nil, fmt.Errorf("%w: invalid salt decoding", ErrInvalidHash)
	}
	p.SaltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return p, nil, nil, fmt.Errorf("%w: invalid hash decoding", ErrInvalidHash)
	}
	p.KeyLength = uint32(len(hash))

	return p, salt, hash, nil
}
