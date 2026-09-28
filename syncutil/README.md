# SyncUtil Helper (`/syncutil`)

Synchronization and concurrency primitives designed for high-concurrency Go services.

## 🌟 Highlights

- **`KeyedMutex[K]`**: Mutual exclusion locks keyed by identifier (e.g., `user_id`, `account_id`, `order_id`). Operations for different keys proceed concurrently in parallel, while operations for the same key are guaranteed to execute strictly sequentially. Includes automatic reference-counted memory cleanup to prevent memory leaks when keys are released.
- **`ForEach[T]`**: Concurrent slice iteration with a bounded worker pool / semaphore, early context cancellation, and fail-fast error propagation.
- **`Map[T, R]`**: Concurrent generic slice mapping that strictly preserves the original order of items.

## 📦 Usage Examples

### 1. `KeyedMutex` (Fine-Grained Per-Entity Locking)

Avoid lock contention caused by a global mutex when protecting account balances or user updates:

```go
import "github.com/Jkenyut/nvx-go-helper/syncutil"

var userLocks = syncutil.NewKeyedMutex[string]()

func UpdateUserBalance(userID string, amount float64) error {
    // Acquire lock specific to this userID only
    // Other users are NOT blocked!
    unlock := userLocks.Lock(userID)
    defer unlock()

    // Critical section for userID...
    return nil
}
```

### 2. `ForEach` (Bounded Parallel Execution)

Process batches of jobs without exhausting system resources or remote rate limits:

```go
items := []string{"job1", "job2", "job3", "job4", "job5"}

// Run with max 3 concurrent workers
err := syncutil.ForEach(ctx, items, 3, func(ctx context.Context, item string) error {
    return processRemoteWebhook(ctx, item)
})
if err != nil {
    // Fails fast if any item returns error or if context is cancelled
    return err
}
```

### 3. `Map` (Parallel Mapping with Order Preservation)

Transform data concurrently while guaranteeing output elements align with input indices:

```go
userIDs := []int{101, 102, 103, 104, 105}

// Fetch user profiles in parallel (up to 4 goroutines)
profiles, err := syncutil.Map(ctx, userIDs, 4, func(ctx context.Context, id int) (UserProfile, error) {
    return fetchProfileFromDB(ctx, id)
})
if err != nil {
    return err
}
// profiles[0] is guaranteed to correspond to userIDs[0], etc.
```
