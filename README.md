# NVX Go Helper

**nvx-go-helper** is a collection of **production-grade** utility functions designed to accelerate backend service development in Go (Golang). This library is built according to 2025 enterprise standards.

**Key Design Principles:**
- **Zero dependencies** (wherever possible).
- **High performance** (optimized for speed & zero allocations).
- **Opinionated yet flexible** (following standard best practices).

## 📦 Installation

```bash
go get github.com/Jkenyut/nvx-go-helper
```

## ✨ Core Features

### 1. Crypto (`/crypto`)
Ultra-fast and secure AES-256-GCM encryption. A simple, misuse-resistant wrapper around the standard `crypto/cipher`.

```go
import "github.com/Jkenyut/nvx-go-helper/crypto"

// Init (call once at startup)
enc, err := crypto.NewAESGCM("32-byte-secret-key-must-be-exact!")

// Encrypt Any Struct -> URL specific Base64
token, _ := enc.Encrypt(map[string]string{"user_id": "123"})

// Decrypt
var data map[string]string
err := enc.Decrypt(token, &data)
```

### 2. Format (`/format`)
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

### 3. Response (`/model`)
Standardized JSON API response format (`{ meta, data }`). Automatically handles `request_id` context propagation.

```go
import "github.com/Jkenyut/nvx-go-helper/model"

func CreateUser(c *gin.Context) {
    // ... logic ...
    
    // Returns 201 Created
    c.JSON(201, response.Created(c.Request.Context(), "user created", user))
}

func GetUser(c *gin.Context) {
    // Returns 404 Not Found
    c.JSON(404, response.NotFound(c.Request.Context(), "user not found"))
}
```

### 4. Pagination (`/pagination`)
Robust helper for handling pagination query parameters and generating Link headers (RFC 5988).

```go
import "github.com/Jkenyut/nvx-go-helper/pagination"

// Parse from query param
p := pagination.New("1", "10", 100) // page, limit, total

// Use in DB query
db.Limit(p.Limit).Offset(p.Offset()).Find(&users)

// Generate response info
c.JSON(200, gin.H{
    "data": users,
    "pagination": p,
})
```

## 🤝 Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

[MIT](LICENSE)
