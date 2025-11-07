package engine

import (
	"context"
	"fmt"
	"os/exec"
	"queuectl/internal/store"
	"time"
)

type Worker struct {
	Store *store.Store
	Base  int
	Cap   int
}

func NewWorker(st *store.Store) *Worker {
	return &Worker{Store: st, Base: 2, Cap: 60}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Worker shutting down gracefully...")
			return
		default:
		}

		now := time.Now().UTC()
		job, err := w.Store.ClaimOne(ctx, now)
		if err != nil {
			fmt.Println("Claim error:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if job == nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}

		fmt.Printf("Running job %s: %s\n", job.ID, job.Command)

		cmd := exec.CommandContext(ctx, "bash", "-lc", job.Command)
		err = cmd.Run()

		if err == nil {
			_ = w.Store.Complete(ctx, job.ID, time.Now().UTC())
			fmt.Printf("Job %s completed\n", job.ID)
		} else {
			moved, _ := w.Store.FailRetry(ctx, job, time.Now().UTC(), w.Base, w.Cap, err)
			if moved {
				fmt.Printf("Job %s moved to DLQ\n", job.ID)
			} else {
				fmt.Printf("Job %s failed, retry scheduled\n", job.ID)
			}
		}
	}
}
