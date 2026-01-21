// Package env provides safe, consistent access to environment variables.
// It allows defining default values to avoid hardcoded fallbacks scattered in the code.
//
// All functions handle missing or empty values by returning the fallback.
package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// GetString returns the env value as string, or fallback if empty.
//
// Example:
//
//	host := env.GetString("DB_HOST", "localhost")
func GetString(key, fallback string) string {
	// Read directly from OS environment
	val := os.Getenv(key)
	// If empty string, return the provided fallback
	if val == "" {
		return fallback
	}
	return val
}

// GetInt returns the env value as int, or fallback if empty or invalid.
//
// Example:
//
//	port := env.GetInt("PORT", 8080)
func GetInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	// Try converting string to int
	i, err := strconv.Atoi(val)
	if err != nil {
		// If conversion fails (e.g. "abc"), safely return fallback
		return fallback
	}
	return i
}

// GetBool returns true if the env value is "true", "1", "yes", or "on" (case insensitive).
// Returns fallback if empty or invalid.
//
// Example:
//
//	debug := env.GetBool("DEBUG", false)
func GetBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}

	// Normalize to lowercase for flexible matching
	switch strings.ToLower(val) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		// If value is not a recognized boolean string, return fallback
		return fallback
	}
}

// GetDuration returns the env value as time.Duration, or fallback if empty or invalid.
//
// Example:
//
//	timeout := env.GetDuration("TIMEOUT", 5*time.Second)
func GetDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	// Parse duration string (e.g., "10s", "1h30m")
	d, err := time.ParseDuration(val)
	if err != nil {
		// If format is invalid, return fallback
		return fallback
	}
	return d
}
