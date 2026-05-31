// Package request provides utility functions for handling HTTP requests.
package request

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jkenyut/nvx-go-helper/validator"
	"github.com/bytedance/sonic"
)

// BindJSON reads the HTTP request body and decodes the JSON data into the provided struct (dest).
// It verifies that the Content-Type is application/json and uses bytedance/sonic for high-performance JSON decoding.
func BindJSON(r *http.Request, dest interface{}) error {
	// Ensure the request has a JSON Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return errors.New("unsupported content type: must be application/json")
	}

	defer func() {
		_ = r.Body.Close()
	}()

	// Use sonic's streaming decoder
	decoder := sonic.ConfigDefault.NewDecoder(r.Body)

	if err := decoder.Decode(dest); err != nil {
		return err
	}

	return nil
}

// BindAndValidate reads the HTTP request body, decodes it, and validates the struct
// using the project's validator package.
func BindAndValidate(r *http.Request, dest interface{}) error {
	if err := BindJSON(r, dest); err != nil {
		return err
	}
	return validator.Struct(dest)
}

// GetQueryString retrieves a query parameter as a string, returning defaultValue if it's empty.
func GetQueryString(r *http.Request, key string, defaultValue string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetQueryInt retrieves a query parameter as an integer, returning defaultValue if it's missing or invalid.
func GetQueryInt(r *http.Request, key string, defaultValue int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetQueryBool retrieves a query parameter as a boolean, returning defaultValue if it's missing or invalid.
func GetQueryBool(r *http.Request, key string, defaultValue bool) bool {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetClientIP extracts the real client IP address from the request, securely parsing
// True-Client-IP, X-Real-IP, and X-Forwarded-For headers. It falls back to RemoteAddr.
func GetClientIP(r *http.Request) string {
	var ip string

	if tcip := r.Header.Get("True-Client-IP"); tcip != "" {
		ip = strings.TrimSpace(tcip)
	} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		ip = strings.TrimSpace(xrip)
	} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip, _, _ = strings.Cut(xff, ",")
		ip = strings.TrimSpace(ip)
	}

	if ip != "" && net.ParseIP(ip) != nil {
		return ip
	}

	ip = r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	ip = strings.Trim(ip, "[]")

	if net.ParseIP(ip) != nil {
		return ip
	}

	return ""
}

// GetBearerToken extracts the Bearer token from the Authorization header.
func GetBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
