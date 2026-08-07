package pagination

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
)

// EncodeDynamicCursor takes arbitrary values (e.g., from the last row of a query)
// and encodes them into a base64 JSON array string to be used as a cursor.
// It uses URLEncoding to be safe for HTTP query parameters.
func EncodeDynamicCursor(values ...interface{}) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	b, err := sonic.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to encode cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DecodeDynamicCursor decodes a base64 JSON array string back into an array of values.
func DecodeDynamicCursor(cursor string) ([]interface{}, error) {
	if cursor == "" {
		return nil, nil
	}

	// First try URLEncoding
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		// Fallback to StdEncoding for backwards compatibility
		b, err = base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 cursor: %w", err)
		}
	}

	var values []interface{}
	if err := sonic.Unmarshal(b, &values); err != nil {
		return nil, fmt.Errorf("failed to decode cursor values: %w", err)
	}

	return values, nil
}

// BuildDynamicKeyset generates a nested OR/AND SQL condition for keyset pagination.
// It returns a raw SQL string and a slice of arguments ready to be passed to a WHERE clause.
//
// Example for columns=["name", "status"], operators=[">", "<"], values=["App A", 1]:
// Returns SQL: "(name > ?) OR (name = ? AND status < ?)"
// Returns Args: ["App A", "App A", 1]
func BuildDynamicKeyset(columns []string, operators []string, values []interface{}) (string, []interface{}) {
	n := min(len(columns), len(operators), len(values))
	if n == 0 {
		return "", nil
	}

	var orClauses []string
	var finalArgs []interface{}

	for i := 0; i < n; i++ {
		var andClauses []string

		// Add equals for all preceding columns
		for j := 0; j < i; j++ {
			andClauses = append(andClauses, fmt.Sprintf("%s = ?", columns[j]))
			finalArgs = append(finalArgs, values[j])
		}

		// Add operator for the current column
		andClauses = append(andClauses, fmt.Sprintf("%s %s ?", columns[i], operators[i]))
		finalArgs = append(finalArgs, values[i])

		// Join AND clauses for this block
		if len(andClauses) == 1 {
			orClauses = append(orClauses, fmt.Sprintf("(%s)", andClauses[0]))
		} else {
			orClauses = append(orClauses, fmt.Sprintf("(%s)", strings.Join(andClauses, " AND ")))
		}
	}

	sqlStr := strings.Join(orClauses, " OR ")
	return sqlStr, finalArgs
}
