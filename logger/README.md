# Logger Helper (`/logger`)

Structured logging wrapper integrating **Zerolog** with standard log levels, console formatting, and activity context correlation.

## 📖 Quickstart & Examples

```go
import (
	"github.com/Jkenyut/nvx-go-helper/logger"
	"github.com/rs/zerolog/log"
)

// Initialize logger
logger.Init(logger.Config{
	Level:       "debug",
	PrettyPrint: true, // Human-readable console formatting for local development
})

log.Info().Str("module", "payment").Msg("processing payment transaction")
```
