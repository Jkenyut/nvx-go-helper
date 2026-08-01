// Package cryptoutil provides a minimal, opinionated, and battle-tested UUID generator
// tailored for modern Go applications in 2025 and beyond.
//
// Only two UUID versions are exposed because they are the only ones you actually need:
//
//   - V4 – cryptographically random. Use for tokens, secrets, and anything that must be unguessable.
//   - V7 – Unix timestamp + monotonic counter + random. Use for database primary keys,
//     orders, events, logs, and any identifier that benefits from natural ordering.
//
// This package is intentionally tiny and has a single dependency:
// github.com/google/uuid – the de-facto standard UUID library maintained by Google
// and on track to become part of the Go standard library in Go 1.24+.
//
// Example usage:
//
//	userID := cryptoutil.V7UUID()          // primary key (zero allocation)
//	token  := cryptoutil.V4()              // password reset token
//	valid  := cryptoutil.IsValid(input)    // fast validation
//
// All functions are safe for concurrent use and allocate zero heap memory
// when the UUID object form is used (V4UUID / V7UUID).
package cryptoutil

import (
	"github.com/google/uuid"
)

// V4 returns a random UUID v4 as a string.
//
// Use this for:
//   - Session IDs
//   - Password reset tokens
//   - API keys
//   - One-time login links
//   - Any value that must be completely unguessable
//
// Example:
//
//	token := cryptoutil.V4() // "a3f1b9c2-8e4d-4912-b7c5-d3e8f9a1b2c3"
func V4() string {
	// uuid.New() creates a Version 4 UUID (random)
	// It uses crypto/rand for entropy
	return uuid.New().String()
}

// V4UUID returns a random UUID v4 as uuid.UUID (zero heap allocation).
//
// Use this when storing the ID in a struct or passing it through hot code paths.
//
// Example:
//
//	type Session struct {
//	    ID uuid.UUID // ← use cryptoutil.V4UUID()
//	}
func V4UUID() uuid.UUID {
	return uuid.New()
}

// V7 returns a time-ordered UUID v7 as a string.
//
// This is the RECOMMENDED identifier for database primary keys in 2025+.
// Benefits:
//   - Monotonic → perfect for B-Tree indexes (PostgreSQL, MySQL, CockroachDB, etc.)
//   - No index fragmentation
//   - Natural chronological sorting without a separate created_at column
//
// Example:
//
//	userID := cryptoutil.V7() // "0192c84f-17a1-7d2b-9f8a-3c4d5e6f7890"
func V7() string {
	// uuid.NewV7() allocates a new Version 7 UUID
	// Format: unix_ts_ms (48 bits) + ver (4 bits) + rand_a (12 bits) + var (2 bits) + rand_b (62 bits)
	u, _ := uuid.NewV7()
	return u.String()
}

// V7UUID returns a time-ordered UUID v7 as uuid.UUID (zero heap allocation).
//
// This is the single best choice for primary keys and high-throughput systems.
//
// Example:
//
//	type User struct {
//	    ID uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v7()"`
//	}
//
//	user := User{ID: cryptoutil.V7UUID()}
func V7UUID() uuid.UUID {
	u, _ := uuid.NewV7()
	return u
}

// Parse converts a UUID string (with or without hyphens) into a uuid.UUID value.
//
// Returns uuid.Nil if the input is invalid.
//
// Example:
//
//	u := cryptoutil.Parse("0192c84f-17a1-7d2b-9f8a-3c4d5e6f7890")
//	if u == uuid.Nil { ... }
func Parse(s string) uuid.UUID {
	// uuid.Parse handles hex string validation and parsing
	u, _ := uuid.Parse(s)
	// If error occurs, u is uuid.Nil (0000...)
	return u
}

// IsValid reports whether s is a valid UUID string (any version, with or without hyphens).
//
// Zero allocation. Perfect for middleware, validators, or API request checks.
//
// Example:
//
//	if !cryptoutil.IsValid(c.Param("id")) {
//	    c.JSON(400, "invalid uuid")
//	    return
//	}
func IsValid(s string) bool {
	// Try parsing; if no error, the string format is valid
	_, err := uuid.Parse(s)
	return err == nil
}
