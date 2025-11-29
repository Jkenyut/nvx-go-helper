package format

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimezoneConstants(t *testing.T) {
	assert.Equal(t, "UTC", UTC.String())
	assert.Equal(t, "Asia/Jakarta", WIB.String())
	assert.Equal(t, "Asia/Jakarta", Jakarta.String())
	assert.Equal(t, "Asia/Bangkok", Bangkok.String())
}

func TestNowUTC(t *testing.T) {
	now := NowUTC()
	assert.Equal(t, "UTC", now.Location().String())
	assert.WithinDuration(t, time.Now().UTC(), now, 100*time.Millisecond)
}

func TestNowWIB(t *testing.T) {
	now := NowWIB()
	assert.Equal(t, "Asia/Jakarta", now.Location().String())

	// FIXED: Gunakan time.FixedZone yang sama (bukan .UTC() yang bisa dipengaruhi local)
	// WIB offset = +7 jam = 7*3600 detik
	expectedOffset := 7 * 60 * 60
	_, actualOffset := now.Zone()
	assert.Equal(t, expectedOffset, actualOffset)
}

func TestNow(t *testing.T) {
	assert.Equal(t, NowUTC().Format(time.RFC3339Nano), Now().Format(time.RFC3339Nano))
}

func TestToWIB(t *testing.T) {
	// Gunakan waktu tetap di UTC
	utcTime := time.Date(2025, 10, 20, 8, 30, 45, 123456789, time.UTC)
	wibTime := ToWIB(utcTime)

	assert.Equal(t, "Asia/Jakarta", wibTime.Location().String())
	assert.Equal(t, 15, wibTime.Hour()) // 8 + 7 = 15
	assert.Equal(t, 30, wibTime.Minute())
	assert.Equal(t, 45, wibTime.Second())
	assert.Equal(t, 123456789, wibTime.Nanosecond())

	// Cek offset langsung dari .Zone()
	_, offset := wibTime.Zone()
	assert.Equal(t, 7*3600, offset) // +7 jam = 25200 detik
}

func TestToUTC(t *testing.T) {
	wibTime := time.Date(2025, 12, 31, 23, 59, 59, 0, WIB)
	utcTime := ToUTC(wibTime)

	assert.Equal(t, "UTC", utcTime.Location().String())
	assert.Equal(t, 16, utcTime.Hour()) // 23 - 7 = 16

	_, offset := utcTime.Zone()
	assert.Equal(t, 0, offset) // UTC offset selalu 0
}

func TestFormatWIB(t *testing.T) {
	utcTime := time.Date(2025, 7, 7, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		layout   string
		expected string
	}{
		{LayoutDateOnly, "07-07-2025"},
		{LayoutDateTime, "07-07-2025 07:00"},
		{LayoutDateTimeSec, "07-07-2025 07:00:00"},
		{LayoutDB, "2025-07-07 07:00:00"},
		{LayoutRFC3339WIB, "2025-07-07T07:00:00+07:00"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, FormatWIB(utcTime, tt.layout))
	}
}

func TestFormatUTC(t *testing.T) {
	wibTime := time.Date(2025, 1, 1, 12, 0, 0, 0, WIB)
	assert.Equal(t, "2025-01-01T05:00:00Z", FormatUTC(wibTime, time.RFC3339))
}

func TestParseRFC3339Safe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
	}{
		{"valid", "2025-04-05T10:20:30Z", false},
		{"empty", "", true},
		{"zero date", "0001-01-01T00:00:00Z", true},
		{"partial zero", "0001-01-01", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParseRFC3339Safe(tt.input)
			assert.Equal(t, tt.wantZero, got.IsZero())
		})
	}

	_, err := ParseRFC3339Safe("invalid")
	assert.Error(t, err)
}

func TestIsZeroOrDefault(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"zero", time.Time{}, true},
		{"mysql zero", time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"valid", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsZeroOrDefault(tt.t))
		})
	}
}
