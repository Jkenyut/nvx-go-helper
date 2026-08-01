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
		{"Nested path", "/var/log/app.log", "app.log"},
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

	// GIF header
	gifData := []byte("GIF89a" + string(make([]byte, 100)))
	assert.Equal(t, "image/gif", GetMimeType(gifData))
}

func TestIsSafeImage(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	assert.True(t, IsSafeImage(jpegData))

	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	assert.True(t, IsSafeImage(pngData))

	gifData := []byte("GIF89a" + string(make([]byte, 100)))
	assert.True(t, IsSafeImage(gifData))

	textData := []byte("<html><body>Hello</body></html>")
	assert.False(t, IsSafeImage(textData))

	// Binary garbage
	assert.False(t, IsSafeImage([]byte{0x00, 0x01, 0x02, 0x03}))
}

func TestIsSafeDocument(t *testing.T) {
	// PDF header
	pdfData := []byte("%PDF-1.4 some content here")
	assert.True(t, IsSafeDocument(pdfData))

	// HTML should NOT be a safe document
	htmlData := []byte("<html><body>Hello</body></html>")
	assert.False(t, IsSafeDocument(htmlData))
}

func TestIsSafeVideo(t *testing.T) {
	// Not a video (just text)
	textData := []byte("this is not a video")
	assert.False(t, IsSafeVideo(textData))

	// JPEG is not a video
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	assert.False(t, IsSafeVideo(jpegData))
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{25165824, "24.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatFileSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetExtensionFromMimeType(t *testing.T) {
	tests := []struct {
		mime string
		ext  string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/jpg", ".jpg"},
		{"application/pdf", ".pdf"},
		{"audio/mpeg", ".mp3"},
		{"audio/wav", ".wav"},
		{"audio/ogg", ".ogg"},
		{"audio/x-m4a", ".m4a"},
		{"application/msword", ".doc"},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", ".docx"},
		{"application/vnd.ms-excel", ".xls"},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
		{"application/vnd.ms-powerpoint", ".ppt"},
		{"application/vnd.openxmlformats-officedocument.presentationml.presentation", ".pptx"},
		{"video/mp4", ".mp4"},
		{"video/mpeg", ".mpeg"},
		{"video/ogg", ".ogg"},
		{"video/webm", ".webm"},
		{"video/quicktime", ".mov"},
		{"application/zip", ".zip"},
		{"application/x-rar-compressed", ".rar"},
		{"application/x-7z-compressed", ".7z"},
		{"text/plain", ".txt"},
		{"text/html", ".html"},
		{"text/css", ".css"},
		{"text/javascript", ".js"},
		{"application/json", ".json"},
		{"application/xml", ".xml"},
		{"application/x-javascript", ".js"},
		{"application/javascript", ".js"},
		{"application/x-shockwave-flash", ".swf"},
		{"application/rtf", ".rtf"},
		{"application/postscript", ".ai"},
		{"image/svg+xml", ".svg"},
		{"image/bmp", ".bmp"},
		{"image/tiff", ".tif"},
		{"unknown/type", ""},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			assert.Equal(t, tt.ext, GetExtensionFromMimeType(tt.mime))
		})
	}
}

func TestGetMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext  string
		mime string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".pdf", "application/pdf"},
		{".mp3", "audio/mpeg"},
		{".wav", "audio/wav"},
		{".ogg", "audio/ogg"},
		{".m4a", "audio/x-m4a"},
		{".doc", "application/msword"},
		{".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{".xls", "application/vnd.ms-excel"},
		{".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{".ppt", "application/vnd.ms-powerpoint"},
		{".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{".mp4", "video/mp4"},
		{".mpeg", "video/mpeg"},
		{".webm", "video/webm"},
		{".mov", "video/quicktime"},
		{".zip", "application/zip"},
		{".rar", "application/x-rar-compressed"},
		{".7z", "application/x-7z-compressed"},
		{".txt", "text/plain"},
		{".html", "text/html"},
		{".css", "text/css"},
		{".js", "application/javascript"},
		{".json", "application/json"},
		{".xml", "application/xml"},
		{".swf", "application/x-shockwave-flash"},
		{".rtf", "application/rtf"},
		{".ai", "application/postscript"},
		{".svg", "image/svg+xml"},
		{".bmp", "image/bmp"},
		{".tif", "image/tiff"},
		{".unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			assert.Equal(t, tt.mime, GetMimeTypeFromExtension(tt.ext))
		})
	}
}
