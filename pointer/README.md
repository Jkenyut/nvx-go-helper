# Pointer Helper (`/pointer`)

Generic utility to convert literal values and primitives into pointers safely.

## 📖 Quickstart & Examples

```go
import "github.com/Jkenyut/nvx-go-helper/pointer"

// Convert literals into pointers
isActive := pointer.Of(true)       // *bool
maxRetries := pointer.Of(5)        // *int
status := pointer.Of("active")     // *string

// Safe dereference with fallback default
val := pointer.Get(isActive, false) // true
nilVal := pointer.Get[int](nil, 10) // 10
```
