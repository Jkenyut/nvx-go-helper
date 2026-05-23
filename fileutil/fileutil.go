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
	switch GetMimeType(data) {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/jpg":
		return true
	default:
		return false
	}
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

// GetExtensionFromMimeType returns the file extension for a given MIME type.
func GetExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/jpg":
		return ".jpg"
	case "application/pdf":
		return ".pdf"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/x-m4a":
		return ".m4a"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "application/vnd.ms-excel":
		return ".xls"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx"
	case "application/vnd.ms-powerpoint":
		return ".ppt"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ".pptx"
	case "video/mp4":
		return ".mp4"
	case "video/mpeg":
		return ".mpeg"
	case "video/ogg":
		return ".ogg"
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	case "application/zip":
		return ".zip"
	case "application/x-rar-compressed":
		return ".rar"
	case "application/x-7z-compressed":
		return ".7z"
	case "text/plain":
		return ".txt"
	case "text/html":
		return ".html"
	case "text/css":
		return ".css"
	case "text/javascript":
		return ".js"
	case "application/json":
		return ".json"
	case "application/xml":
		return ".xml"
	case "application/x-javascript":
		return ".js"
	case "application/javascript":
		return ".js"
	case "application/x-shockwave-flash":
		return ".swf"
	case "application/rtf":
		return ".rtf"
	case "application/postscript":
		return ".ai"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tif"
	default:
		return ""
	}
}

// GetMimeTypeFromExtension returns the MIME type for a given file extension.
// This is the reverse of GetExtensionFromMimeType.
// It returns "application/octet-stream" if the extension is not found.
func GetMimeTypeFromExtension(extension string) string {
	switch extension {
	case ".jpg":
		return "image/jpeg"
	case ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/x-m4a"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".mp4":
		return "video/mp4"
	case ".mpeg":
		return "video/mpeg"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	case ".7z":
		return "application/x-7z-compressed"
	case ".txt":
		return "text/plain"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".swf":
		return "application/x-shockwave-flash"
	case ".rtf":
		return "application/rtf"
	case ".ai":
		return "application/postscript"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".tif":
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}
