// Package env provides safe, consistent access to environment variables.
// It allows defining default values to avoid hardcoded fallbacks scattered in the code.
//
// All functions handle missing or empty values by returning the fallback.
package env

import (
	"fmt"
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

// GetFloat64 returns the env value as float64, or fallback if empty or invalid.
//
// Example:
//
//	rate := env.GetFloat64("TAX_RATE", 0.11)
func GetFloat64(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
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

// GetStringSlice returns the env value split by the given separator, or fallback if empty.
// Each element is trimmed of whitespace.
//
// Example:
//
//	origins := env.GetStringSlice("CORS_ORIGINS", ",", []string{"http://localhost:3000"})
//	// CORS_ORIGINS="http://a.com, http://b.com" → ["http://a.com", "http://b.com"]
func GetStringSlice(key string, separator string, fallback []string) []string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parts := strings.Split(val, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

// MustGet returns the env value as string, or panics if not set.
// Use ONLY for required configuration at application startup (e.g., in main or init).
//
// Example:
//
//	dbURL := env.MustGet("DATABASE_URL") // panics if not set
func MustGet(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("env: required environment variable %q is not set", key))
	}
	return val
}
