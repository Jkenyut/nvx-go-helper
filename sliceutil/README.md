# Slice Utilities (`/sliceutil`)

The sliceutil package provides generic transformation functions for Go slices.

## 1. Usage Examples

```go
import (
	"fmt"

	"github.com/Jkenyut/nvx-go-helper/sliceutil"
)

numbers := []int{1, 2, 3, 4, 5}

// 1. Transform items with Map
strings := sliceutil.Map(numbers, func(n int) string {
	return fmt.Sprintf("num-%d", n)
})

// 2. Filter items with a predicate
evens := sliceutil.Filter(numbers, func(n int) bool {
	return n%2 == 0
}) // [2, 4]

// 3. Split slice into chunks
chunks := sliceutil.Chunk(numbers, 2) // [[1, 2], [3, 4], [5]]

// 4. Remove duplicate elements
dupes := []string{"a", "b", "a", "c"}
unique := sliceutil.Unique(dupes) // ["a", "b", "c"]

// 5. Reverse slice ordering
reversed := sliceutil.Reverse([]int{1, 2, 3}) // [3, 2, 1]
```
