# SyncUtil Helper (`/syncutil`)

The syncutil package provides concurrency primitives for fine-grained per-key locking.
For concurrent worker pools and parallel task pipelines, see `/worker`.

## 1. KeyedMutex Architecture

The `KeyedMutex[K]` primitive serializes operations for matching keys while executing different keys concurrently in parallel.
It tracks lock usage with an internal reference counter and removes empty entries to prevent memory leaks.

```mermaid
flowchart TD
    LockReq(["Call km.Lock(key)"]) --> InspectMap["Acquire Global Mutex (km.mu.Lock)"]
    InspectMap --> CheckEntry{"Entry Exists for Key?"}

    CheckEntry -- No --> MakeEntry["Make sync.Mutex with refCount = 1"]
    MakeEntry --> StoreEntry["Store Entry in Map"]
    CheckEntry -- Yes --> IncRef["Increment refCount += 1"]

    StoreEntry & IncRef --> UnlockGlobal["Release Global Mutex (km.mu.Unlock)"]
    UnlockGlobal --> LockKey["Acquire Entry Mutex (entry.mu.Lock)"]
    LockKey --> ReturnUnlockFn(["Return Unlock Function to Caller"])

    subgraph UnlockFlow ["Unlock Callback Execution"]
        ExecUnlock(["Call Returned unlock()"]) --> ReleaseKey["Release Entry Mutex (entry.mu.Unlock)"]
        ReleaseKey --> LockGlobal2["Acquire Global Mutex (km.mu.Lock)"]
        LockGlobal2 --> DecRef["Decrement refCount -= 1"]
        DecRef --> CheckZero{"refCount == 0?"}
        CheckZero -- Yes --> DeleteKey["Delete Key from Map"]
        CheckZero -- No --> RetainKey["Retain Key in Map"]
        DeleteKey & RetainKey --> UnlockGlobal2["Release Global Mutex (km.mu.Unlock)"]
        UnlockGlobal2 --> FinishUnlock(["Unlock Complete"])
    end
```

### Procedural Steps (ASD-STE100)
1. Initialize the keyed mutex with `syncutil.NewKeyedMutex[K]()`.
2. Call `Lock(key)` to acquire a mutual exclusion lock for the specified key.
3. Defer the returned unlock function immediately after acquiring the lock.
4. Execute the critical section for that entity.
5. The unlock function decrements the reference counter and deletes unused map entries.

---

## 2. Usage Examples

### 2.1 Per-Entity Locking (`KeyedMutex`)

```go
import "github.com/Jkenyut/nvx-go-helper/syncutil"

var userLocks = syncutil.NewKeyedMutex[string]()

func UpdateUserBalance(userID string, amount float64) error {
	// Acquire lock for this userID only
	// Other user operations proceed in parallel
	unlock := userLocks.Lock(userID)
	defer unlock()

	// Execute critical balance update
	return nil
}
```
