// Package response provides helper functions to write responses and set HTTP headers.
package response

import (
	"net/http"
)

// SetHeader sets a single HTTP header on the provided ResponseWriter.
func SetHeader(w http.ResponseWriter, key, value string) {
	w.Header().Set(key, value)
}

// SetHeaders sets multiple HTTP headers on the provided ResponseWriter.
// The headers map keys are header names and values are header values.
func SetHeaders(w http.ResponseWriter, headers map[string]string) {
	for k, v := range headers {
		w.Header().Set(k, v)
	}
}

// WriteJSONResponse writes the Response as JSON to the http.ResponseWriter.
// It sets the appropriate status code and Content-Type header.
func WriteJSONResponse(w http.ResponseWriter, resp Response) {
	// Ensure status code is set
	if resp.Meta.StatusCode == 0 {
		resp.Meta.StatusCode = 200
	}
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	// Write status code
	w.WriteHeader(resp.Meta.StatusCode)
	// Encode JSON to the writer
	resp.JSONEncoder(w)
}
