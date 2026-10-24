# Environment Helper (`/env`)

The env package reads environment variables and provides fallback values when keys are missing.

## 1. Usage Examples

```go
import "github.com/Jkenyut/nvx-go-helper/env"

// Read string value with fallback
dbHost := env.GetString("DB_HOST", "127.0.0.1")

// Read integer value with fallback
port := env.GetInt("PORT", 8080)

// Read boolean value with fallback
debug := env.GetBool("DEBUG", false)

// Read float64 value with fallback
rate := env.GetFloat64("RATE_LIMIT", 100.5)

// Read comma-separated string list with fallback
corsOrigins := env.GetSlice("CORS_ORIGINS", []string{"*"})
```
