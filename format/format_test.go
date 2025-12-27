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
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, Title(tt.input))
		})
	}
}

func TestAddStringUnique(t *testing.T) {
	var slice []string

	cases := []string{"Admin", "admin", "USER", "User", "guest", "", "  ", "Manager", "MANAGER"}

	for _, c := range cases {
		AddStringUnique(c, &slice)
	}

	expected := []string{"Admin", "User", "Guest", "Manager"}
	assert.Equal(t, expected, slice)
	assert.Len(t, slice, 4)
}

func TestFormatBRINorek(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid 15 digits", "348601006415103", "3486-01-006415-10-3"},
		{"with dashes", "3486-01006415103", "3486-01-006415-10-3"},
		{"too short", "123", ""},
		{"exactly 15", "123456789012345", "1234-56-789012-34-5"},
		{"more than 15", "12345678901234567", "1234-56-789012-34-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatBRINorek(tt.input))
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
			assert.Equal(t, tt.expected, FormatRupiah(tt.input))
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
		{
			name:     "string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "int",
			input:    12345,
			expected: "12345",
		},
		{
			name:     "int64",
			input:    int64(999999999),
			expected: "999999999", // ← DULU SALAH JADI "999"
		},
		{
			name:     "float64",
			input:    99.98765,
			expected: "99.98765",
		},
		{
			name:     "float32",
			input:    float32(12.34),
			expected: "12.34",
		},
		{
			name:     "bool true",
			input:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			input:    false,
			expected: "false",
		},
		{
			name:     "[]byte",
			input:    []byte("byte array"),
			expected: "byte array",
		},
		{
			name:     "time.Time valid",
			input:    now,
			expected: now.Format(time.RFC3339),
		},
		{
			name:     "time.Time zero → empty",
			input:    zeroTime,
			expected: "",
		},
		{
			name:     "nil",
			input:    nil,
			expected: "",
		},
		{
			name:     "pointer nil",
			input:    (*string)(nil),
			expected: "",
		},
		{
			name:     "map → JSON",
			input:    map[string]any{"name": "Budi", "age": 30},
			expected: `{"age":30,"name":"Budi"}`, // urutan bisa beda, tapi isi sama
		},
		{
			name:     "struct → JSON",
			input:    struct{ Name string }{Name: "Siti"},
			expected: `{"Name":"Siti"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})

	}
}
