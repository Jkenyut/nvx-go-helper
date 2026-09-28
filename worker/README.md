# Worker Pool (`/worker`)

Generic concurrent worker pool with context cancellation, order preservation, and progress tracking.

## ⚙️ Worker Pool Lifecycle (Activity Diagram)

```mermaid
flowchart TD
    Start([RunGenericWorkerPool]) --> InitQueue[Initialize Jobs Channel & Results Slice]
    InitQueue --> SpawnWorkers[Spawn N Worker Goroutines with WaitGroup]
    
    subgraph Workers ["Worker Execution Loop"]
        FetchJob{Job in Channel?}
        FetchJob -- Yes --> ExecJob[Execute WorkerFunc ctx, job.ID, job.Data]
        ExecJob --> SuccessCheck{Error?}
        SuccessCheck -- No Error --> SaveRes[Save Result to Preserved Index]
        SuccessCheck -- Err & StopOnError --> TriggerCancel[Cancel Pool Context]
        SaveRes --> Progress[Trigger OnProgress Callback]
        Progress --> FetchJob
        FetchJob -- Channel Closed / Context Done --> WorkerDone([Worker Goroutine Exits])
    end

    SpawnWorkers --> FeedJobs[Feed Jobs Slice into Channel in Background]
    FeedJobs --> CloseJobs[Close Jobs Channel when Feed Complete]
    CloseJobs --> WaitWorkers[Wait for All Workers wg.Wait]
    WaitWorkers --> ReturnRes([Return Filtered Results Slice & First Error])
```

---

## 📖 Quickstart

```go
import (
	"context"
	"fmt"
	"github.com/Jkenyut/nvx-go-helper/worker"
)

jobs := []worker.Job[string, int]{
	{ID: "job-1", Data: 10},
	{ID: "job-2", Data: 20},
}

cfg := worker.PoolConfig{
	NumWorkers:    4,
	PreserveOrder: true,
	OnProgress: func(completed, total int) {
		fmt.Printf("Completed %d of %d\n", completed, total)
	},
}

workerFn := func(ctx context.Context, id string, data int) (string, error) {
	return fmt.Sprintf("result-%d", data), nil
}

results, err := worker.RunGenericWorkerPool(context.Background(), jobs, workerFn, nil, cfg)
```
