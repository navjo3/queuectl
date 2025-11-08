package tests

import (
	"context"
	"testing"
	"time"

	"queuectl/internal/model"
	"queuectl/internal/store"
)

func TestEnqueueJob(t *testing.T) {
	st := newStore(t)
	_ = (*store.Store)(nil) // Ensure store package is used
	ctx := context.Background()

	// Test enqueueing a job
	err := enqueueTestJob(st, "job1", "echo hello", 3)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Verify job was inserted
	jobs, err := st.ListJobs(ctx, "pending")
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.ID != "job1" {
		t.Errorf("Expected job ID 'job1', got '%s'", job.ID)
	}
	if job.Command != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%s'", job.Command)
	}
	if job.State != "pending" {
		t.Errorf("Expected state 'pending', got '%s'", job.State)
	}
	if job.Attempts != 0 {
		t.Errorf("Expected attempts 0, got %d", job.Attempts)
	}
	if job.MaxRetries != 3 {
		t.Errorf("Expected max_retries 3, got %d", job.MaxRetries)
	}
	if job.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if job.AvailableAt.IsZero() {
		t.Error("Expected AvailableAt to be set")
	}
}

func TestEnqueueMultipleJobs(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	// Enqueue multiple jobs
	jobs := []struct {
		id      string
		command string
		retries int
	}{
		{"job1", "echo hello", 3},
		{"job2", "echo world", 5},
		{"job3", "echo test", 1},
	}

	for _, j := range jobs {
		err := enqueueTestJob(st, j.id, j.command, j.retries)
		if err != nil {
			t.Fatalf("Failed to enqueue job %s: %v", j.id, err)
		}
	}

	// Verify all jobs were inserted
	pendingJobs, err := st.ListJobs(ctx, "pending")
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(pendingJobs) != 3 {
		t.Fatalf("Expected 3 jobs, got %d", len(pendingJobs))
	}

	// Verify job details
	jobMap := make(map[string]*model.Job)
	for i := range pendingJobs {
		jobMap[pendingJobs[i].ID] = &pendingJobs[i]
	}

	for _, expected := range jobs {
		job, ok := jobMap[expected.id]
		if !ok {
			t.Errorf("Job %s not found", expected.id)
			continue
		}
		if job.Command != expected.command {
			t.Errorf("Job %s: expected command '%s', got '%s'", expected.id, expected.command, job.Command)
		}
		if job.MaxRetries != expected.retries {
			t.Errorf("Job %s: expected max_retries %d, got %d", expected.id, expected.retries, job.MaxRetries)
		}
	}
}

func TestEnqueueJobWithFutureAvailableAt(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	future := now.Add(10 * time.Minute)

	// Manually insert a job with future available_at
	_, err := st.DB.ExecContext(ctx, `
		INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, available_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "future-job", "echo future", "pending", 0, 3,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		future.Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("Failed to insert job: %v", err)
	}

	// Job should not be claimable until available_at
	job, err := st.ClaimOne(ctx, now)
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job != nil {
		t.Error("Expected no job to be claimable before available_at")
	}

	// Job should be claimable after available_at
	job, err = st.ClaimOne(ctx, future.Add(1*time.Second))
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job == nil {
		t.Error("Expected job to be claimable after available_at")
	} else if job.ID != "future-job" {
		t.Errorf("Expected job ID 'future-job', got '%s'", job.ID)
	}
}

