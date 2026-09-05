package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDetector() *TypeDetector {
	return NewTypeDetector().
		WithColumnOverride("custom_code", TypeInteger).
		WithBooleanSuffixes("is_active", "flag").
		WithFloatSuffixes("price", "discount_rate", "latitude", "markup").
		WithIntegerSuffixes("version", "rate_limit_rpm").
		WithDateSuffixes("birth_date", "issue_date").
		WithTimeSuffixes("opening_time").
		WithJSONSuffixes("metadata", "settings").
		WithTimestampSuffixes("created_at").
		WithUUIDSuffixes("id", "user_id_str").
		WithBytesSuffixes("hash", "avatar_blob")
}

func TestDetectColumnType(t *testing.T) {
	detector := newTestDetector()

	assert.Equal(t, TypeBoolean, detector.DetectColumnType("is_active"))
	assert.Equal(t, TypeBoolean, detector.DetectColumnType("ga_is_active"))
	assert.Equal(t, TypeBoolean, detector.DetectColumnType("status_flag"))
	assert.Equal(t, TypeFloat, detector.DetectColumnType("price"))
	assert.Equal(t, TypeFloat, detector.DetectColumnType("discount_rate"))
	assert.Equal(t, TypeFloat, detector.DetectColumnType("item_markup"))
	assert.Equal(t, TypeFloat, detector.DetectColumnType("latitude"))
	assert.Equal(t, TypeInteger, detector.DetectColumnType("rate_limit_rpm"))
	assert.Equal(t, TypeInteger, detector.DetectColumnType("custom_code"))
	assert.Equal(t, TypeDate, detector.DetectColumnType("birth_date"))
	assert.Equal(t, TypeDate, detector.DetectColumnType("issue_date"))
	assert.Equal(t, TypeTime, detector.DetectColumnType("opening_time"))
	assert.Equal(t, TypeJSON, detector.DetectColumnType("user_metadata"))
	assert.Equal(t, TypeTimestamp, detector.DetectColumnType("created_at"))
	assert.Equal(t, TypeTimestamp, detector.DetectColumnType("ga_created_at"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("id"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("ga_id"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("user_id_str"))
	assert.Equal(t, TypeBytes, detector.DetectColumnType("file_hash"))
	assert.Equal(t, TypeBytes, detector.DetectColumnType("avatar_blob"))
	assert.Equal(t, TypeString, detector.DetectColumnType("name"))

	// Table-qualified columns (with dot)
	assert.Equal(t, TypeBoolean, detector.DetectColumnType("users.is_active"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("u.id"))
	assert.Equal(t, TypeTimestamp, detector.DetectColumnType("orders.created_at"))
	assert.Equal(t, TypeInteger, detector.DetectColumnType("items.custom_code"))

	// WithSuffixes API
	dSuffix := NewTypeDetector().WithSuffixes(TypeUUID, "guid")
	assert.Equal(t, TypeUUID, dSuffix.DetectColumnType("user_guid"))

	// Default unconfigured detector defaults to string
	unconfigured := NewTypeDetector()
	assert.Equal(t, TypeString, unconfigured.DetectColumnType("is_active"))
}

func TestNormalizeFilterValue(t *testing.T) {
	d := newTestDetector()

	t.Run("boolean", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("is_active", []string{"true"})
		require.NoError(t, err)
		assert.Equal(t, true, val)

		_, err = d.NormalizeFilterValue("is_active", []string{"not_bool"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidFilterValue)
	})

	t.Run("integer single and slice", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("version", []string{"42"})
		require.NoError(t, err)
		assert.Equal(t, 42, val)

		vals, err := d.NormalizeFilterValue("version", []string{"1", "2", "3"})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, vals)

		_, err = d.NormalizeFilterValue("version", []string{"invalid"})
		require.Error(t, err)
	})

	t.Run("float single and slice", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("price", []string{"19.99"})
		require.NoError(t, err)
		assert.Equal(t, 19.99, val)

		vals, err := d.NormalizeFilterValue("price", []string{"10.5", "20.25"})
		require.NoError(t, err)
		assert.Equal(t, []float64{10.5, 20.25}, vals)

		_, err = d.NormalizeFilterValue("price", []string{"invalid_float"})
		require.Error(t, err)
	})

	t.Run("uuid single and slice", func(t *testing.T) {
		uid := uuid.New()
		val, err := d.NormalizeFilterValue("id", []string{uid.String()})
		require.NoError(t, err)
		assert.Equal(t, uid, val)

		uid2 := uuid.New()
		vals, err := d.NormalizeFilterValue("id", []string{uid.String(), uid2.String()})
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{uid, uid2}, vals)

		_, err = d.NormalizeFilterValue("id", []string{"invalid-uuid"})
		require.Error(t, err)
	})

	t.Run("date", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("birth_date", []string{"2026-09-05"})
		require.NoError(t, err)
		dt, ok := val.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2026, dt.Year())
		assert.Equal(t, time.Month(9), dt.Month())
		assert.Equal(t, 5, dt.Day())

		_, err = d.NormalizeFilterValue("birth_date", []string{"not-a-date"})
		require.Error(t, err)
	})

	t.Run("time", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("opening_time", []string{"14:30:00"})
		require.NoError(t, err)
		assert.Equal(t, "14:30:00", val)

		_, err = d.NormalizeFilterValue("opening_time", []string{"invalid_time"})
		require.Error(t, err)
	})

	t.Run("json", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("metadata", []string{`{"role":"admin"}`})
		require.NoError(t, err)
		assert.Equal(t, `{"role":"admin"}`, val)

		_, err = d.NormalizeFilterValue("metadata", []string{`{invalid_json`})
		require.Error(t, err)
	})

	t.Run("timestamp", func(t *testing.T) {
		nowStr := "2026-09-05T12:00:00Z"
		val, err := d.NormalizeFilterValue("created_at", []string{nowStr})
		require.NoError(t, err)
		assert.IsType(t, time.Time{}, val)

		_, err = d.NormalizeFilterValue("created_at", []string{"invalid-date"})
		require.Error(t, err)

		_, err = d.NormalizeFilterValue("created_at", []string{nowStr, nowStr})
		require.Error(t, err)
	})

	t.Run("bytes", func(t *testing.T) {
		// Standard Base64 with padding
		val, err := d.NormalizeFilterValue("file_hash", []string{"aGVsbG8="})
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), val)

		// PostgreSQL hex prefix \x
		valPg, err := d.NormalizeFilterValue("file_hash", []string{`\x68656c6c6f`})
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), valPg)

		// MySQL/programming hex prefix 0x
		valMysql, err := d.NormalizeFilterValue("file_hash", []string{"0x68656c6c6f"})
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), valMysql)

		// Invalid hex prefix should return error
		_, err = d.NormalizeFilterValue("file_hash", []string{`\xnot-hex`})
		require.Error(t, err)

		// Hex whose length is multiple of 4 (must decode as hex, NOT Base64)
		valHex4, err := d.NormalizeFilterValue("file_hash", []string{"68656c6c"})
		require.NoError(t, err)
		assert.Equal(t, []byte("hell"), valHex4)

		// SHA256 (64 hex characters) must decode as 32 hex bytes, NOT Base64
		shaHex := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		valSha, err := d.NormalizeFilterValue("file_hash", []string{shaHex})
		require.NoError(t, err)
		assert.Len(t, valSha, 32)

		// URL-safe Base64
		valUrlB64, err := d.NormalizeFilterValue("file_hash", []string{"-_=="})
		require.NoError(t, err)
		assert.NotEmpty(t, valUrlB64)

		// Plain bytes fallback
		valPlain, err := d.NormalizeFilterValue("file_hash", []string{"raw-bytes-val"})
		require.NoError(t, err)
		assert.Equal(t, []byte("raw-bytes-val"), valPlain)
	})

	t.Run("string", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("name", []string{"test"})
		require.NoError(t, err)
		assert.Equal(t, "test", val)
	})

	t.Run("empty", func(t *testing.T) {
		val, err := d.NormalizeFilterValue("name", nil)
		require.NoError(t, err)
		assert.Nil(t, val)
	})
}

func TestNormalizeCursorValues(t *testing.T) {
	d := newTestDetector()

	uid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	cols := []string{"created_at", "id", "price", "birth_date", "metadata", "version"}
	rawVals := []any{now.Format(time.RFC3339), uid.String(), "49.95", "2026-09-05", `{"key":"val"}`, float64(10)}

	normalized, err := d.NormalizeCursorValues(cols, rawVals)
	require.NoError(t, err)
	assert.Len(t, normalized, 6)
	assert.Equal(t, now, normalized[0])
	assert.Equal(t, uid, normalized[1])
	assert.Equal(t, 49.95, normalized[2])
	assert.IsType(t, time.Time{}, normalized[3])
	assert.Equal(t, `{"key":"val"}`, normalized[4])
	assert.Equal(t, int64(10), normalized[5])

	t.Run("date rfc3339 cursor fallback", func(t *testing.T) {
		cols := []string{"birth_date"}
		rawVals := []any{"2026-09-05T15:04:05Z"}
		normalized, err := d.NormalizeCursorValues(cols, rawVals)
		require.NoError(t, err)
		dt, ok := normalized[0].(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2026, dt.Year())
		assert.Equal(t, time.Month(9), dt.Month())
		assert.Equal(t, 5, dt.Day())
		assert.Equal(t, 0, dt.Hour())
	})

	t.Run("varied integer and float types", func(t *testing.T) {
		cols := []string{"version", "price"}
		rawVals := []any{int32(100), float32(12.5)}
		normalized, err := d.NormalizeCursorValues(cols, rawVals)
		require.NoError(t, err)
		assert.Equal(t, int64(100), normalized[0])
		assert.Equal(t, float64(float32(12.5)), normalized[1])
	})

	t.Run("override column type", func(t *testing.T) {
		dOver := d.Clone().WithOverride("custom_code", TypeInteger)
		val, err := dOver.NormalizeFilterValue("custom_code", []string{"999"})
		require.NoError(t, err)
		assert.Equal(t, 999, val)

		cur, err := dOver.NormalizeCursorValues([]string{"custom_code"}, []any{"999"})
		require.NoError(t, err)
		assert.Equal(t, int64(999), cur[0])
	})

	t.Run("bytes cursor decoding", func(t *testing.T) {
		cols := []string{"avatar_blob", "avatar_blob", "avatar_blob", "avatar_blob"}
		rawVals := []any{
			[]byte("raw"),
			"aGVsbG8=",
			`\x68656c6c6f`,
			[]any{float64(104), float64(105)},
		}
		normalized, err := d.NormalizeCursorValues(cols, rawVals)
		require.NoError(t, err)
		assert.Equal(t, []byte("raw"), normalized[0])
		assert.Equal(t, []byte("hello"), normalized[1])
		assert.Equal(t, []byte("hello"), normalized[2])
		assert.Equal(t, []byte("hi"), normalized[3])

		// Invalid hex cursor
		_, err = d.NormalizeCursorValues([]string{"avatar_blob"}, []any{`\xinvalid`})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCursorValue)
	})

	t.Run("length mismatch", func(t *testing.T) {
		_, err := d.NormalizeCursorValues([]string{"id"}, []any{1, 2})
		require.ErrorIs(t, err, ErrCursorLengthMismatch)
	})
}

func TestDetectorFormatsAndClone(t *testing.T) {
	d := NewTypeDetector().
		WithTimestampFormats("02/01/2006 15:04:05").
		WithTimeFormats("3:04PM").
		WithTimestampSuffixes("logged_at").
		WithTimeSuffixes("start_time")

	clone := d.Clone()

	val, err := clone.NormalizeFilterValue("user_logged_at", []string{"05/09/2026 12:30:00"})
	require.NoError(t, err)
	assert.IsType(t, time.Time{}, val)

	timeVal, err := clone.NormalizeFilterValue("event_start_time", []string{"3:30PM"})
	require.NoError(t, err)
	assert.Equal(t, "3:30PM", timeVal)
}

func TestPackageLevelNormalizers(t *testing.T) {
	val, err := NormalizeFilterValue("name", []string{"sample"})
	require.NoError(t, err)
	assert.Equal(t, "sample", val)

	cur, err := NormalizeCursorValues([]string{"name"}, []any{"sample"})
	require.NoError(t, err)
	assert.Equal(t, []any{"sample"}, cur)
}

func TestSanitizeLimit(t *testing.T) {
	assert.Equal(t, DefaultLimit, SanitizeLimit(0))
	assert.Equal(t, DefaultLimit, SanitizeLimit(-5))
	assert.Equal(t, 50, SanitizeLimit(50))
	assert.Equal(t, MaxLimit, SanitizeLimit(100000))
	assert.Equal(t, 200, SanitizeLimit(500, 200))
}
