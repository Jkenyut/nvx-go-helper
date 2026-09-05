package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectColumnType(t *testing.T) {
	detector := NewTypeDetector().
		WithColumnOverride("custom_code", TypeInteger).
		WithBooleanSuffixes("flag").
		WithUUIDSuffixes("id_str")

	assert.Equal(t, TypeBoolean, detector.DetectColumnType("is_active"))
	assert.Equal(t, TypeBoolean, detector.DetectColumnType("ga_is_active"))
	assert.Equal(t, TypeBoolean, detector.DetectColumnType("status_flag"))
	assert.Equal(t, TypeInteger, detector.DetectColumnType("rate_limit_rpm"))
	assert.Equal(t, TypeInteger, detector.DetectColumnType("custom_code"))
	assert.Equal(t, TypeTimestamp, detector.DetectColumnType("created_at"))
	assert.Equal(t, TypeTimestamp, detector.DetectColumnType("ga_created_at"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("id"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("ga_id"))
	assert.Equal(t, TypeUUID, detector.DetectColumnType("user_id_str"))
	assert.Equal(t, TypeString, detector.DetectColumnType("name"))
}

func TestNormalizeFilterValue(t *testing.T) {
	t.Run("boolean", func(t *testing.T) {
		val, err := NormalizeFilterValue("is_active", []string{"true"})
		require.NoError(t, err)
		assert.Equal(t, true, val)

		_, err = NormalizeFilterValue("is_active", []string{"not_bool"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidFilterValue)
	})

	t.Run("integer single and slice", func(t *testing.T) {
		val, err := NormalizeFilterValue("version", []string{"42"})
		require.NoError(t, err)
		assert.Equal(t, 42, val)

		vals, err := NormalizeFilterValue("version", []string{"1", "2", "3"})
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, vals)

		_, err = NormalizeFilterValue("version", []string{"invalid"})
		require.Error(t, err)
	})

	t.Run("uuid single and slice", func(t *testing.T) {
		uid := uuid.New()
		val, err := NormalizeFilterValue("id", []string{uid.String()})
		require.NoError(t, err)
		assert.Equal(t, uid, val)

		uid2 := uuid.New()
		vals, err := NormalizeFilterValue("id", []string{uid.String(), uid2.String()})
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{uid, uid2}, vals)

		_, err = NormalizeFilterValue("id", []string{"invalid-uuid"})
		require.Error(t, err)
	})

	t.Run("timestamp", func(t *testing.T) {
		nowStr := "2026-09-05T12:00:00Z"
		val, err := NormalizeFilterValue("created_at", []string{nowStr})
		require.NoError(t, err)
		assert.IsType(t, time.Time{}, val)

		_, err = NormalizeFilterValue("created_at", []string{"invalid-date"})
		require.Error(t, err)

		_, err = NormalizeFilterValue("created_at", []string{nowStr, nowStr})
		require.Error(t, err)
	})

	t.Run("string", func(t *testing.T) {
		val, err := NormalizeFilterValue("name", []string{"test"})
		require.NoError(t, err)
		assert.Equal(t, "test", val)
	})

	t.Run("empty", func(t *testing.T) {
		val, err := NormalizeFilterValue("name", nil)
		require.NoError(t, err)
		assert.Nil(t, val)
	})
}

func TestNormalizeCursorValues(t *testing.T) {
	uid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	cols := []string{"created_at", "id", "version"}
	rawVals := []any{now.Format(time.RFC3339), uid.String(), float64(10)}

	normalized, err := NormalizeCursorValues(cols, rawVals)
	require.NoError(t, err)
	assert.Len(t, normalized, 3)
	assert.Equal(t, now, normalized[0])
	assert.Equal(t, uid, normalized[1])
	assert.Equal(t, int64(10), normalized[2])

	t.Run("length mismatch", func(t *testing.T) {
		_, err := NormalizeCursorValues([]string{"id"}, []any{1, 2})
		require.ErrorIs(t, err, ErrCursorLengthMismatch)
	})
}

func TestSanitizeLimit(t *testing.T) {
	assert.Equal(t, DefaultLimit, SanitizeLimit(0))
	assert.Equal(t, DefaultLimit, SanitizeLimit(-5))
	assert.Equal(t, 50, SanitizeLimit(50))
	assert.Equal(t, MaxLimit, SanitizeLimit(100000))
	assert.Equal(t, 200, SanitizeLimit(500, 200))
}
