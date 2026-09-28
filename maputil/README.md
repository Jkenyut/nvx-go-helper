# Map Utilities (`/maputil`)

Generic, type-safe map manipulation utilities powered by Go generics.

## 📖 Quickstart & Examples

```go
import "github.com/Jkenyut/nvx-go-helper/maputil"

m := map[string]int{"a": 1, "b": 2, "c": 3}

// Extract Keys and Values
keys := maputil.Keys(m)       // ["a", "b", "c"]
values := maputil.Values(m)   // [1, 2, 3]

// Pick subset of keys
subset := maputil.Pick(m, "a", "c") // map[string]int{"a": 1, "c": 3}

// Omit keys
omitted := maputil.Omit(m, "b")     // map[string]int{"a": 1, "c": 3}

// Filter map
filtered := maputil.Filter(m, func(k string, v int) bool {
	return v > 1
}) // map[string]int{"b": 2, "c": 3}

// Invert keys and values
inverted := maputil.Invert(m) // map[int]string{1: "a", 2: "b", 3: "c"}

// Merge multiple maps
merged := maputil.Merge(m, map[string]int{"d": 4})
```
