# NVX Go Helper

**nvx-go-helper** is a collection of **production-grade** utility functions designed to accelerate backend service development in Go (Golang). This library is built according to 2025 enterprise standards.

**Key Design Principles:**
- **Zero dependencies** (for most packages; explicitly kept minimal).
- **High performance** (optimized for speed & zero allocations).
- **Opinionated yet flexible** (following standard best practices).

## 📦 Installation

```bash
go get github.com/Jkenyut/nvx-go-helper
```

## ✨ Core Features

### 1. Cryptoutil (`/cryptoutil`)
Unified package for all things crypto: AES-GCM, HMAC, SHA, UUIDs, and Random strings.

**AES-256-GCM**
Ultra-fast and secure encryption.

```go
import "github.com/Jkenyut/nvx-go-helper/cryptoutil"

// Init (call once at startup)
enc, err := cryptoutil.NewAESGCM("32-byte-secret-key-must-be-exact!")

// Encrypt Any Struct -> URL safe Base64
token, _ := enc.Encrypt(map[string]string{"user_id": "123"})

// Decrypt
var data map[string]string
err := enc.Decrypt(token, &data)
```

**Signature**
Secure HMAC-SHA256 signature generation.

```go
secret := "my-secret-key"
sig := cryptoutil.Signature(secret, "data1", "data2")
```

**UUID (V4 & V7)**
Battle-tested UUID generator.

```go
// Random (V4)
token := cryptoutil.V4()

// Time-ordered (V7) - Recommended for DB Primary Keys
id := cryptoutil.V7()
```

**Random Strings**
Cryptographically secure random generators.

```go
otp := cryptoutil.Numbers(6) // "123456"
ref := cryptoutil.String(8)  // "A1B2C3D4"
```

### 2. Env (`/env`)
Safe environment variable access with default values.

```go
import "github.com/Jkenyut/nvx-go-helper/env"

port := env.GetInt("PORT", 8080)
dbHost := env.GetString("DB_HOST", "localhost")
debug := env.GetBool("DEBUG", false)
timeout := env.GetDuration("TIMEOUT", 5*time.Second)
```

### 3. Format (`/format`)
Helpers for string manipulation, number formatting, and local banking standards (Indonesia).

```go
import "github.com/Jkenyut/nvx-go-helper/format"

// Format Rupiah
fmt.Println(format.Rupiah(150000)) // "150.000,00"

// Format BRI Account Number
fmt.Println(format.BRINorek("123456789012345")) // "1234-56-789012-34-5"

// Title Case (Smart)
fmt.Println(format.Title("admin-role")) // "Admin-Role"

// Safe String (for filename/key)
fmt.Println(format.ToSafeString("User Name / 123")) // "User_Name___123"
```

### 4. Pointer (`/pointer`)
Generic helpers to easily create pointers from literals (Go 1.18+).

```go
import "github.com/Jkenyut/nvx-go-helper/pointer"

user := User{
    IsActive: pointer.Of(true),
    Age:      pointer.Of(25),
}
```

### 5. Validator (`/validator`)
Singleton wrapper for `go-playground/validator`.

```go
import "github.com/Jkenyut/nvx-go-helper/validator"

type User struct {
    Email string `validate:"required,email"`
}

err := validator.Struct(user)
```

### 6. Response (`/model`)
Standardized JSON API response format (`{ meta, data }`). Automatically handles `request_id` context propagation.

```go
import "github.com/Jkenyut/nvx-go-helper/model"

func CreateUser(c *gin.Context) {
    // Returns 201 Created
    c.JSON(201, response.Created(c.Request.Context(), "user created", user))
}

func GetUser(c *gin.Context) {
    // Returns 404 Not Found
    c.JSON(404, response.NotFound(c.Request.Context(), "user not found"))
}
```

### 7. Pagination (`/pagination`)
Robust helper for handling pagination query parameters.

```go
import "github.com/Jkenyut/nvx-go-helper/pagination"

// Parse from query param
p := pagination.New("1", "10", 100) // page, limit, total

// Use in DB query
db.Limit(p.Limit).Offset(p.Offset()).Find(&users)
```

## 🤝 Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## 📄 License

[MIT](LICENSE)
