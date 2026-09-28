# Format Helper (`/format`)

String, currency, and date formatting utilities: Indonesian Rupiah formatting, rune-safe string truncation, keyword masking, and calendar boundaries.

## 📖 Quickstart & Examples

```go
import "github.com/Jkenyut/nvx-go-helper/format"

// 1. Indonesian Rupiah formatting
rupiah := format.FormatRupiah(1500000) // "Rp 1.500.000"

// 2. Sensitive Keyword Masking
maskedEmail := format.MaskEmail("budi.santoso@example.com") // "bu***so@example.com"
maskedPhone := format.MaskPhone("081234567890")             // "0812****7890"

// 3. Rune-Safe Truncation (Preserves multi-byte UTF-8 safely)
short := format.Truncate("Belajar Pemrograman Go", 10, "...") // "Belajar..."

// 4. Date Boundaries
startOfDay := format.StartOfDay(time.Now())
endOfMonth := format.EndOfMonth(time.Now())
```
