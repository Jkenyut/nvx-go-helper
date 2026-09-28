# Slice Utilities (`/sliceutil`)

High-performance, generic slice manipulation utilities: Map, Filter, Chunk, Unique, and ordering reversals.

## 📖 Quickstart & Examples

```go
import "github.com/Jkenyut/nvx-go-helper/sliceutil"

numbers := []int{1, 2, 3, 4, 5}

// 1. Transform / Map
strings := sliceutil.Map(numbers, func(n int) string {
	return fmt.Sprintf("num-%d", n)
})

// 2. Filter
evens := sliceutil.Filter(numbers, func(n int) bool {
	return n%2 == 0
}) // [2, 4]

// 3. Chunk
chunks := sliceutil.Chunk(numbers, 2) // [[1, 2], [3, 4], [5]]

// 4. Unique / Deduplicate
dupes := []string{"a", "b", "a", "c"}
unique := sliceutil.Unique(dupes) // ["a", "b", "c"]

// 5. In-place / Reversal
reversed := sliceutil.Reverse([]int{1, 2, 3}) // [3, 2, 1]
```
