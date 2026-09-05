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
	"github.com/google/uuid"
)

// BindJSON reads the HTTP request body and decodes the JSON data into the provided struct (dest).
// It verifies that the Content-Type is application/json and uses bytedance/sonic for high-performance JSON decoding.
func BindJSON(r *http.Request, dest any) error {
	// Ensure the request has a JSON Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return errors.New("unsupported content type: must be application/json")
	}

	defer func() {
		_ = r.Body.Close()
	}()

	decoder := sonic.ConfigDefault.NewDecoder(r.Body)
	if err := decoder.Decode(dest); err != nil {
		return err
	}

	return nil
}

// BindAndValidate reads the HTTP request body, decodes it, and validates the struct
// using the project's validator package.
func BindAndValidate(r *http.Request, dest any) error {
	if err := BindJSON(r, dest); err != nil {
		return err
	}
	return validator.Struct(dest)
}

func getQueryValue(r *http.Request, key string) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Query().Get(key)
}

// GetQueryString retrieves a query parameter as a string, returning defaultValue if it's empty.
// It reads directly from URL query parameters without consuming or parsing request body.
func GetQueryString(r *http.Request, key string, defaultValue string) string {
	val := getQueryValue(r, key)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetQueryInt retrieves a query parameter as an integer, returning defaultValue if it's missing or invalid.
func GetQueryInt(r *http.Request, key string, defaultValue int) int {
	val := getQueryValue(r, key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetQueryInt64 retrieves a query parameter as a 64-bit integer, returning defaultValue if it's missing or invalid.
func GetQueryInt64(r *http.Request, key string, defaultValue int64) int64 {
	val := getQueryValue(r, key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetQueryBool retrieves a query parameter as a boolean, returning defaultValue if it's missing or invalid.
func GetQueryBool(r *http.Request, key string, defaultValue bool) bool {
	val := getQueryValue(r, key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getPathValue(r *http.Request, key string) string {
	if r == nil {
		return ""
	}
	cleanKey := strings.Trim(strings.TrimSpace(key), "{}")
	return r.PathValue(cleanKey)
}

// GetPathValue retrieves a path parameter string directly from the request URL path pattern.
func GetPathValue(r *http.Request, key string) string {
	return getPathValue(r, key)
}

// GetPathString retrieves a path parameter as a string, returning defaultValue if it is empty.
// If defaultValue is omitted, it returns an empty string when the path parameter is missing.
func GetPathString(r *http.Request, key string, defaultValue ...string) string {
	val := getPathValue(r, key)
	if val == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return val
}

// GetPathInt retrieves a path parameter as an integer, returning defaultValue if it is missing or invalid.
// If defaultValue is omitted, it returns 0 when missing or invalid.
func GetPathInt(r *http.Request, key string, defaultValue ...int) int {
	val := getPathValue(r, key)
	if val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return parsed
}

// GetPathInt64 retrieves a path parameter as a 64-bit integer, returning defaultValue if it is missing or invalid.
// If defaultValue is omitted, it returns 0 when missing or invalid.
func GetPathInt64(r *http.Request, key string, defaultValue ...int64) int64 {
	val := getPathValue(r, key)
	if val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return parsed
}

// GetPathBool retrieves a path parameter as a boolean, returning defaultValue if it is missing or invalid.
// If defaultValue is omitted, it returns false when missing or invalid.
func GetPathBool(r *http.Request, key string, defaultValue ...bool) bool {
	val := getPathValue(r, key)
	if val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return parsed
}

// GetPathUUID retrieves and parses a path parameter as a UUID.
func GetPathUUID(r *http.Request, key string) (uuid.UUID, error) {
	val := getPathValue(r, key)
	if val == "" {
		return uuid.Nil, errors.New("empty path parameter: " + key)
	}
	return uuid.Parse(val)
}

// GetQueryStringSlice retrieves a query parameter and splits it by comma into a slice of strings.
// It automatically trims whitespace and removes empty segments.
// Useful for supporting multiple filter codes (e.g. ?code=APP1,APP2).
func GetQueryStringSlice(r *http.Request, key string) []string {
	val := getQueryValue(r, key)
	if val == "" {
		return nil
	}

	var result []string
	parts := strings.Split(val, ",")
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

// GetQueryIntSlice retrieves a query parameter and splits it by comma into a slice of integers.
// Invalid integers are safely ignored.
func GetQueryIntSlice(r *http.Request, key string) []int {
	strs := GetQueryStringSlice(r, key)
	if len(strs) == 0 {
		return nil
	}

	result := make([]int, 0, len(strs))
	for _, s := range strs {
		if num, err := strconv.Atoi(s); err == nil {
			result = append(result, num)
		}
	}
	return result
}

// GetQueryMapSlice retrieves grouped query parameters in a custom delimiter format.
// Example format: ?group=legal:approved,draft&group=operation:draft
// It automatically handles multiple keys and cleans up whitespace.
func GetQueryMapSlice(r *http.Request, key string, delimiter string) map[string][]string {
	result := make(map[string][]string)

	// A query parameter can be provided multiple times (e.g. ?group=A&group=B)
	values := r.URL.Query()[key]
	if len(values) == 0 {
		return result
	}

	for _, val := range values {
		// Split into GroupKey and Values, e.g., "legal:approved,draft" -> "legal" and "approved,draft"
		parts := strings.SplitN(val, delimiter, 2)
		if len(parts) != 2 {
			continue // Ignore if it doesn't match the delimiter format
		}

		groupKey := strings.TrimSpace(parts[0])
		if groupKey == "" {
			continue
		}

		// Split the values by comma
		rawValues := strings.Split(parts[1], ",")
		for _, rv := range rawValues {
			clean := strings.TrimSpace(rv)
			if clean != "" {
				result[groupKey] = append(result[groupKey], clean)
			}
		}
	}

	return result
}

// GetQueryMapIntSlice retrieves grouped query parameters and converts the values into integers.
// Invalid integers are safely ignored.
// Example format: ?group=status:1,2&group=category:3,4
func GetQueryMapIntSlice(r *http.Request, key string, delimiter string) map[string][]int {
	rawMap := GetQueryMapSlice(r, key, delimiter)
	if len(rawMap) == 0 {
		return make(map[string][]int)
	}

	result := make(map[string][]int, len(rawMap))
	for groupKey, items := range rawMap {
		var nums []int
		for _, item := range items {
			if num, err := strconv.Atoi(item); err == nil {
				nums = append(nums, num)
			}
		}
		if len(nums) > 0 {
			result[groupKey] = nums
		}
	}

	return result
}

// GetQueryMapBoolSlice retrieves grouped query parameters and converts the values into booleans.
// Invalid booleans are safely ignored.
// Example format: ?group=featureA:true,false&group=featureB:true
func GetQueryMapBoolSlice(r *http.Request, key string, delimiter string) map[string][]bool {
	rawMap := GetQueryMapSlice(r, key, delimiter)
	if len(rawMap) == 0 {
		return make(map[string][]bool)
	}

	result := make(map[string][]bool, len(rawMap))
	for groupKey, items := range rawMap {
		var bools []bool
		for _, item := range items {
			if b, err := strconv.ParseBool(item); err == nil {
				bools = append(bools, b)
			}
		}
		if len(bools) > 0 {
			result[groupKey] = bools
		}
	}

	return result
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
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	} else {
		ip = strings.Trim(ip, "[]")
	}

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

// GetHeader retrieves the first value of the specified header name from the request.
// It performs a canonical, case-insensitive header lookup and returns the value and a boolean.
func GetHeader(r *http.Request, name string) (string, bool) {
	if r == nil || r.Header == nil {
		return "", false
	}
	if values := r.Header.Values(name); len(values) > 0 {
		return values[0], true
	}
	return "", false
}

// GetBasicAuthUsernamePassword extracts username and password from Basic Authentication header
func GetBasicAuthUsernamePassword(r *http.Request) (string, string, bool) {
	return r.BasicAuth()
}
