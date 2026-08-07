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
	val := r.FormValue(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetQueryInt retrieves a query parameter as an integer, returning defaultValue if it's missing or invalid.
func GetQueryInt(r *http.Request, key string, defaultValue int) int {
	val := r.FormValue(key)
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
	val := r.FormValue(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// GetQueryStringSlice retrieves a query parameter and splits it by comma into a slice of strings.
// It automatically trims whitespace and removes empty segments.
// Useful for supporting multiple filter codes (e.g. ?code=APP1,APP2).
func GetQueryStringSlice(r *http.Request, key string) []string {
	val := r.FormValue(key)
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
	val := r.FormValue(key)
	if val == "" {
		return nil
	}
	
	var result []int
	parts := strings.Split(val, ",")
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if clean == "" {
			continue
		}
		num, err := strconv.Atoi(clean)
		if err == nil {
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
	result := make(map[string][]int)

	values := r.URL.Query()[key]
	if len(values) == 0 {
		return result
	}

	for _, val := range values {
		parts := strings.SplitN(val, delimiter, 2)
		if len(parts) != 2 {
			continue // Ignore if it doesn't match the delimiter format
		}
		
		groupKey := strings.TrimSpace(parts[0])
		if groupKey == "" {
			continue
		}

		rawValues := strings.Split(parts[1], ",")
		for _, rv := range rawValues {
			clean := strings.TrimSpace(rv)
			if clean == "" {
				continue
			}
			num, err := strconv.Atoi(clean)
			if err == nil {
				result[groupKey] = append(result[groupKey], num)
			}
		}
	}

	return result
}

// GetQueryMapBoolSlice retrieves grouped query parameters and converts the values into booleans.
// Invalid booleans are safely ignored.
// Example format: ?group=featureA:true,false&group=featureB:true
func GetQueryMapBoolSlice(r *http.Request, key string, delimiter string) map[string][]bool {
	result := make(map[string][]bool)

	values := r.URL.Query()[key]
	if len(values) == 0 {
		return result
	}

	for _, val := range values {
		parts := strings.SplitN(val, delimiter, 2)
		if len(parts) != 2 {
			continue // Ignore if it doesn't match the delimiter format
		}
		
		groupKey := strings.TrimSpace(parts[0])
		if groupKey == "" {
			continue
		}

		rawValues := strings.Split(parts[1], ",")
		for _, rv := range rawValues {
			clean := strings.TrimSpace(rv)
			if clean == "" {
				continue
			}
			b, err := strconv.ParseBool(clean)
			if err == nil {
				result[groupKey] = append(result[groupKey], b)
			}
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

// GetBasicAuthUsernamePassword extracts username and password from Basic Authentication header
func GetBasicAuthUsernamePassword(r *http.Request) (string, string, bool) {
	return r.BasicAuth()
}
