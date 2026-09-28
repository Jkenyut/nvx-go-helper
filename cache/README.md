# Cache Helper (`/cache`)

A thread-safe, high-performance in-memory generic cache (`Cache[K, V]`) with automatic TTL expiration, leak-free background sweeping, and built-in **Singleflight** protection against cache stampedes.

## 🌟 Highlights

- **Type-Safe Generics**: Works with any comparable key and any value type (`Cache[K comparable, V any]`).
- **Zero Cache Stampede**: `GetOrCompute` utilizes duplicate function call suppression (singleflight pattern) so that 100 concurrent requests for the same missing key trigger exactly 1 upstream calculation/query.
- **Leak-Free Timer Hygiene**: Background expiration sweeps run via graceful ticker loops that terminate cleanly upon `Close()`.
- **Flexible Expiration**: Per-key TTL durations. Setting `ttl <= 0` retains items indefinitely.

## 📦 Usage Examples

### 1. Basic In-Memory Caching

```go
import (
    "time"
    "github.com/Jkenyut/nvx-go-helper/cache"
)

// Create a cache with background sweeper every 30 seconds
c := cache.New[string, UserProfile](cache.WithCleanupInterval(30 * time.Second))
defer c.Close()

// Set key with 5 minutes TTL
c.Set("user:101", profile, 5*time.Minute)

// Retrieve key
if profile, ok := c.Get("user:101"); ok {
    // Cache hit
}

// Delete or Clear
c.Delete("user:101")
c.Clear()
```

### 2. Singleflight Computation (`GetOrCompute`)

Prevents the "thundering herd" problem where multiple goroutines concurrently miss the cache and overwhelm your database:

```go
ctx := context.Background()

profile, err := c.GetOrCompute(ctx, "user:101", 10*time.Minute, func(ctx context.Context) (UserProfile, error) {
    // This expensive database query is executed ONLY ONCE even if 100 goroutines 
    // request "user:101" at the exact same millisecond.
    return db.FindUserByID(ctx, 101)
})
if err != nil {
    // Handle error (failed computation is not stored in cache)
    return err
}
```
