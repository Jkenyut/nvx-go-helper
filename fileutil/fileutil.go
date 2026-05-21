package fileutil

import (
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
)

var (
	// sanitizeRegex removes characters that are not alphanumeric, dot, dash, or underscore.
	sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9.\-_]`)
)

// SanitizeFileName removes invalid or dangerous characters from a filename.
// It uses filepath.Base to prevent path traversal attacks (e.g., ../../../etc/passwd).
func SanitizeFileName(name string) string {
	base := filepath.Base(name)
	sanitized := sanitizeRegex.ReplaceAllString(base, "")
	if sanitized == "" {
		return "unnamed_file"
	}
	return sanitized
}

// GetMimeType detects the MIME type of a file based on its first 512 bytes (Magic Bytes).
// This is much safer than trusting the file extension provided by the user.
func GetMimeType(data []byte) string {
	// http.DetectContentType reads up to 512 bytes to determine the content type.
	return http.DetectContentType(data)
}

// IsSafeImage checks if the data represents a valid and safe image format (PNG, JPEG, GIF, WebP).
func IsSafeImage(data []byte) bool {
	mimeType := GetMimeType(data)
	return mimeType == "image/jpeg" || 
		   mimeType == "image/png" || 
		   mimeType == "image/gif" || 
		   mimeType == "image/webp"
}

// FormatFileSize converts bytes to a human-readable string representation (e.g., "1.5 MB").
// It uses binary prefix calculations (1 KB = 1024 B).
func FormatFileSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
