package engine

import (
	"context"
	"fmt"
	"math/rand"
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
	base := st.MustGetInt("backoff_base", 2)
	cap := st.MustGetInt("backoff_cap_seconds", 60)
	return &Worker{Store: st, Base: base, Cap: cap}
}

func (w *Worker) Run(ctx context.Context) {
	for {

		//checks for stop file
		if ShouldStop() {
			fmt.Println("Worker stopping gracefully...")
			return
		}

		select {
		case <-ctx.Done():
			fmt.Println("Worker shutting down gracefully...")
			return
		default:
		}

		//claim job from queue
		now := time.Now().UTC()
		job, err := w.Store.ClaimOne(ctx, now)
		if err != nil {
			fmt.Println("Claim error:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if job == nil {
			time.Sleep(time.Duration(200+rand.Intn(200)) * time.Millisecond)
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
