// Package pagination provides robust pagination, keyset cursor, and type normalization utilities.
package pagination

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

// ColumnType represents universal database column types for query parameter conversion.
type ColumnType int

// Supported database column types for type coercion and normalization.
const (
	TypeString ColumnType = iota
	TypeInteger
	TypeFloat
	TypeBoolean
	TypeUUID
	TypeDate
	TypeTime
	TypeTimestamp
	TypeJSON
	TypeBytes
)

// String returns the string representation of ColumnType.
func (c ColumnType) String() string {
	switch c {
	case TypeInteger:
		return "integer"
	case TypeFloat:
		return "float"
	case TypeBoolean:
		return "boolean"
	case TypeUUID:
		return "uuid"
	case TypeDate:
		return "date"
	case TypeTime:
		return "time"
	case TypeTimestamp:
		return "timestamp"
	case TypeJSON:
		return "json"
	case TypeBytes:
		return "bytes"
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

var defaultTimeFormats = []string{
	"15:04:05.999999999",
	"15:04:05.999999",
	"15:04:05",
	"15:04:05Z07:00",
	"15:04:05-07:00",
	"15:04",
}

// TypeDetector determines column data types based on naming conventions and explicit overrides
// without imposing hardcoded opinions.
type TypeDetector struct {
	booleanSuffixes   []string
	integerSuffixes   []string
	floatSuffixes     []string
	uuidSuffixes      []string
	dateSuffixes      []string
	timeSuffixes      []string
	timestampSuffixes []string
	jsonSuffixes      []string
	bytesSuffixes     []string
	overrides         map[string]ColumnType
	timestampFormats  []string
	timeFormats       []string
}

// NewTypeDetector creates a clean, non-opinionated TypeDetector with no hardcoded column lists.
func NewTypeDetector() *TypeDetector {
	return &TypeDetector{
		overrides:        make(map[string]ColumnType),
		timestampFormats: defaultTimestampFormats,
		timeFormats:      defaultTimeFormats,
	}
}

// DefaultDetector is the globally accessible default column type detector.
var DefaultDetector = NewTypeDetector()

// Clone creates an independent copy of TypeDetector.
func (d *TypeDetector) Clone() *TypeDetector {
	cp := &TypeDetector{
		booleanSuffixes:   append([]string(nil), d.booleanSuffixes...),
		integerSuffixes:   append([]string(nil), d.integerSuffixes...),
		floatSuffixes:     append([]string(nil), d.floatSuffixes...),
		uuidSuffixes:      append([]string(nil), d.uuidSuffixes...),
		dateSuffixes:      append([]string(nil), d.dateSuffixes...),
		timeSuffixes:      append([]string(nil), d.timeSuffixes...),
		timestampSuffixes: append([]string(nil), d.timestampSuffixes...),
		jsonSuffixes:      append([]string(nil), d.jsonSuffixes...),
		bytesSuffixes:     append([]string(nil), d.bytesSuffixes...),
		overrides:         make(map[string]ColumnType, len(d.overrides)),
		timestampFormats:  append([]string(nil), d.timestampFormats...),
		timeFormats:       append([]string(nil), d.timeFormats...),
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

// WithFloatSuffixes adds custom float column suffixes.
func (d *TypeDetector) WithFloatSuffixes(suffixes ...string) *TypeDetector {
	d.floatSuffixes = append(d.floatSuffixes, suffixes...)
	return d
}

// WithUUIDSuffixes adds custom UUID column suffixes.
func (d *TypeDetector) WithUUIDSuffixes(suffixes ...string) *TypeDetector {
	d.uuidSuffixes = append(d.uuidSuffixes, suffixes...)
	return d
}

// WithDateSuffixes adds custom date column suffixes.
func (d *TypeDetector) WithDateSuffixes(suffixes ...string) *TypeDetector {
	d.dateSuffixes = append(d.dateSuffixes, suffixes...)
	return d
}

// WithTimeSuffixes adds custom time-of-day column suffixes.
func (d *TypeDetector) WithTimeSuffixes(suffixes ...string) *TypeDetector {
	d.timeSuffixes = append(d.timeSuffixes, suffixes...)
	return d
}

// WithTimestampSuffixes adds custom timestamp column suffixes.
func (d *TypeDetector) WithTimestampSuffixes(suffixes ...string) *TypeDetector {
	d.timestampSuffixes = append(d.timestampSuffixes, suffixes...)
	return d
}

// WithJSONSuffixes adds custom JSON column suffixes.
func (d *TypeDetector) WithJSONSuffixes(suffixes ...string) *TypeDetector {
	d.jsonSuffixes = append(d.jsonSuffixes, suffixes...)
	return d
}

// WithBytesSuffixes adds custom bytes/binary column suffixes.
func (d *TypeDetector) WithBytesSuffixes(suffixes ...string) *TypeDetector {
	d.bytesSuffixes = append(d.bytesSuffixes, suffixes...)
	return d
}

// WithTimestampFormats adds custom timestamp parsing layouts.
func (d *TypeDetector) WithTimestampFormats(formats ...string) *TypeDetector {
	d.timestampFormats = append(d.timestampFormats, formats...)
	return d
}

// WithTimeFormats adds custom time parsing layouts.
func (d *TypeDetector) WithTimeFormats(formats ...string) *TypeDetector {
	d.timeFormats = append(d.timeFormats, formats...)
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

// WithSuffixes registers suffixes for a specific column type.
func (d *TypeDetector) WithSuffixes(colType ColumnType, suffixes ...string) *TypeDetector {
	switch colType {
	case TypeBoolean:
		return d.WithBooleanSuffixes(suffixes...)
	case TypeInteger:
		return d.WithIntegerSuffixes(suffixes...)
	case TypeFloat:
		return d.WithFloatSuffixes(suffixes...)
	case TypeUUID:
		return d.WithUUIDSuffixes(suffixes...)
	case TypeDate:
		return d.WithDateSuffixes(suffixes...)
	case TypeTime:
		return d.WithTimeSuffixes(suffixes...)
	case TypeTimestamp:
		return d.WithTimestampSuffixes(suffixes...)
	case TypeJSON:
		return d.WithJSONSuffixes(suffixes...)
	case TypeBytes:
		return d.WithBytesSuffixes(suffixes...)
	default:
		return d
	}
}

func hasMatch(col string, names []string) bool {
	for _, name := range names {
		cleanName := strings.TrimPrefix(name, "_")
		if col == cleanName || strings.HasSuffix(col, "_"+cleanName) {
			return true
		}
	}
	return false
}

// DetectColumnType infers the column data type from column names (supporting table prefixes like ga_id, u.id, users.is_active).
func (d *TypeDetector) DetectColumnType(col string) ColumnType {
	col = strings.ToLower(strings.TrimSpace(col))
	if t, exists := d.overrides[col]; exists {
		return t
	}

	baseCol := col
	if idx := strings.LastIndexByte(col, '.'); idx != -1 {
		baseCol = col[idx+1:]
		if t, exists := d.overrides[baseCol]; exists {
			return t
		}
	}

	switch {
	case hasMatch(baseCol, d.booleanSuffixes):
		return TypeBoolean
	case hasMatch(baseCol, d.floatSuffixes):
		return TypeFloat
	case hasMatch(baseCol, d.integerSuffixes):
		return TypeInteger
	case hasMatch(baseCol, d.uuidSuffixes):
		return TypeUUID
	case hasMatch(baseCol, d.dateSuffixes):
		return TypeDate
	case hasMatch(baseCol, d.timeSuffixes):
		return TypeTime
	case hasMatch(baseCol, d.timestampSuffixes):
		return TypeTimestamp
	case hasMatch(baseCol, d.jsonSuffixes):
		return TypeJSON
	case hasMatch(baseCol, d.bytesSuffixes):
		return TypeBytes
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

func (d *TypeDetector) parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	for _, layout := range d.timestampFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date (expected YYYY-MM-DD): %s", s)
}

func (d *TypeDetector) parseTime(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range d.timeFormats {
		if _, err := time.Parse(layout, s); err == nil {
			return s, nil
		}
	}
	return "", fmt.Errorf("unable to parse time: %s", s)
}

func isHexString(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func (d *TypeDetector) parseBytes(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return []byte{}, nil
	}

	// 1. Explicit hex prefixes: \x (PostgreSQL), 0x (MySQL/programming)
	if strings.HasPrefix(s, `\x`) || strings.HasPrefix(s, `\X`) ||
		strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid hex encoding: %w", err)
		}
		return b, nil
	}

	// 2. Base64 with padding or base64-specific characters (=, +, /, _, -)
	if strings.ContainsAny(s, "=+/_-") {
		if b, err := base64.StdEncoding.DecodeString(s); err == nil {
			return b, nil
		}
		if b, err := base64.URLEncoding.DecodeString(s); err == nil {
			return b, nil
		}
		if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
			return b, nil
		}
		if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
			return b, nil
		}
	}

	// 3. Pure hex string (even length, characters 0-9, a-f, A-F)
	// Prioritized over Base64 to prevent corrupting SHA256, MD5, and binary hashes.
	if len(s)%2 == 0 && isHexString(s) {
		if b, err := hex.DecodeString(s); err == nil {
			return b, nil
		}
	}

	// 4. Raw Base64 (unpadded alphanumeric base64)
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return b, nil
	}

	// 5. Fallback to raw string bytes
	return []byte(s), nil
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

	case TypeFloat:
		if len(rawVals) == 1 {
			f, err := strconv.ParseFloat(rawVals[0], 64)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects float, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return f, nil
		}
		floats := make([]float64, 0, len(rawVals))
		for _, v := range rawVals {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects float in list, got %q", ErrInvalidFilterValue, dbCol, v)
			}
			floats = append(floats, f)
		}
		return floats, nil

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

	case TypeDate:
		if len(rawVals) == 1 {
			t, err := d.parseDate(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects date (YYYY-MM-DD), got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return t, nil
		}
		dates := make([]time.Time, 0, len(rawVals))
		for _, v := range rawVals {
			t, err := d.parseDate(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects date in list, got %q", ErrInvalidFilterValue, dbCol, v)
			}
			dates = append(dates, t)
		}
		return dates, nil

	case TypeTime:
		if len(rawVals) == 1 {
			tStr, err := d.parseTime(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects time format, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return tStr, nil
		}
		return rawVals, nil

	case TypeTimestamp:
		if len(rawVals) == 1 {
			t, err := d.parseTimestamp(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects timestamp, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return t, nil
		}
		return nil, fmt.Errorf("%w: column %s expects single timestamp", ErrInvalidFilterValue, dbCol)

	case TypeJSON:
		if len(rawVals) == 1 {
			if !json.Valid([]byte(rawVals[0])) {
				return nil, fmt.Errorf("%w: column %s expects valid JSON, got %q", ErrInvalidFilterValue, dbCol, rawVals[0])
			}
			return rawVals[0], nil
		}
		return nil, fmt.Errorf("%w: column %s expects single JSON string", ErrInvalidFilterValue, dbCol)

	case TypeBytes:
		if len(rawVals) == 1 {
			b, err := d.parseBytes(rawVals[0])
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects valid bytes, got %q: %w", ErrInvalidFilterValue, dbCol, rawVals[0], err)
			}
			return b, nil
		}
		bytesList := make([][]byte, 0, len(rawVals))
		for _, v := range rawVals {
			b, err := d.parseBytes(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects valid bytes in list, got %q: %w", ErrInvalidFilterValue, dbCol, v, err)
			}
			bytesList = append(bytesList, b)
		}
		return bytesList, nil

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
// typed values matching database column types.
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
	case TypeDate:
		switch v := val.(type) {
		case string:
			t, err := d.parseDate(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects date cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return t, nil
		case time.Time:
			return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC), nil
		default:
			return nil, fmt.Errorf("%w: column %s expects date cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeTime:
		switch v := val.(type) {
		case string:
			tStr, err := d.parseTime(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects time cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return tStr, nil
		case time.Time:
			return v.Format("15:04:05"), nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

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
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case int32:
			return int64(v), nil
		case int16:
			return int64(v), nil
		case int8:
			return int64(v), nil
		case uint:
			return int64(v), nil
		case uint64:
			return int64(v), nil
		case uint32:
			return int64(v), nil
		case uint16:
			return int64(v), nil
		case uint8:
			return int64(v), nil
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		case string:
			num, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects integer cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return num, nil
		default:
			return nil, fmt.Errorf("%w: column %s expects integer cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeFloat:
		switch v := val.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int16:
			return float64(v), nil
		case int8:
			return float64(v), nil
		case uint:
			return float64(v), nil
		case uint64:
			return float64(v), nil
		case uint32:
			return float64(v), nil
		case uint16:
			return float64(v), nil
		case uint8:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects float cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("%w: column %s expects float cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeBoolean:
		switch v := val.(type) {
		case bool:
			return v, nil
		case int:
			return v != 0, nil
		case int64:
			return v != 0, nil
		case int32:
			return v != 0, nil
		case float64:
			return v != 0, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects boolean cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return b, nil
		default:
			return nil, fmt.Errorf("%w: column %s expects boolean cursor, got %T", ErrInvalidCursorValue, col, val)
		}

	case TypeJSON:
		switch v := val.(type) {
		case string:
			if !json.Valid([]byte(v)) {
				return nil, fmt.Errorf("%w: column %s expects valid JSON cursor, got %q", ErrInvalidCursorValue, col, v)
			}
			return v, nil
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s cannot serialize to JSON cursor: %w", ErrInvalidCursorValue, col, err)
			}
			return string(b), nil
		}

	case TypeBytes:
		switch v := val.(type) {
		case []byte:
			return v, nil
		case string:
			b, err := d.parseBytes(v)
			if err != nil {
				return nil, fmt.Errorf("%w: column %s expects valid bytes cursor, got %q: %w", ErrInvalidCursorValue, col, v, err)
			}
			return b, nil
		case []any:
			b := make([]byte, len(v))
			for i, elem := range v {
				num, ok := elem.(float64)
				if !ok {
					return nil, fmt.Errorf("%w: column %s byte array contains non-numeric element %T", ErrInvalidCursorValue, col, elem)
				}
				b[i] = byte(num)
			}
			return b, nil
		default:
			return fmt.Sprintf("%v", v), nil
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
	allowedMax := MaxLimit
	if len(maxLimit) > 0 && maxLimit[0] > 0 {
		allowedMax = maxLimit[0]
	}

	if limit <= 0 {
		return DefaultLimit
	}
	if limit > allowedMax {
		return allowedMax
	}
	return limit
}
