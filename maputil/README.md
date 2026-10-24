# Map Utilities (`/maputil`)

The maputil package provides generic functions for Go maps with capacity pre-allocation.

## 1. Usage Examples

```go
import "github.com/Jkenyut/nvx-go-helper/maputil"

m := map[string]int{"a": 1, "b": 2, "c": 3}

// Extract keys and values
keys := maputil.Keys(m)     // ["a", "b", "c"]
values := maputil.Values(m) // [1, 2, 3]

// Select matching keys
subset := maputil.Pick(m, "a", "c") // map[string]int{"a": 1, "c": 3}

// Exclude specified keys
omitted := maputil.Omit(m, "b") // map[string]int{"a": 1, "c": 3}

// Filter map items using a predicate function
filtered := maputil.Filter(m, func(k string, v int) bool {
	return v > 1
}) // map[string]int{"b": 2, "c": 3}

// Invert map keys and values
inverted := maputil.Invert(m) // map[int]string{1: "a", 2: "b", 3: "c"}

// Merge maps with capacity pre-allocation
merged := maputil.Merge(m, map[string]int{"d": 4})
```
