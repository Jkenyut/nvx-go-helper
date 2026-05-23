package fileutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal file", "document.pdf", "document.pdf"},
		{"Path traversal", "../../../etc/passwd", "passwd"},
		{"Special chars", "my#file*name!.png", "myfilename.png"},
		{"Empty fallback", "!@#$%", "unnamed_file"},
		{"With spaces", "my image.jpg", "myimage.jpg"},
		{"Valid symbols", "my_file-v1.0.tar.gz", "my_file-v1.0.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMimeType(t *testing.T) {
	// Fake JPEG header
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	assert.Equal(t, "image/jpeg", GetMimeType(jpegData))

	// Fake PNG header
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	assert.Equal(t, "image/png", GetMimeType(pngData))

	// Plain text
	textData := []byte("hello world")
	assert.Equal(t, "text/plain; charset=utf-8", GetMimeType(textData))
}

func TestIsSafeImage(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	assert.True(t, IsSafeImage(jpegData))

	textData := []byte("<html><body>Hello</body></html>")
	assert.False(t, IsSafeImage(textData))
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{25165824, "24.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatFileSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetExtensionFromMimeType(t *testing.T) {
	assert.Equal(t, ".jpg", GetExtensionFromMimeType("image/jpeg"))
	assert.Equal(t, ".png", GetExtensionFromMimeType("image/png"))
	assert.Equal(t, ".pdf", GetExtensionFromMimeType("application/pdf"))
	assert.Equal(t, "", GetExtensionFromMimeType("unknown/type"))
}

func TestGetMimeTypeFromExtension(t *testing.T) {
	assert.Equal(t, "image/jpeg", GetMimeTypeFromExtension(".jpg"))
	assert.Equal(t, "image/jpeg", GetMimeTypeFromExtension(".jpeg"))
	assert.Equal(t, "application/pdf", GetMimeTypeFromExtension(".pdf"))
	assert.Equal(t, "application/octet-stream", GetMimeTypeFromExtension(".unknown"))
}
