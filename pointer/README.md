# Pointer Helper (`/pointer`)

The pointer package converts primitive literal values into pointers and provides safe dereferencing functions.

## 1. Usage Examples

```go
import "github.com/Jkenyut/nvx-go-helper/pointer"

// Convert literal values into pointers
isActive := pointer.Of(true)   // *bool
maxRetries := pointer.Of(5)    // *int
status := pointer.Of("active") // *string

// Read pointers with fallback values
val := pointer.Get(isActive, false) // true
nilVal := pointer.Get[int](nil, 10) // 10
```
