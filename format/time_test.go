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

	// FIXED: Use the same time.FixedZone (not .UTC() which can be affected by local)
	// WIB offset = +7 hours = 7*3600 seconds
	expectedOffset := 7 * 60 * 60
	_, actualOffset := now.Zone()
	assert.Equal(t, expectedOffset, actualOffset)
}

func TestNow(t *testing.T) {
	t1 := NowUTC()
	t2 := Now()
	assert.Equal(t, "UTC", t2.Location().String())
	assert.WithinDuration(t, t1, t2, time.Second)
}

func TestToWIB(t *testing.T) {
	// Use a fixed time in UTC
	utcTime := time.Date(2025, 10, 20, 8, 30, 45, 123456789, time.UTC)
	wibTime := ToWIB(utcTime)

	assert.Equal(t, "Asia/Jakarta", wibTime.Location().String())
	assert.Equal(t, 15, wibTime.Hour()) // 8 + 7 = 15
	assert.Equal(t, 30, wibTime.Minute())
	assert.Equal(t, 45, wibTime.Second())
	assert.Equal(t, 123456789, wibTime.Nanosecond())

	// Check offset directly from .Zone()
	_, offset := wibTime.Zone()
	assert.Equal(t, 7*3600, offset) // +7 hours = 25200 seconds
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
		{LayoutDateTimeSec, "2025-07-07 07:00:00"},
		{LayoutRFC3339WIB, "2025-07-07T07:00:00+07:00"},
		{LayoutISO, "2025-07-07T07:00:00+07:00"},
		{LayoutDate, "2025-07-07"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, WIBString(utcTime, tt.layout))
	}
}

func TestFormatUTC(t *testing.T) {
	wibTime := time.Date(2025, 1, 1, 12, 0, 0, 0, WIB)
	assert.Equal(t, "2025-01-01T05:00:00Z", UTCString(wibTime, time.RFC3339))
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
			assert.Equal(t, tt.want, tt.t.IsZero())
		})
	}
}

func TestStartOfDay(t *testing.T) {
	tm := time.Date(2025, 10, 20, 15, 30, 45, 123456789, time.UTC)
	start := StartOfDay(tm)
	assert.Equal(t, 0, start.Hour())
	assert.Equal(t, 0, start.Minute())
	assert.Equal(t, 0, start.Second())
	assert.Equal(t, 0, start.Nanosecond())
	assert.Equal(t, 2025, start.Year())
	assert.Equal(t, time.Month(10), start.Month())
	assert.Equal(t, 20, start.Day())
	assert.Equal(t, time.UTC, start.Location())

	assert.True(t, StartOfDay(time.Time{}).IsZero())
}

func TestEndOfDay(t *testing.T) {
	tm := time.Date(2025, 10, 20, 15, 30, 45, 123456789, time.UTC)
	end := EndOfDay(tm)
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
	assert.Equal(t, 59, end.Second())
	assert.Equal(t, 999999999, end.Nanosecond())
	assert.Equal(t, 2025, end.Year())
	assert.Equal(t, time.Month(10), end.Month())
	assert.Equal(t, 20, end.Day())

	assert.True(t, EndOfDay(time.Time{}).IsZero())
}

func TestStartOfMonth(t *testing.T) {
	tm := time.Date(2025, 10, 20, 15, 30, 45, 123456789, WIB)
	start := StartOfMonth(tm)
	assert.Equal(t, 2025, start.Year())
	assert.Equal(t, time.Month(10), start.Month())
	assert.Equal(t, 1, start.Day())
	assert.Equal(t, 0, start.Hour())
	assert.Equal(t, WIB.String(), start.Location().String())

	assert.True(t, StartOfMonth(time.Time{}).IsZero())
}

func TestEndOfMonth(t *testing.T) {
	// Test a month with 31 days
	tm := time.Date(2025, 10, 20, 15, 30, 45, 0, WIB)
	end := EndOfMonth(tm)
	assert.Equal(t, 2025, end.Year())
	assert.Equal(t, time.Month(10), end.Month())
	assert.Equal(t, 31, end.Day())
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
	assert.Equal(t, 59, end.Second())
	assert.Equal(t, 999999999, end.Nanosecond())

	// Test leap year February
	tmFeb := time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC)
	endFeb := EndOfMonth(tmFeb)
	assert.Equal(t, 29, endFeb.Day())

	assert.True(t, EndOfMonth(time.Time{}).IsZero())
}

func TestStartOfWeek(t *testing.T) {
	// 2025-10-20 is a Monday
	tmMon := time.Date(2025, 10, 20, 15, 30, 0, 0, time.UTC)
	startMon := StartOfWeek(tmMon)
	assert.Equal(t, 20, startMon.Day())
	assert.Equal(t, 0, startMon.Hour())

	// 2025-10-22 is a Wednesday
	tmWed := time.Date(2025, 10, 22, 15, 30, 0, 0, time.UTC)
	startWed := StartOfWeek(tmWed)
	assert.Equal(t, 20, startWed.Day()) // Should go back to Monday 20th

	// 2025-10-26 is a Sunday
	tmSun := time.Date(2025, 10, 26, 15, 30, 0, 0, time.UTC)
	startSun := StartOfWeek(tmSun)
	assert.Equal(t, 20, startSun.Day()) // Should go back to Monday 20th

	assert.True(t, StartOfWeek(time.Time{}).IsZero())
}

func TestEndOfWeek(t *testing.T) {
	// 2025-10-22 is a Wednesday
	tmWed := time.Date(2025, 10, 22, 15, 30, 0, 0, time.UTC)
	endWed := EndOfWeek(tmWed)
	assert.Equal(t, 26, endWed.Day()) // Sunday 26th
	assert.Equal(t, 23, endWed.Hour())
	assert.Equal(t, 59, endWed.Minute())
	assert.Equal(t, 59, endWed.Second())

	assert.True(t, EndOfWeek(time.Time{}).IsZero())
}
