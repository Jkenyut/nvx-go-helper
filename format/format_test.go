package format

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"budi santoso", "Budi Santoso"},
		{"ADMIN", "Admin"},
		{"user-role", "User-Role"},
		{"  trim me  ", "  Trim Me  "},
		{"", ""},
		{"a", "A"},
		{"under_score", "Under_Score"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Title(tt.input))
		})
	}
}

func TestFormatBRINorek(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid 15 digits", "348601006415103", "3486-01-006415-10-3"},
		{"with dashes", "3486-01006415103", "3486-01-006415-10-3"},
		{"with spaces", "3486 01006415103", "3486-01-006415-10-3"},
		{"too short", "123", ""},
		{"exactly 15", "123456789012345", "1234-56-789012-34-5"},
		{"more than 15", "12345678901234567", "1234-56-789012-34-5"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, BRINorek(tt.input))
		})
	}
}

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0,00"},
		{7, "7,00"},
		{70.5, "70,50"},
		{999, "999,00"},
		{1000, "1.000,00"},
		{1234.56, "1.234,56"},
		{12345.67, "12.345,67"},
		{123456.78, "123.456,78"},
		{1234567.89, "1.234.567,89"},
		{12345678.90, "12.345.678,90"},
		{987654321.12, "987.654.321,12"},
		{-5000.75, "-5.000,75"},
	}

	for _, tt := range tests {
		t.Run(strconv.FormatFloat(tt.input, 'f', 2, 64), func(t *testing.T) {
			assert.Equal(t, tt.expected, Rupiah(tt.input))
		})
	}
}

func TestDollar(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0.00"},
		{1234.56, "1,234.56"},
		{1234567.89, "1,234,567.89"},
		{-5000.75, "-5,000.75"},
		{999, "999.00"},
		{0.99, "0.99"},
	}

	for _, tt := range tests {
		t.Run(strconv.FormatFloat(tt.input, 'f', 2, 64), func(t *testing.T) {
			assert.Equal(t, tt.expected, Dollar(tt.input))
		})
	}
}

func TestToString(t *testing.T) {
	now := time.Now()
	zeroTime := time.Time{}

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello world", "hello world"},
		{"int", 12345, "12345"},
		{"int8", int8(127), "127"},
		{"int16", int16(32000), "32000"},
		{"int32", int32(123456), "123456"},
		{"int64", int64(999999999), "999999999"},
		{"uint", uint(42), "42"},
		{"uint8", uint8(255), "255"},
		{"uint16", uint16(65535), "65535"},
		{"uint32", uint32(123456), "123456"},
		{"uint64", uint64(123456789), "123456789"},
		{"float64", 99.98765, "99.98765"},
		{"float32", float32(12.34), "12.34"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"[]byte", []byte("byte array"), "byte array"},
		{"time.Time valid", now, now.Format(time.RFC3339)},
		{"time.Time zero", zeroTime, ""},
		{"nil", nil, ""},
		{"pointer nil", (*string)(nil), ""},
		{"nil map", (map[string]string)(nil), ""},
		{"nil slice", ([]int)(nil), ""},
		{"nil chan", (chan int)(nil), ""},
		{"map → JSON", map[string]any{"name": "Budi", "age": 30}, `{"name":"Budi","age":30}`},
		{"struct → JSON", struct{ Name string }{Name: "Siti"}, `{"Name":"Siti"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
	}{
		{"int", int(123), 123},
		{"int8", int8(123), 123},
		{"int16", int16(123), 123},
		{"int32", int32(123), 123},
		{"int64", int64(123), 123},
		{"uint", uint(123), 123},
		{"uint8", uint8(123), 123},
		{"uint16", uint16(123), 123},
		{"uint32", uint32(123), 123},
		{"uint64", uint64(123), 123},
		{"float32", float32(123.45), 123},
		{"float64", float64(123.45), 123},
		{"string valid", "123", 123},
		{"string invalid", "abc", 0},
		{"nil", nil, 0},
		{"bool unsupported", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ToInt64(tt.input))
		})
	}
}

func TestToSafeString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"normal string", "hello world", "hello_world"},
		{"with slashes", "user/name:123", "user_name_123"},
		{"with backslash", "path\\to\\file", "path_to_file"},
		{"empty", "", "empty"},
		{"whitespace only", "   ", "empty"},
		{"integer", 42, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ToSafeString(tt.input))
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"john@example.com", "j***@example.com"},
		{"ab@example.com", "a***@example.com"},
		{"x@y.com", "x***@y.com"},
		{"", ""},
		{"invalid-email", ""},
		{"@domain.com", ""},
		{"user@", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskEmail(tt.input))
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskPhone(tt.input))
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello, World!", 10, "Hello, ..."},
		{"Hi", 10, "Hi"},
		{"Hello", 5, "Hello"},
		{"Hello", 4, "H..."},
		{"Hello", 3, "..."},
		{"Hello", 2, "Hello"}, // maxLen < 3 returns original
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Truncate(tt.input, tt.maxLen))
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		pad      rune
		expected string
	}{
		{"42", 5, '0', "00042"},
		{"hello", 3, ' ', "hello"},
		{"", 3, '*', "***"},
		{"abc", 3, '0', "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, PadLeft(tt.input, tt.length, tt.pad))
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		pad      rune
		expected string
	}{
		{"42", 5, '0', "42000"},
		{"hello", 3, ' ', "hello"},
		{"", 3, '*', "***"},
		{"abc", 3, '0', "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, PadRight(tt.input, tt.length, tt.pad))
		})
	}
}
