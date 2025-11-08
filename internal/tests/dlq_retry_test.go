package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"queuectl/internal/engine"
	"queuectl/internal/model"
	"queuectl/internal/store"
)

func TestDLQRetryMovesJobBackToQueue(t *testing.T) {
	st := newStore(t)
	_ = (*store.Store)(nil) // Ensure store package is used
	ctx := context.Background()
	now := time.Now().UTC()

	// Enqueue a job and fail it until it moves to DLQ
	err := enqueueTestJob(st, "retry-dlq-job", "echo 'retry test'", 1)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	base := 2
	capSeconds := 60

	// Fail job until it moves to DLQ
	for {
		job, err := st.ClaimOne(ctx, now.Add(1*time.Hour))
		if err != nil {
			t.Fatalf("Failed to claim job: %v", err)
		}
		if job == nil {
			break
		}
		if job.ID != "retry-dlq-job" {
			continue
		}

		now = time.Now().UTC()
		moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
		if err != nil {
			t.Fatalf("Failed to fail/retry job: %v", err)
		}
		if moved {
			break
		}
	}

	// Verify job is in DLQ
	dlqJobs, err := st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}
	if len(dlqJobs) != 1 {
		t.Fatalf("Expected 1 job in DLQ, got %d", len(dlqJobs))
	}

	// Retry the job from DLQ
	err = st.RetryDLQ(ctx, "retry-dlq-job")
	if err != nil {
		t.Fatalf("Failed to retry DLQ job: %v", err)
	}

	// Verify job is removed from DLQ
	dlqJobs, err = st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}
	if len(dlqJobs) != 0 {
		t.Fatalf("Expected 0 jobs in DLQ, got %d", len(dlqJobs))
	}

	// Verify job is back in jobs table with pending state
	job, err := getJob(st, "retry-dlq-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.State != "pending" {
		t.Errorf("Expected state 'pending', got '%s'", job.State)
	}
	if job.Attempts != 0 {
		t.Errorf("Expected attempts 0, got %d", job.Attempts)
	}
	if job.Command != "echo 'retry test'" {
		t.Errorf("Expected command 'echo 'retry test'', got '%s'", job.Command)
	}
	if job.ID != "retry-dlq-job" {
		t.Errorf("Expected ID 'retry-dlq-job', got '%s'", job.ID)
	}
}

func TestDLQRetryJobCanBeProcessedAgain(t *testing.T) {
	st := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Enqueue a job that will succeed and move it to DLQ first
	err := enqueueTestJob(st, "process-again-job", "echo 'success'", 1)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	now := time.Now().UTC()
	base := 2
	capSeconds := 60

	// Fail job until it moves to DLQ
	for {
		job, err := st.ClaimOne(ctx, now.Add(1*time.Hour))
		if err != nil {
			t.Fatalf("Failed to claim job: %v", err)
		}
		if job == nil {
			break
		}
		if job.ID != "process-again-job" {
			continue
		}

		now = time.Now().UTC()
		moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
		if err != nil {
			t.Fatalf("Failed to fail/retry job: %v", err)
		}
		if moved {
			break
		}
	}

	// Verify job is in DLQ
	dlqJobs, err := st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}
	if len(dlqJobs) != 1 {
		t.Fatalf("Expected 1 job in DLQ, got %d", len(dlqJobs))
	}

	// Retry the job from DLQ
	err = st.RetryDLQ(ctx, "process-again-job")
	if err != nil {
		t.Fatalf("Failed to retry DLQ job: %v", err)
	}

	// Now process the job with a worker (it should succeed this time)
	worker := engine.NewWorker(st)
	go worker.Run(ctx)

	// Give worker time to process
	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify job is completed
	storeCtx := context.Background()
	completedJobs, err := st.ListJobs(storeCtx, "completed")
	if err != nil {
		t.Fatalf("Failed to list completed jobs: %v", err)
	}

	if len(completedJobs) != 1 {
		t.Fatalf("Expected 1 completed job, got %d", len(completedJobs))
	}

	job := completedJobs[0]
	if job.ID != "process-again-job" {
		t.Errorf("Expected job ID 'process-again-job', got '%s'", job.ID)
	}
	if job.State != "completed" {
		t.Errorf("Expected state 'completed', got '%s'", job.State)
	}
	if job.Attempts != 0 {
		t.Errorf("Expected attempts 0 (reset), got %d", job.Attempts)
	}
}

func TestDLQRetryMultipleJobs(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Enqueue multiple jobs and move them all to DLQ
	jobIDs := []string{"retry1", "retry2", "retry3"}
	for _, jobID := range jobIDs {
		err := enqueueTestJob(st, jobID, "echo 'test'", 1)
		if err != nil {
			t.Fatalf("Failed to enqueue job %s: %v", jobID, err)
		}
	}

	base := 2
	capSeconds := 60

	// Move all jobs to DLQ
	for _, jobID := range jobIDs {
		for {
			job, err := st.ClaimOne(ctx, now.Add(1*time.Hour))
			if err != nil {
				t.Fatalf("Failed to claim job: %v", err)
			}
			if job == nil {
				break
			}
			if job.ID != jobID {
				continue
			}

			now = time.Now().UTC()
			moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
			if err != nil {
				t.Fatalf("Failed to fail/retry job: %v", err)
			}
			if moved {
				break
			}
		}
	}

	// Verify all jobs are in DLQ
	dlqJobs, err := st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}
	if len(dlqJobs) != 3 {
		t.Fatalf("Expected 3 jobs in DLQ, got %d", len(dlqJobs))
	}

	// Retry all jobs from DLQ
	for _, jobID := range jobIDs {
		err = st.RetryDLQ(ctx, jobID)
		if err != nil {
			t.Fatalf("Failed to retry DLQ job %s: %v", jobID, err)
		}
	}

	// Verify all jobs are removed from DLQ
	dlqJobs, err = st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}
	if len(dlqJobs) != 0 {
		t.Fatalf("Expected 0 jobs in DLQ, got %d", len(dlqJobs))
	}

	// Verify all jobs are back in jobs table with pending state
	pendingJobs, err := st.ListJobs(ctx, "pending")
	if err != nil {
		t.Fatalf("Failed to list pending jobs: %v", err)
	}
	if len(pendingJobs) != 3 {
		t.Fatalf("Expected 3 pending jobs, got %d", len(pendingJobs))
	}

	// Verify each job
	jobMap := make(map[string]*model.Job)
	for i := range pendingJobs {
		jobMap[pendingJobs[i].ID] = &pendingJobs[i]
	}

	for _, expectedID := range jobIDs {
		job, ok := jobMap[expectedID]
		if !ok {
			t.Errorf("Job %s not found in pending jobs", expectedID)
			continue
		}
		if job.State != "pending" {
			t.Errorf("Job %s: expected state 'pending', got '%s'", expectedID, job.State)
		}
		if job.Attempts != 0 {
			t.Errorf("Job %s: expected attempts 0, got %d", expectedID, job.Attempts)
		}
	}
}

func TestDLQRetryPreservesJobCommand(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	originalCommand := "echo 'preserve this command'"
	
	// Enqueue a job with a specific command
	err := enqueueTestJob(st, "preserve-job", originalCommand, 1)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	base := 2
	capSeconds := 60

	// Move job to DLQ
	for {
		job, err := st.ClaimOne(ctx, now.Add(1*time.Hour))
		if err != nil {
			t.Fatalf("Failed to claim job: %v", err)
		}
		if job == nil {
			break
		}
		if job.ID != "preserve-job" {
			continue
		}

		now = time.Now().UTC()
		moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
		if err != nil {
			t.Fatalf("Failed to fail/retry job: %v", err)
		}
		if moved {
			break
		}
	}

	// Retry from DLQ
	err = st.RetryDLQ(ctx, "preserve-job")
	if err != nil {
		t.Fatalf("Failed to retry DLQ job: %v", err)
	}

	// Verify command is preserved
	job, err := getJob(st, "preserve-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if job.Command != originalCommand {
		t.Errorf("Expected command '%s', got '%s'", originalCommand, job.Command)
	}
}

