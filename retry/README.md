# Retry Helper (`/retry`)

The retry package executes functions with configurable exponential backoff, randomized jitter, and context cancellation.

## 1. Retry Execution Logic

The diagram below shows the retry evaluation loop and timer hygiene.

```mermaid
flowchart TD
    Start(["Call retry.Do or DoWithData"]) --> InitConfig["Configure Options: attempts, delay, jitter, ctx"]
    InitConfig --> AttemptLoop["Execute Function: attempt = 1..N"]

    AttemptLoop --> CheckResult{"Function Result"}
    CheckResult -- Nil Error / Success --> ReturnSuccess(["Return Success and Data"])
    CheckResult -- Error Returned --> EvalRetryIf{"Matches retryIf Condition?"}

    EvalRetryIf -- No (Permanent Error) --> ReturnErr(["Return Error Immediately"])
    EvalRetryIf -- Yes (Transient Error) --> CheckAttempt{"Attempt < maxAttempts?"}

    CheckAttempt -- No (Exhausted) --> ReturnMaxErr(["Return ErrMaxRetriesReached"])
    CheckAttempt -- Yes --> CalcDelay["Calculate Delay with Backoff and Jitter"]

    CalcDelay --> StartTimer["Start Timer (time.NewTimer)"]
    StartTimer --> WaitSelect{"Wait for ctx.Done or timer.C"}

    WaitSelect -- ctx.Done Fired --> StopTimerLeak["Stop Timer and Drain Channel"]
    StopTimerLeak --> ReturnCtxErr(["Return ctx.Err()"])

    WaitSelect -- timer.C Fired --> LoopNext["Increment Attempt Counter"]
    LoopNext --> AttemptLoop
```

### Procedural Steps (ASD-STE100)
1. Pass the target function and execution options to `retry.Do` or `retry.DoWithData`.
2. Configure the maximum retry attempts, backoff factor, and context.
3. The helper executes the function.
4. If the function succeeds, the helper returns the result immediately.
5. If an error occurs, the helper evaluates the `RetryIf` condition.
6. If the error is permanent, the helper returns the error without retrying.
7. If the error is transient, the helper calculates the backoff delay.
8. The helper starts a timer and waits for timer completion or context cancellation.
9. If the context cancels, the helper stops the timer and returns the context error.
10. The helper repeats the loop until the function succeeds or attempts expire.

---

## 2. Usage Examples

### 2.1 Simple Retry

```go
import (
	"context"
	"time"

	"github.com/Jkenyut/nvx-go-helper/retry"
)

err := retry.Do(func() error {
	return callExternalPaymentGateway()
},
	retry.WithMaxAttempts(4),
	retry.WithBackoff(100*time.Millisecond, 2.0),
	retry.WithJitter(0.2),
	retry.WithContext(ctx),
)
```

### 2.2 Retry with Return Value and Condition Filter

```go
user, err := retry.DoWithData(func() (*User, error) {
	return fetchUserFromRPC(userID)
},
	retry.WithMaxAttempts(3),
	retry.WithRetryIf(func(err error) bool {
		// Retry transient network errors only
		return isTemporaryNetworkError(err)
	}),
)
```
