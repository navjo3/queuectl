package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"queuectl/internal/model"
	"queuectl/internal/store"
)

func TestDLQMovementOnMaxRetries(t *testing.T) {
	st := newStore(t)
	_ = (*store.Store)(nil) // Ensure store package is used
	ctx := context.Background()
	now := time.Now().UTC()

	// Enqueue a job with max_retries = 2
	// This means: initial attempts=0, after 1st failure attempts=1 (retry), after 2nd failure attempts=2 (moves to DLQ)
	err := enqueueTestJob(st, "dlq-job", "false", 2)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	base := 2
	capSeconds := 60

	// First failure: attempts goes from 0 to 1, should retry (not move to DLQ)
	job, err := st.ClaimOne(ctx, now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job == nil {
		t.Fatal("Expected to claim job for first attempt")
	}

	now = time.Now().UTC()
	moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("first error"))
	if err != nil {
		t.Fatalf("Failed to fail/retry job: %v", err)
	}
	if moved {
		t.Error("Expected job not to be moved to DLQ after first failure")
	}

	// Verify job was retried with attempts = 1
	updatedJob, err := getJob(st, "dlq-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if updatedJob.Attempts != 1 {
		t.Errorf("Expected attempts 1 after first failure, got %d", updatedJob.Attempts)
	}

	// Second failure: attempts goes from 1 to 2, which equals max_retries (2), should move to DLQ
	job, err = st.ClaimOne(ctx, now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job == nil {
		t.Fatal("Expected to claim job for second attempt")
	}

	if job.Attempts != 1 {
		t.Errorf("Expected attempts 1 before second failure, got %d", job.Attempts)
	}

	now = time.Now().UTC()
	moved, err = st.FailRetry(ctx, job, now, base, capSeconds, errors.New("final error"))
	if err != nil {
		t.Fatalf("Failed to fail/retry job: %v", err)
	}

	if !moved {
		t.Error("Expected job to be moved to DLQ after second failure")
	}

	// Verify job is removed from jobs table
	_, err = getJob(st, "dlq-job")
	if err == nil {
		t.Error("Expected job to be removed from jobs table")
	}

	// Verify job is in DLQ
	dlqJobs, err := st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}

	if len(dlqJobs) != 1 {
		t.Fatalf("Expected 1 job in DLQ, got %d", len(dlqJobs))
	}

	dlqJob := dlqJobs[0]
	if dlqJob.ID != "dlq-job" {
		t.Errorf("Expected DLQ job ID 'dlq-job', got '%s'", dlqJob.ID)
	}
	if dlqJob.Command != "false" {
		t.Errorf("Expected command 'false', got '%s'", dlqJob.Command)
	}
	if dlqJob.State != "dead" {
		t.Errorf("Expected state 'dead', got '%s'", dlqJob.State)
	}
	if dlqJob.Attempts != 2 {
		t.Errorf("Expected attempts 2 (equals max_retries), got %d", dlqJob.Attempts)
	}
	if dlqJob.MaxRetries != 2 {
		t.Errorf("Expected max_retries 2, got %d", dlqJob.MaxRetries)
	}
}

func TestDLQMultipleJobs(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	base := 2
	capSeconds := 60

	// Enqueue multiple jobs with different max_retries
	jobs := []struct {
		id         string
		maxRetries int
	}{
		{"job1", 1},
		{"job2", 2},
		{"job3", 1},
	}

	for _, j := range jobs {
		err := enqueueTestJob(st, j.id, "false", j.maxRetries)
		if err != nil {
			t.Fatalf("Failed to enqueue job %s: %v", j.id, err)
		}
	}

	now := time.Now().UTC()

	// Fail each job until it moves to DLQ
	for _, j := range jobs {
		for {
			job, err := st.ClaimOne(ctx, now.Add(1*time.Hour))
			if err != nil {
				t.Fatalf("Failed to claim job %s: %v", j.id, err)
			}
			if job == nil {
				// Job might have been moved to DLQ, check
				break
			}
			if job.ID != j.id {
				// Wrong job, continue
				continue
			}

			now = time.Now().UTC()
			moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
			if err != nil {
				t.Fatalf("Failed to fail/retry job %s: %v", j.id, err)
			}

			if moved {
				// Job moved to DLQ, done with this job
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

	// Verify each job
	jobMap := make(map[string]*model.Job)
	for i := range dlqJobs {
		jobMap[dlqJobs[i].ID] = &dlqJobs[i]
	}

	for _, expected := range jobs {
		job, ok := jobMap[expected.id]
		if !ok {
			t.Errorf("Job %s not found in DLQ", expected.id)
			continue
		}
		if job.State != "dead" {
			t.Errorf("Job %s: expected state 'dead', got '%s'", expected.id, job.State)
		}
		// When a job moves to DLQ, attempts equals max_retries (newAttempts = oldAttempts + 1, and newAttempts >= max_retries)
		if job.Attempts != expected.maxRetries {
			t.Errorf("Job %s: expected attempts == %d (equals max_retries), got %d", expected.id, expected.maxRetries, job.Attempts)
		}
	}

	// Verify no jobs remain in jobs table
	allJobs, err := st.ListJobs(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(allJobs) != 0 {
		t.Errorf("Expected 0 jobs in jobs table, got %d", len(allJobs))
	}
}

func TestDLQPreservesJobDetails(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Enqueue a job
	err := enqueueTestJob(st, "details-job", "echo 'test command'", 1)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Get original job
	originalJob, err := getJob(st, "details-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
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
		if job.ID != "details-job" {
			continue
		}

		now = time.Now().UTC()
		moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("detailed error message"))
		if err != nil {
			t.Fatalf("Failed to fail/retry job: %v", err)
		}
		if moved {
			break
		}
	}

	// Verify DLQ job has correct details
	dlqJobs, err := st.ListDLQ(ctx)
	if err != nil {
		t.Fatalf("Failed to list DLQ jobs: %v", err)
	}

	if len(dlqJobs) != 1 {
		t.Fatalf("Expected 1 job in DLQ, got %d", len(dlqJobs))
	}

	dlqJob := dlqJobs[0]
	if dlqJob.ID != originalJob.ID {
		t.Errorf("Expected ID %s, got %s", originalJob.ID, dlqJob.ID)
	}
	if dlqJob.Command != originalJob.Command {
		t.Errorf("Expected command %s, got %s", originalJob.Command, dlqJob.Command)
	}
	if dlqJob.MaxRetries != originalJob.MaxRetries {
		t.Errorf("Expected max_retries %d, got %d", originalJob.MaxRetries, dlqJob.MaxRetries)
	}

	// Verify error message is stored (check DLQ table directly)
	var lastError string
	err = st.DB.QueryRowContext(ctx, `SELECT last_error FROM dlq WHERE id=?`, "details-job").Scan(&lastError)
	if err != nil {
		t.Fatalf("Failed to get last_error: %v", err)
	}
	if lastError != "detailed error message" {
		t.Errorf("Expected last_error 'detailed error message', got '%s'", lastError)
	}
}

