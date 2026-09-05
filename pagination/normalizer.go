// Package pagination provides robust pagination and filter utilities.
package pagination

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidFilterValue indicates that a filter query parameter contains an incompatible type value.
	ErrInvalidFilterValue = errors.New("invalid filter value format")
	// ErrInvalidCursorValue indicates that a cursor token contains a value with an incompatible type.
	ErrInvalidCursorValue = errors.New("invalid cursor value format")
	// ErrCursorLengthMismatch indicates the cursor token does not match the required number of sort columns.
	ErrCursorLengthMismatch = errors.New("cursor length mismatch")
)

// ColumnType represents the database column data type for query parameter conversion.
type ColumnType int

// Supported database column types for type coercion and normalization.
const (
	TypeString ColumnType = iota
	TypeInteger
	TypeBoolean
	TypeUUID
	TypeTimestamp
)

// String returns the string representation of ColumnType.
func (c ColumnType) String() string {
	switch c {
	case TypeInteger:
		return "integer"
	case TypeBoolean:
		return "boolean"
	case TypeUUID:
		return "uuid"
	case TypeTimestamp:
		return "timestamp"
	default:
		return "string"
	}
}

var defaultTimestampFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999-07",
	"2006-01-02 15:04:05-07",
	"2006-01-02 15:04:05.999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// TypeDetector determines column data types based on column naming conventions and explicit overrides.
type TypeDetector struct {
	booleanSuffixes   []string
	integerSuffixes   []string
	timestampSuffixes []string
	uuidSuffixes      []string
	overrides         map[string]ColumnType
	timestampFormats  []string
}

// NewTypeDetector creates a new TypeDetector with standard enterprise column naming conventions.
func NewTypeDetector() *TypeDetector {
	return &TypeDetector{
		booleanSuffixes: []string{
			"is_active", "is_verified", "is_deleted", "is_revoked", "is_enabled",
			"has_access", "can_edit", "status_active",
		},
		integerSuffixes: []string{
			"count", "total", "amount", "quantity", "version", "status",
			"role", "order", "sort_order", "level", "rpm", "attempts",
			"limit", "offset", "type", "failed_login_attempts", "rate_limit_rpm",
		},
		timestampSuffixes: []string{
			"created_at", "updated_at", "deleted_at", "expires_at", "revoked_at",
			"date", "timestamp", "last_login_at",
		},
		uuidSuffixes: []string{
			"id", "app_id", "user_id", "actor_id", "target_user_id",
			"created_by", "updated_by", "parent_id", "tenant_id", "account_id",
		},
		overrides:        make(map[string]ColumnType),
		timestampFormats: defaultTimestampFormats,
	}
}

// DefaultDetector is the globally accessible default column type detector.
var DefaultDetector = NewTypeDetector()

// Clone creates an independent copy of TypeDetector.
func (d *TypeDetector) Clone() *TypeDetector {
	cp := &TypeDetector{
		booleanSuffixes:   append([]string(nil), d.booleanSuffixes...),
		integerSuffixes:   append([]string(nil), d.integerSuffixes...),
		timestampSuffixes: append([]string(nil), d.timestampSuffixes...),
		uuidSuffixes:      append([]string(nil), d.uuidSuffixes...),
		overrides:         make(map[string]ColumnType, len(d.overrides)),
		timestampFormats:  append([]string(nil), d.timestampFormats...),
	}
	for k, v := range d.overrides {
		cp.overrides[k] = v
	}
	return cp
}

// WithBooleanSuffixes adds custom boolean column suffixes.
func (d *TypeDetector) WithBooleanSuffixes(suffixes ...string) *TypeDetector {
	d.booleanSuffixes = append(d.booleanSuffixes, suffixes...)
	return d
}

// WithIntegerSuffixes adds custom integer column suffixes.
func (d *TypeDetector) WithIntegerSuffixes(suffixes ...string) *TypeDetector {
	d.integerSuffixes = append(d.integerSuffixes, suffixes...)
	return d
}

// WithTimestampSuffixes adds custom timestamp column suffixes.
func (d *TypeDetector) WithTimestampSuffixes(suffixes ...string) *TypeDetector {
	d.timestampSuffixes = append(d.timestampSuffixes, suffixes...)
	return d
}

// WithUUIDSuffixes adds custom UUID column suffixes.
func (d *TypeDetector) WithUUIDSuffixes(suffixes ...string) *TypeDetector {
	d.uuidSuffixes = append(d.uuidSuffixes, suffixes...)
	return d
}

// WithOverride explicitly sets the column type for a specific column name.
func (d *TypeDetector) WithOverride(column string, colType ColumnType) *TypeDetector {
	d.overrides[strings.ToLower(strings.TrimSpace(column))] = colType
	return d
}

// WithColumnOverride is an alias for WithOverride.
func (d *TypeDetector) WithColumnOverride(column string, colType ColumnType) *TypeDetector {
	return d.WithOverride(column, colType)
}

func hasMatch(col string, names []string) bool {
	for _, name := range names {
		if col == name || strings.HasSuffix(col, "_"+name) || strings.HasPrefix(col, name+"_") {
			return true
		}
	}
	return false
}

// DetectColumnType infers the column data type from column names (supporting table prefixes like ga_id, gr_is_active).
func (d *TypeDetector) DetectColumnType(col string) ColumnType {
	col = strings.ToLower(strings.TrimSpace(col))
	if t, exists := d.overrides[col]; exists {
		return t
	}

	switch {
	case hasMatch(col, d.booleanSuffixes):
		return TypeBoolean
	case hasMatch(col, d.integerSuffixes):
		return TypeInteger
	case hasMatch(col, d.timestampSuffixes):
		return TypeTimestamp
	case hasMatch(col, d.uuidSuffixes):
		return TypeUUID
	default:
		return TypeString
	}
}

// DetectColumnType infers column type using the default detector.
func DetectColumnType(col string) ColumnType {
	return DefaultDetector.DetectColumnType(col)
}

func (d *TypeDetector) parseTimestamp(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range d.timestampFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

// NormalizeFilterValue converts raw query string filter values into type-safe Go representations
// ready for parameterized SQL queries.
func (d *TypeDetector) NormalizeFilterValue(dbCol string, rawVals []string) (any, error) {
	if len(rawVals) == 0 {
		return nil, nil
	}

	switch d.DetectColumnType(dbCol) {
	case TypeBoolean:
		b, err := strconv.ParseBool(rawVals[0])
		if err != nil {
			return nil, fmt.Errorf("%w: column %s expects boolean, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
		}
		return b, nil

	case TypeInteger:
		if len(rawVals) == 1 {
			num, err := strconv.Atoi(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects integer, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return num, nil
		}
		nums := make([]int, 0, len(rawVals))
		for _, v := range rawVals {
			num, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects integer in list, got %q", ErrInvalidFilterValue, dbCol, v)
			}
			nums = append(nums, num)
		}
		return nums, nil

	case TypeUUID:
		if len(rawVals) == 1 {
			u, err := uuid.Parse(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects UUID, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return u, nil
		}
		uuids := make([]uuid.UUID, 0, len(rawVals))
		for _, v := range rawVals {
			u, err := uuid.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects UUID in list, got %q", ErrInvalidFilterValue, dbCol, v)
			}
			uuids = append(uuids, u)
		}
		return uuids, nil

	case TypeTimestamp:
		if len(rawVals) == 1 {
			t, err := d.parseTimestamp(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects timestamp, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return t, nil
		}
		return nil, fmt.Errorf("%w: column %s expects single timestamp", ErrInvalidFilterValue, dbCol)

	default: // TypeString
		if len(rawVals) == 1 {
			return rawVals[0], nil
		}
		return rawVals, nil
	}
}

// NormalizeFilterValue normalizes raw filter values using DefaultDetector.
func NormalizeFilterValue(dbCol string, rawVals []string) (any, error) {
	return DefaultDetector.NormalizeFilterValue(dbCol, rawVals)
}

// NormalizeCursorValues validates and converts raw JSON-unmarshaled cursor values into strict,
// typed values matching database column types (e.g. converting ISO-8601 strings to time.Time,
// strings to uuid.UUID, float64 to int64).
func (d *TypeDetector) NormalizeCursorValues(columns []string, cursorValues []any) ([]any, error) {
	if len(columns) == 0 || len(cursorValues) == 0 {
		return nil, nil
	}

	if len(columns) != len(cursorValues) {
		return nil, fmt.Errorf("%w: expected %d values, got %d", ErrCursorLengthMismatch, len(columns), len(cursorValues))
	}

	normalized := make([]any, len(cursorValues))
	for i, col := range columns {
		normVal, err := d.normalizeCursorValue(col, d.DetectColumnType(col), cursorValues[i])
		if err != nil {
			return nil, err
		}
		normalized[i] = normVal
	}

	return normalized, nil
}

// NormalizeCursorValues normalizes cursor values using DefaultDetector.
func NormalizeCursorValues(columns []string, cursorValues []any) ([]any, error) {
	return DefaultDetector.NormalizeCursorValues(columns, cursorValues)
}

func (d *TypeDetector) normalizeCursorValue(col string, colType ColumnType, val any) (any, error) {
	if val == nil {
		return nil, nil
	}

	switch colType {
	case TypeTimestamp:
		switch v := val.(type) {
		case string:
			t, err := d.parseTimestamp(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects timestamp cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return t, nil
		case time.Time:
			return v.UTC(), nil
		default:
			return nil, fmt.Errorf("%w: column %s expects timestamp cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeUUID:
		switch v := val.(type) {
		case string:
			u, err := uuid.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects UUID cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return u, nil
		case uuid.UUID:
			return v, nil
		default:
			return nil, fmt.Errorf("%w: column %s expects UUID cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeInteger:
		switch v := val.(type) {
		case float64:
			return int64(v), nil
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case string:
			num, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects integer cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return num, nil
		default:
			return nil, fmt.Errorf("%w: column %s expects integer cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeBoolean:
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects boolean cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return b, nil
		default:
			return nil, fmt.Errorf("%w: column %s expects boolean cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	default: // TypeString
		switch v := val.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	}
}

// SanitizeLimit clamps limit between 1 and maxLimit (defaulting to MaxLimit: 99999).
// If limit <= 0, it falls back to DefaultLimit (10).
func SanitizeLimit(limit int, maxLimit ...int) int {
	max := MaxLimit
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		max = maxLimit[0]
	}

	if limit <= 0 {
		return DefaultLimit
	}
	if limit > max {
		return max
	}
	return limit
}
