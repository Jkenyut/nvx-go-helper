# NVX Go Helper

[![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)]()
[![Lint](https://img.shields.io/badge/golangci--lint-passing-brightgreen.svg)]()

**nvx-go-helper** is a comprehensive collection of **production-ready, high-performance Go utilities** designed to accelerate backend microservice and API development. Built following modern Go idioms, enterprise standards, and zero-compromise engineering.

---

## 🎯 Core Principles

- **Un-opinionated & Helper-First**: Utilities never impose artificial business limits, forced default limits, or arbitrary payload caps. You retain total control over your applications and middleware.
- **Minimal Dependencies**: Strictly standard library first. External dependencies are kept to essential, battle-tested components (`bytedance/sonic`, `go-playground/validator/v10`).
- **High Performance & Zero Allocations**: Pre-allocated memory buffers, zero-copy hashing (`io.WriteString`), in-place AEAD sealing, and hyper-fast JSON serialization via Sonic.
- **Thread-Safety & Leak Prevention**: All registries are concurrency-safe, and retry loops employ clean timer management (`time.NewTimer` + `Stop()`) preventing goroutine or timer leaks.
- **Forward Compatibility**: 100% backward-compatible public APIs with strict type safety through Go 1.18+ generics.

---

## 📦 Installation

```bash
go get github.com/Jkenyut/nvx-go-helper
```

---

## 📑 Package Directory

| Package | Path | Highlights |
| :--- | :--- | :--- |
| **`activity`** | [`/activity`](#1-activity-activity) | Context request tracking (`request_id`, `user_id`, metadata) for structured logging |
| **`cryptoutil`** | [`/cryptoutil`](#2-cryptoutil-cryptoutil) | Argon2id, AES-256-GCM, ECC (ECIES), UUID v4/v7, cryptographically secure randoms |
| **`env`** | [`/env`](#3-env-env) | Safe environment variable extraction with type-safe fallbacks |
| **`fileutil`** | [`/fileutil`](#4-file-utilities-fileutil) | Path traversal protection, deep MIME magic bytes (CFBF & OOXML zip detection) |
| **`format`** | [`/format`](#5-format-format) | Rune-safe truncation, keyword masking, Rupiah currency, date boundaries |
| **`maputil`** | [`/maputil`](#6-map-utilities-maputil) | Generic map manipulation: Keys, Values, Merge, Pick, Omit, Invert, Filter |
| **`pagination`** | [`/pagination`](#7-pagination-pagination) | Dynamic keyset (cursor) & offset pagination with zero forced default limits |
| **`pointer`** | [`/pointer`](#8-pointer-pointer) | Generic literals to pointer conversion (`pointer.Of`) |
| **`request`** | [`/request`](#9-request-request) | Fast Sonic JSON binding, body-preserving query parsing, IPv4/IPv6 client IP |
| **`response`** | [`/response`](#10-response-response) | Standardized API JSON responses powered by `bytedance/sonic` |
| **`retry`** | [`/retry`](#11-retry-retry) | Configurable exponential backoff retry with leak-free timer hygiene |
| **`sliceutil`** | [`/sliceutil`](#12-slice-utilities-sliceutil) | Generic slice transformations: Chunk, Map, Filter, Unique, Contains, IndexOf |
| **`token`** | [`/token`](#13-token-token) | ES256 JWT generation and validation with algorithm confusion prevention |
| **`validator`** | [`/validator`](#14-validator-validator) | `go-playground/validator/v10` wrapper with custom Indonesian ID tags (NIK, NPWP, Phone) |
| **`worker`** | [`/worker`](#15-worker-pool-worker) | Generic concurrency worker pool with context cancellation and progress tracking |

---

## 🚀 Package Guides & Examples

### 1. Activity (`/activity`)
Inject and extract request metadata across context boundaries. Integrates seamlessly with structured loggers (Zap, Logrus, `slog`).

```go
import "github.com/Jkenyut/nvx-go-helper/activity"

// Inject into Context (typically in HTTP Middleware)
ctx := activity.WithRequestID(context.Background(), "req-abc-123")
ctx = activity.WithUserID(ctx, "user-456")

// Extract anywhere downstream
userID, ok := activity.GetUserID(ctx)
requestID := activity.GetRequestID(ctx)

// Export all fields as a map for structured loggers
fields := activity.GetAllFieldsFromContext(ctx)
// map[string]any{"request_id": "req-abc-123", "user_id": "user-456"}
```

---

### 2. Cryptoutil (`/cryptoutil`)
Unified cryptographic primitives for password hashing, symmetric & asymmetric encryption, identifiers, and random tokens.

```go
import "github.com/Jkenyut/nvx-go-helper/cryptoutil"

// 1. Password Hashing (Argon2id in standard PHC format)
hashed, err := cryptoutil.HashPassword("UserPassword123!")
valid, err := cryptoutil.VerifyPassword("UserPassword123!", hashed)

// 2. Hybrid Asymmetric Encryption (ECIES with P-256 + AES-GCM)
privKey, pubKey, _ := cryptoutil.GenerateECCKeyPair()
ciphertext, _ := cryptoutil.EncryptECC(pubKey, []byte("sensitive-payload"))
plaintext, _ := cryptoutil.DecryptECC(privKey, ciphertext)

// 3. Symmetric Encryption (AES-256-GCM with zero-allocation Seal)
enc, _ := cryptoutil.NewAESGCMFromHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
encryptedToken, _ := enc.Encrypt(map[string]any{"account_id": "12345"})

// 4. UUIDs
uuidV4 := cryptoutil.V4() // Cryptographically random
uuidV7 := cryptoutil.V7() // Time-ordered (optimal for DB primary keys)

// 5. High-Entropy Random Strings (Numbers, Alpha, Base64, Hex)
otp := cryptoutil.Numbers(6) // "582194"
key, _ := cryptoutil.GenerateKeyHex(32)
```

---

### 3. Env (`/env`)
Safe, type-safe environment variable parsing with fallbacks.

```go
import "github.com/Jkenyut/nvx-go-helper/env"

port := env.GetInt("PORT", 8080)
dbHost := env.GetString("DB_HOST", "127.0.0.1")
debug := env.GetBool("DEBUG_MODE", false)
```

---

### 4. File Utilities (`/fileutil`)
Secure file handling, upload validation, and cross-platform path traversal defense.

```go
import "github.com/Jkenyut/nvx-go-helper/fileutil"

// Path traversal defense (strips Windows '\' separators and leading dots)
filename := fileutil.SanitizeFileName("../../.env") // "env"

// Deep magic-byte validation (inspects CFBF and OOXML zip headers)
isSafeDoc := fileutil.IsSafeDocument(fileBytes) // Validates PDF, DOCX, XLSX, PPTX, TXT, CSV
isSafeImg := fileutil.IsSafeImage(fileBytes)    // Validates JPEG, PNG, GIF, WebP

// Human-readable size formatting
size := fileutil.FormatFileSize(2097152) // "2.0 MB"
```

---

### 5. Format (`/format`)
String manipulation, UTF-8 rune-safe truncation, keyword masking, and Indonesian localized formatting.

```go
import "github.com/Jkenyut/nvx-go-helper/format"

// Rune-safe truncation (never cuts multi-byte UTF-8 or emojis mid-character)
format.Truncate("Promo Spesial 💰 Diskon!", 15) // "Promo Spesial..."

// Sensitive Keyword Masking
masked := format.MaskAfterKeywords(`{"password": "secret", "token": "abc"}`, []string{"password", "token"}, "*")

// Indonesian Rupiah & Account Number Formatting
rupiah := format.Rupiah(1500000) // "1.500.000,00"
bri := format.BRINorek("123456789012345") // "1234-56-789012-34-5"

// Date Boundaries
start := format.StartOfDay(format.Now())
end := format.EndOfMonth(format.Now())
```

---

### 6. Map Utilities (`/maputil`)
Generic-powered, reflection-free map operations with pre-allocated capacity for zero rehashing.

```go
import "github.com/Jkenyut/nvx-go-helper/maputil"

m1 := map[string]int{"a": 1, "b": 2}
m2 := map[string]int{"b": 3, "c": 4}

// Keys & Values
keys := maputil.Keys(m1)     // []string{"a", "b"}
values := maputil.Values(m1) // []int{1, 2}

// Merge (pre-calculates total capacity to eliminate rehashing)
merged := maputil.Merge(m1, m2) // map[a:1 b:3 c:4]

// Pick, Omit, Filter
picked := maputil.Pick(m1, "a") // map[a:1]
omitted := maputil.Omit(m1, "b") // map[a:1]
filtered := maputil.Filter(m1, func(k string, v int) bool { return v > 1 })
```

---

### 7. Pagination (`/pagination`)
Bidirectional dynamic keyset (cursor) and offset pagination. Un-opinionated: no forced default limits or artificial clamping. If limit is omitted, it defaults to `0` (unconstrained).

```go
import "github.com/Jkenyut/nvx-go-helper/pagination"

// 1. Traditional Offset Pagination
req := pagination.BindOffsetFilterRequest(r, map[string]string{"status": "users.status"})
pageData := pagination.NewFromInt(req.Page, req.Limit, totalCount)
resp := pagination.NewListResponse(users, pageData)

// 2. Dynamic Keyset Cursor Pagination (Supports multi-column sorting & tie-breakers)
allowedCols := map[string]string{"name": "user_name", "created_at": "user_created_at"}
cursorReq := pagination.BindCursorFilterRequest(r, allowedCols)

sortRes := pagination.PrepareDynamicSort(pagination.DynamicSortParams{
    SortBy:         cursorReq.SortBy,
    SortType:       cursorReq.SortType,
    Direction:      cursorReq.Direction,
    AllowedColumns: allowedCols,
    UniqueColumn:   "user_id",
    UniqueSortType: "DESC",
})

cursorMeta := pagination.GenerateBidirectionalCursor(
    users, cursorReq.Limit, cursorReq.Direction, cursorReq.Cursor,
    func(u User) []any { return []any{u.Name, u.CreatedAt, u.ID} },
)
cursorResp := pagination.NewCursorListResponse(users, cursorMeta)

// 3. Unified Mode (Auto-detects cursor vs offset within a single endpoint)
unified := pagination.BindUnifiedFilterRequest(r, allowedCols)
```

---

### 8. Pointer (`/pointer`)
Generic helper to convert primitive literals into pointers.

```go
import "github.com/Jkenyut/nvx-go-helper/pointer"

isActive := pointer.Of(true)
maxRetries := pointer.Of(5)
status := pointer.Of("active")
```

---

### 9. Request (`/request`)
Safe HTTP request parsing, JSON binding via `bytedance/sonic`, body-preserving query parsing, and IPv4/IPv6 client IP resolution.

```go
import "github.com/Jkenyut/nvx-go-helper/request"

// 1. JSON Binding & Validation
var req CreateUserRequest
err := request.BindAndValidate(r, &req)

// 2. Body-Preserving Query Extraction (Does NOT drain or parse request body)
status := request.GetQueryString(r, "status", "active")
limit := request.GetQueryInt(r, "limit", 0)
ids := request.GetQueryIntSlice(r, "ids") // Parses "?ids=1,2,3"

// 3. Canonical Case-Insensitive Header Extraction
apiKey, ok := request.GetHeader(r, "x-api-key")
bearer := request.GetBearerToken(r)

// 4. Safe Client IP Resolution (Supports IPv4, IPv6, and X-Forwarded-For)
clientIP := request.GetClientIP(r)
```

---

### 10. Response (`/response`)
Standardized JSON API responses with fast Sonic marshaling and fluent status helpers.

```go
import (
    "net/http"
    "github.com/Jkenyut/nvx-go-helper/response"
)

func Handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 1. Send directly to http.ResponseWriter (sets Content-Type: application/json & status code)
    response.OK(ctx, "data retrieved successfully", userPayload).WriteHTTP(w)

    // 2. Standard Error Responses
    // response.BadRequest(ctx, "invalid request parameter").WriteHTTP(w)
    // response.NotFound(ctx, "user not found").WriteHTTP(w)
    // response.WithMessageData(ctx, "validation failed", 422, validationErrors).WriteHTTP(w)

    // 3. For Web Frameworks (Gin, Echo, Fiber):
    // resp := response.Created(ctx, "user created", createdUser)
    // c.Data(resp.Meta.StatusCode, "application/json", resp.JSONMarshal())
}
```

---

### 11. Retry (`/retry`)
Configurable exponential backoff retry mechanism with guaranteed timer cleanup.

```go
import (
    "time"
    "github.com/Jkenyut/nvx-go-helper/retry"
)

// Simple retry
err := retry.Do(func() error {
    return db.Ping()
},
    retry.WithMaxAttempts(3),
    retry.WithBackoff(500*time.Millisecond, 2.0),
    retry.WithContext(ctx),
)

// Retry with return value
data, err := retry.DoWithResult(func() (*User, error) {
    return fetchRemoteUser(id)
}, retry.WithMaxAttempts(3))
```

---

### 12. Slice Utilities (`/sliceutil`)
Type-safe generic slice transformations utilizing modern Go 1.21+ standard library features.

```go
import "github.com/Jkenyut/nvx-go-helper/sliceutil"

items := []int{1, 2, 3, 4, 5}

// Chunking & Mapping
chunks := sliceutil.Chunk(items, 2) // [[1, 2], [3, 4], [5]]
mapped := sliceutil.Map(items, strconv.Itoa) // ["1", "2", "3", "4", "5"]

// Filtering & Uniqueness
evens := sliceutil.Filter(items, func(v int) bool { return v%2 == 0 })
unique := sliceutil.Unique([]string{"a", "b", "a", "c"}) // ["a", "b", "c"]

// Lookups
exists := sliceutil.Contains(items, 3) // true
index := sliceutil.IndexOf(items, 4)   // 3
```

---

### 13. Token (`/token`)
Asymmetric ES256 JWT generation and validation with algorithm confusion protection.

```go
import "github.com/Jkenyut/nvx-go-helper/token"

type CustomClaims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
}

// Generate ES256 JWT
claims := token.NewJWTClaims(CustomClaims{UserID: "100", Role: "admin"}, 24*time.Hour)
tokenStr, err := token.GenerateES256JWT(ecdsaPrivateKey, claims)

// Verify ES256 JWT (strictly enforces header alg == "ES256")
parsedClaims, err := token.VerifyES256JWT[CustomClaims](ecdsaPublicKey, tokenStr)
```

---

### 14. Validator (`/validator`)
Singleton wrapper for `go-playground/validator/v10` with user-friendly error formatting and built-in Indonesian validation tags.

```go
import "github.com/Jkenyut/nvx-go-helper/validator"

type UserProfile struct {
    Email string `validate:"required,email"`
    NIK   string `validate:"required,nik"`      // 16-digit Indonesian NIK
    NPWP  string `validate:"omitempty,npwp"`    // 15 or 16-digit Indonesian NPWP
    Phone string `validate:"required,phone_id"` // Indonesian mobile number
}

err := validator.Struct(profile)
if err != nil {
    firstMsg := validator.GetErrorFirstMsg(err) // "email: invalid email address"
    allErrors := validator.GetErrorsFullMsg(err) // "email: invalid email address, nik: invalid NIK"
}
```

---

### 15. Worker Pool (`/worker`)
Generic concurrent worker pool with context-aware cancellation, order preservation, and progress tracking.

```go
import "github.com/Jkenyut/nvx-go-helper/worker"

jobs := []worker.Job[string, int]{
    {ID: "job-1", Data: 10},
    {ID: "job-2", Data: 20},
}

cfg := worker.PoolConfig{
    NumWorkers:    4,
    PreserveOrder: true,
    OnProgress: func(completed, total int) {
        log.Printf("Completed %d of %d", completed, total)
    },
}

workerFn := func(ctx context.Context, id string, data int) (string, error) {
    return fmt.Sprintf("result-%d", data), nil
}

results, err := worker.RunGenericWorkerPool(ctx, jobs, workerFn, nil, cfg)
```

---

## 🧪 Testing & Verification

Run the test suite with Go's race detector and linting gates:

```bash
# Run all unit tests with race detection
go test -v -race ./...

# Format code
gofmt -w .

# Run static analysis
go vet ./...

# Run linter
golangci-lint run ./...
```

---

## 📄 License

This library is licensed under the [Apache License 2.0](LICENSE).
