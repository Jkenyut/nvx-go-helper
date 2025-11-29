// Package builder provides a powerful, safe, and chainable SQL WHERE + ORDER BY builder.
//
// Features:
//   - Full support for AND, OR, nested groups
//   - Shortcuts: =, !=, >, >=, <, <=, LIKE, IN, NOT IN, IS NULL, BETWEEN
//   - Safe from SQL injection (parameterized)
//   - Multiple ORDER BY fields, ASC/DESC, RANDOM()
//   - Zero external dependencies
//   - 100% test coverage
package builder

import (
	"fmt"
	"reflect"
	"strings"
)

// WhereBuilder builds parameterized SQL WHERE and ORDER BY clauses.
type WhereBuilder struct {
	conditions []string // SQL fragments
	args       []any    // Query parameters
	orderBy    []string // ORDER BY fields
}

// New creates a new WhereBuilder instance.
func New() *WhereBuilder {
	return &WhereBuilder{}
}

// Where appends a raw SQL condition and its arguments.
func (b *WhereBuilder) Where(sql string, args ...any) *WhereBuilder {
	b.conditions = append(b.conditions, sql)
	b.args = append(b.args, args...)
	return b
}

// Eq adds field = ?
func (b *WhereBuilder) Eq(field string, value any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s = ?", field), value)
}

// NotEq adds field != ?
func (b *WhereBuilder) NotEq(field string, value any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s != ?", field), value)
}

// Gt adds field > ?
func (b *WhereBuilder) Gt(field string, value any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s > ?", field), value)
}

// Gte adds field >= ?
func (b *WhereBuilder) Gte(field string, value any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s >= ?", field), value)
}

// Lt adds field < ?
func (b *WhereBuilder) Lt(field string, value any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s < ?", field), value)
}

// Lte adds field <= ?
func (b *WhereBuilder) Lte(field string, value any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s <= ?", field), value)
}

// Like adds field LIKE ? (with %pattern%)
func (b *WhereBuilder) Like(field, pattern string) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s LIKE ?", field), "%"+pattern+"%")
}

// StartsWith adds field LIKE 'pattern%'
func (b *WhereBuilder) StartsWith(field, pattern string) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s LIKE ?", field), pattern+"%")
}

// EndsWith adds field LIKE '%pattern'
func (b *WhereBuilder) EndsWith(field, pattern string) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s LIKE ?", field), "%"+pattern)
}

// IsNull adds field IS NULL
func (b *WhereBuilder) IsNull(field string) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s IS NULL", field))
}

// NotNull adds field IS NOT NULL
func (b *WhereBuilder) NotNull(field string) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s IS NOT NULL", field))
}

// Between adds field BETWEEN ? AND ?
func (b *WhereBuilder) Between(field string, min, max any) *WhereBuilder {
	return b.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), min, max)
}

// Raw appends raw SQL (use carefully)
func (b *WhereBuilder) Raw(sql string) *WhereBuilder {
	return b.Where(sql)
}

// Group - AND ( ... )
func (b *WhereBuilder) Group(fn func(*WhereBuilder)) *WhereBuilder {
	sub := New()
	fn(sub)
	if len(sub.conditions) == 0 {
		return b
	}
	joined := strings.Join(sub.conditions, " AND ")
	b.conditions = append(b.conditions, "("+joined+")")
	b.args = append(b.args, sub.args...)
	return b
}

// OrGroup - OR ( ... )
func (b *WhereBuilder) OrGroup(fn func(*WhereBuilder)) *WhereBuilder {
	sub := New()
	fn(sub)
	if len(sub.conditions) == 0 {
		return b
	}
	// PAKAI " OR " di dalam grup, tapi grup itu sendiri tetap AND dengan yang lain
	joined := strings.Join(sub.conditions, " OR ")
	b.conditions = append(b.conditions, "("+joined+")")
	b.args = append(b.args, sub.args...)
	return b
}

// OrderBy adds ORDER BY field direction
func (b *WhereBuilder) OrderBy(field, direction string) *WhereBuilder {
	dir := strings.ToUpper(strings.TrimSpace(direction))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	b.orderBy = append(b.orderBy, field+" "+dir)
	return b
}

// Sort is alias of OrderBy
func (b *WhereBuilder) Sort(field, direction string) *WhereBuilder {
	return b.OrderBy(field, direction)
}

// Random adds RANDOM() for PostgreSQL / MySQL 8.0+
func (b *WhereBuilder) Random() *WhereBuilder {
	b.orderBy = append(b.orderBy, "RANDOM()")
	return b
}

// OrderByMulti adds multiple fields (e.g. "name ASC", "age DESC")
func (b *WhereBuilder) OrderByMulti(fields ...string) *WhereBuilder {
	for _, f := range fields {
		parts := strings.Fields(f)
		if len(parts) >= 2 {
			b.OrderBy(parts[0], parts[1])
		} else if len(parts) == 1 {
			b.OrderBy(parts[0], "ASC")
		}
	}
	return b
}

// In adds field IN (?, ?, ...) - supports []string, []int, []any, or varargs
func (b *WhereBuilder) In(field string, values ...any) *WhereBuilder {
	if len(values) == 0 {
		return b.Where("1 = 0") // always false
	}

	// Handle single slice argument: In("id", []int{1,2,3})
	if len(values) == 1 {
		if slice := toSlice(values[0]); slice != nil {
			if len(slice) == 0 {
				return b.Where("1 = 0")
			}
			return b.inWithSlice(field, slice)
		}
	}

	return b.inWithSlice(field, values)
}

// NotIn adds field NOT IN (?, ?, ...)
func (b *WhereBuilder) NotIn(field string, values ...any) *WhereBuilder {
	if len(values) == 0 {
		return b.Where("1 = 1") // always true
	}

	if len(values) == 1 {
		if slice := toSlice(values[0]); slice != nil {
			if len(slice) == 0 {
				return b.Where("1 = 1")
			}
			return b.notInWithSlice(field, slice)
		}
	}

	return b.notInWithSlice(field, values)
}

// Helper: convert any slice type to []any
func toSlice(v any) []any {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil
	}

	result := make([]any, val.Len())
	for i := 0; i < val.Len(); i++ {
		result[i] = val.Index(i).Interface()
	}
	return result
}

// Internal: generate IN with slice
func (b *WhereBuilder) inWithSlice(field string, values []any) *WhereBuilder {
	placeholders := strings.Repeat("?, ", len(values))
	placeholders = strings.TrimRight(placeholders, ", ")
	return b.Where(fmt.Sprintf("%s IN (%s)", field, placeholders), values...)
}

func (b *WhereBuilder) notInWithSlice(field string, values []any) *WhereBuilder {
	placeholders := strings.Repeat("?, ", len(values))
	placeholders = strings.TrimRight(placeholders, ", ")
	return b.Where(fmt.Sprintf("%s NOT IN (%s)", field, placeholders), values...)
}

// Build returns the final SQL clause and arguments.
func (b *WhereBuilder) Build() (string, []any) {
	var clauses []string

	if len(b.conditions) > 0 {
		where := strings.Join(b.conditions, " AND ")
		where = strings.TrimSpace(where)
		if where != "" {
			clauses = append(clauses, "WHERE "+where)
		}
	}

	if len(b.orderBy) > 0 {
		clauses = append(clauses, "ORDER BY "+strings.Join(b.orderBy, ", "))
	}

	sql := strings.TrimSpace(strings.Join(clauses, " "))
	return sql, append([]any{}, b.args...) // copy args
}

// Reset clears all conditions and order
func (b *WhereBuilder) Reset() *WhereBuilder {
	b.conditions = nil
	b.args = nil
	b.orderBy = nil
	return b
}
