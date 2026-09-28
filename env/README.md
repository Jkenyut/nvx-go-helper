# Environment Helper (`/env`)

Type-safe environment variable parsing with clean fallback defaults.

## 📖 Quickstart & Examples

```go
import "github.com/Jkenyut/nvx-go-helper/env"

// String fallback
dbHost := env.GetString("DB_HOST", "127.0.0.1")

// Integer fallback
port := env.GetInt("PORT", 8080)

// Boolean fallback (parses "true", "1", "yes", "on")
debug := env.GetBool("DEBUG", false)

// Float64 fallback
rate := env.GetFloat64("RATE_LIMIT", 100.5)

// Comma-separated list fallback
corsOrigins := env.GetSlice("CORS_ORIGINS", []string{"*"})
```
