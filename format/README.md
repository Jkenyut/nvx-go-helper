# Format Helper (`/format`)

The format package formats Indonesian Rupiah, masks sensitive strings, truncates UTF-8 text, and calculates date boundaries.

## 1. Usage Examples

```go
import (
	"time"

	"github.com/Jkenyut/nvx-go-helper/format"
)

// 1. Format Indonesian Rupiah currency
rupiah := format.FormatRupiah(1500000) // "Rp 1.500.000"

// 2. Mask sensitive strings
maskedEmail := format.MaskEmail("budi.santoso@example.com") // "bu***so@example.com"
maskedPhone := format.MaskPhone("081234567890")             // "0812****7890"

// 3. Truncate multi-byte UTF-8 strings safely
short := format.Truncate("Belajar Pemrograman Go", 10, "...") // "Belajar..."

// 4. Calculate calendar date boundaries
startOfDay := format.StartOfDay(time.Now())
endOfMonth := format.EndOfMonth(time.Now())
```
