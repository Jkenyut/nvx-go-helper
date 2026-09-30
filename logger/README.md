# Logger Helper (`/logger`)

Structured logging wrapper integrating **Zerolog** with asynchronous diode ring-buffer, environment-aware output formatting, OpenTelemetry trace context correlation, and standard library `log/slog` interoperability.

## 🌟 Highlights

- **Functional Options (`logger.Init`)**: Clean, flexible configuration without requiring rigid structs.
- **Environment-Aware**: Automatically outputs pretty colored logs to `stderr` in development, and high-throughput JSON to `stdout` in production.
- **Diode Ring-Buffer**: High-performance asynchronous non-blocking log writer prevents slow disk or stdout I/O from stalling HTTP requests.
- **OTel & Activity Context**: Automatically extracts `trace_id`, `span_id`, `request_id`, `correlation_id`, `user_id`, and custom metadata from `context.Context`.
- **Stdlib Slog Bridge**: `logger.Slog()` converts the Zerolog logger to standard Go `*slog.Logger` for seamless compatibility with third-party drivers.

## 📖 Quickstart & Examples

### 1. Modern Initialization with Functional Options

```go
import (
	"context"
	"github.com/Jkenyut/nvx-go-helper/logger"
)

// Initialize with functional options
logger.Init(
	logger.WithServiceName("user-service"),
	logger.WithEnv("production"), // "production" -> JSON to stdout; "development" -> console to stderr
	logger.WithPort(8080),
	logger.WithLevel("info"),     // Optional: override log level
)
defer logger.Close()

// Context-aware logging (auto-attaches trace_id, request_id, correlation_id, user_id)
logger.Info(ctx).Str("action", "checkout").Msg("order placed successfully")
```

### 2. Available Options

- `logger.WithServiceName(name string)`: Sets the service name attribute in every log line.
- `logger.WithEnv(env string)`: Sets the environment (`production`, `development`, `staging`).
- `logger.WithPort(port int)`: Sets the listening port metadata.
- `logger.WithLevel(level string)`: Sets the log level (`debug`, `info`, `warn`, `error`, `fatal`).
- `logger.WithBufferSize(size int)`: Overrides the diode ring-buffer queue capacity (default: 10,000 for prod, 1,000 for dev).
- `logger.WithWriter(w io.Writer)`: Sets a custom log sink writer (ideal for unit testing or custom file sinks).

### 3. Standard Library `slog` Interoperability

```go
// Obtain a standard library *slog.Logger backed by the high-performance Zerolog diode
driverLogger := logger.Slog()

// Or with contextual tracing:
reqSlog := logger.SlogWithContext(ctx)
reqSlog.Info("processing database migration", "table", "users")
```
