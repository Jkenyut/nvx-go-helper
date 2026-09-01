# NVX Go Helper

**nvx-go-helper** is a collection of **production-grade** utility functions designed to accelerate backend service development in Go (Golang). This library is built according to enterprise standards.

**Key Design Principles:**
- **Zero dependencies** (for most packages; explicitly kept minimal).
- **High performance** (optimized for speed & zero allocations).
- **Opinionated yet flexible** (following standard best practices).

## 📦 Installation

```bash
go get github.com/Jkenyut/nvx-go-helper
```

## ✨ Core Features

### 1. Activity (`/activity`)
Context-based helpers for tracking request metadata. Ideal for structured logging and distributed tracing.

```go
import "github.com/Jkenyut/nvx-go-helper/activity"

// Inject into Context (usually in Middleware)
ctx := activity.WithRequestID(context.Background(), "req-123")
ctx = activity.WithUserID(ctx, "user-001")

// Extract anywhere in your application
userID, ok := activity.GetUserID(ctx)

// Get all fields as map for structured loggers (Zap, Logrus, etc.)
fields := activity.GetAllFieldsFromContext(ctx)
// map[request_id:"req-123" user_id:"user-001"]
```

### 2. Cryptoutil (`/cryptoutil`)
Unified, production-ready package for all things crypto: Password Hashing (Argon2), AES-GCM, ECC (ECIES), UUIDs, and Random string generators.

**Password Hashing (Argon2id)**
Industry-standard password hashing using the PHC string format (single column storage). Includes built-in mitigation against timing attacks.
```go
import "github.com/Jkenyut/nvx-go-helper/cryptoutil"

// 1. Hash password (returns $argon2id$v=19$m=32768,t=1,p=2$salt$hash)
encodedHash, _ := cryptoutil.HashPassword("MySecretPassword123") 

// 2. Verify password (automatically extracts parameters & salt)
isValid, _ := cryptoutil.VerifyPassword("MySecretPassword123", encodedHash)
```

**Asymmetric / Hybrid Encryption (ECC + AES-GCM)**
Perfect for Frontend-to-Backend secure communication. Frontend encrypts with Public Key, Backend decrypts with Private Key using ECIES.
```go
// Generate or load keys (Supports PEM format)
privKey, pubKey, _ := cryptoutil.GenerateECCKeyPair()

// Encrypt (usually on client/frontend)
ciphertext, _ := cryptoutil.EncryptECC(pubKey, []byte("super-secret-data"))

// Decrypt (on backend)
plaintext, _ := cryptoutil.DecryptECC(privKey, ciphertext)
```

**Symmetric Encryption (AES-256-GCM)**
Ultra-fast, secure encryption utilizing `bytedance/sonic` for JSON map serialization.
```go
// 1. Generate a secure hex key for your .env file
hexKey, _ := cryptoutil.GenerateAESKey() // "a1b2c3..."

// 2. Initialize Encryptor
enc, _ := cryptoutil.NewAESGCMFromHex(hexKey)

// 3. Encrypt any struct/map securely
token, _ := enc.Encrypt(map[string]string{"user_id": "123"})
```

**UUID (V4 & V7)**
```go
token := cryptoutil.V4() // Random
id := cryptoutil.V7()    // Time-ordered (Recommended for DB Primary Keys)
```

**Random Strings & Keys (Turbo Optimized)**
Uses bulk-read OS syscalls (`/dev/urandom`) with rejection sampling for extreme speed without compromising entropy.
```go
otp := cryptoutil.Numbers(6) // "123456"
ref := cryptoutil.String(8)  // "A1B2C3D4"
```

### 3. Env (`/env`)
Safe environment variable access with default values.
```go
import "github.com/Jkenyut/nvx-go-helper/env"

port := env.GetInt("PORT", 8080)
dbHost := env.GetString("DB_HOST", "localhost")
```

### 4. Format (`/format`)
Helpers for string manipulation, formatting, and date boundary calculations.
```go
import "github.com/Jkenyut/nvx-go-helper/format"

fmt.Println(format.Rupiah(150000))              // "150.000,00"
fmt.Println(format.BRINorek("123456789012345")) // "1234-56-789012-34-5"
fmt.Println(format.ToSafeString("User Name!"))  // "User_Name_"

// Date Boundary Helpers
now := format.Now()
startOfDay := format.StartOfDay(now) // 00:00:00
endOfMonth := format.EndOfMonth(now) // 23:59:59.999999999 of the last day
```

### 5. Pointer (`/pointer`)
Generic helpers to create pointers from literals easily.
```go
import "github.com/Jkenyut/nvx-go-helper/pointer"

user := User{ IsActive: pointer.Of(true) }
```

### 6. Validator (`/validator`)
Singleton wrapper for `go-playground/validator` v10, featuring a thread-safe custom validation registry and user-friendly error message translation.

```go
import (
    "github.com/Jkenyut/nvx-go-helper/validator"
    v10 "github.com/go-playground/validator/v10"
)

// 1. (Optional) Register Custom Tags safely at startup (init/main)
validator.RegisterCustomValidation("is_admin", func(fl v10.FieldLevel) bool {
    return fl.Field().String() == "admin"
}, "Role must be exactly %s")

// 2. Validate Structs
type User struct {
    Email string `validate:"required,email"`
    Role  string `validate:"is_admin"`
}

err := validator.Struct(User{Email: "wrong", Role: "guest"})

// 3. Get Human-Readable Errors for Frontend/Mobile API
fmt.Println(validator.GetErrorFirstMsg(err)) // "email: Invalid email address format"

// Includes built-in Indonesian specific validations!
// e.g. `validate:"nik"`, `validate:"npwp"`, `validate:"phone_id"`
```

### 7. Response (`/response`)
Standardized JSON API response format. Powered by **`bytedance/sonic`** for hyper-fast JSON serialization.

```go
import "github.com/Jkenyut/nvx-go-helper/response"

func CreateUser(c *gin.Context) {
    // Elegant Method Chaining + Hyper-Fast Sonic Marshal
    respBytes := response.Created(c.Request.Context(), "user created", user).JSONMarshal()
    c.Data(201, "application/json", respBytes)
}
```

### 8. Pagination (`/pagination`)
Comprehensive, bidirectional dynamic keyset (cursor) and offset pagination with built-in whitelisted filtering, search, and type-safe generic responses. Safe from "ghost pages" and `OFFSET` performance bottlenecks.

```go
import "github.com/Jkenyut/nvx-go-helper/pagination"

// 1. Traditional Offset with Whitelisted Filtering & Response Wrapper
req := pagination.BindOffsetFilterRequest(r, map[string]string{"status": "users.status"})
pageData := pagination.NewFromInt(req.Page, req.Limit, totalCount)
resp := pagination.NewListResponse(users, pageData)

// 2. Modern Dynamic Keyset Cursor (Auto-decoded CursorValues)
cursorReq := pagination.BindCursorFilterRequest(r, map[string]string{"name": "user_name"})
cursorMeta := pagination.GenerateBidirectionalCursor(users, cursorReq.Limit, cursorReq.Direction, cursorReq.Cursor, extractFn)
cursorResp := pagination.NewCursorListResponse(users, cursorMeta)

// 3. Unified Mode (Auto-detects cursor vs offset in a single endpoint)
unified := pagination.BindUnifiedFilterRequest(r, allowedColumns)
```

### 9. Worker Pool (`/worker`)
Advanced, production-ready generic worker pool for concurrent batch processing.

```go
import "github.com/Jkenyut/nvx-go-helper/worker"

jobs := []worker.Job[string, int]{
    {ID: "task-1", Data: 100},
}

cfg := worker.WorkerPoolConfig{
    NumWorkers:    5,
    PreserveOrder: true,
    OnProgress: func(completed, total int) {
        fmt.Printf("Progress: %d/%d\n", completed, total)
    },
}

workerFunc := func(ctx context.Context, id string, data int) (string, error) {
    return fmt.Sprintf("Result: %d", data*2), nil
}

// Blocks until all jobs complete, captures context causes
results, _ := worker.RunGenericWorkerPool(context.Background(), jobs, workerFunc, nil, cfg)
```

### 10. Request (`/request`)
Helpers for safely extracting data from HTTP requests, including Context, JSON Binding, Query Parameters, and Headers.

```go
import "github.com/Jkenyut/nvx-go-helper/request"

// Bind JSON and Validate instantly
var payload UserRequest
err := request.BindAndValidate(c.Request, &payload)

// Safe Query Extractors (with Fallbacks)
page := request.GetQueryInt(c.Request, "page", 1)
active := request.GetQueryBool(c.Request, "active", true)

// Safe IP and Token extraction
ip := request.GetClientIP(c.Request) // parses X-Forwarded-For securely
token := request.GetBearerToken(c.Request)
```

### 11. Slice & Map Utilities (`/sliceutil`)
Generic-powered utilities for collections (Go 1.18+). Zero reflection, full type safety.

```go
import "github.com/Jkenyut/nvx-go-helper/sliceutil"

users := []User{{ID: 1}, {ID: 2}, {ID: 3}}

// Slices
ids := sliceutil.Map(users, func(u User) int { return u.ID })
chunks := sliceutil.Chunk(ids, 2) // [[1, 2], [3]]

// Maps
userMap := sliceutil.ToMap(users, func(u User) int { return u.ID }) // map[int]User
keys := sliceutil.MapKeys(userMap) // []int{1, 2, 3}
```

### 12. File Utilities (`/fileutil`)
Secure file validation and manipulation. Ideal for handling uploads.

```go
import "github.com/Jkenyut/nvx-go-helper/fileutil"

// Prevent Path Traversal
safeName := fileutil.SanitizeFileName("../../../etc/passwd") // "passwd"

// Validate Magic Bytes (MIME Type) securely
isImage := fileutil.IsSafeImage(fileBytes) // true if PNG/JPEG/GIF/WebP

// Format file size
size := fileutil.FormatFileSize(1048576) // "1.0 MB"
```

### 13. Retry Mechanism (`/retry`)
Generic, options-based retry mechanism for failing operations (DB/Network).

```go
import "github.com/Jkenyut/nvx-go-helper/retry"

err := retry.Do(func() error {
    return db.Ping()
}, 
    retry.WithMaxAttempts(3), 
    retry.WithBackoff(1*time.Second, 2.0), // 1s, 2s, 4s...
    retry.WithContext(ctx),
)
```

## 🤝 Contributing
Pull requests are welcome. For major changes, please open an issue first.

## 📄 License
[Apache 2.0](LICENSE)

