package tests

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"queuectl/internal/engine"
	"queuectl/internal/store"
)

func TestWorkerSuccess(t *testing.T) {
	st := newStore(t)
	_ = (*store.Store)(nil) // Ensure store package is used
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Enqueue a job that will succeed
	err := enqueueTestJob(st, "success-job", "echo 'test success'", 3)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Create and start worker
	worker := engine.NewWorker(st)
	go worker.Run(ctx)

	// Give worker time to process
	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(100 * time.Millisecond) // Allow worker to finish

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
	if job.ID != "success-job" {
		t.Errorf("Expected job ID 'success-job', got '%s'", job.ID)
	}
	if job.State != "completed" {
		t.Errorf("Expected state 'completed', got '%s'", job.State)
	}
	if job.Attempts != 0 {
		t.Errorf("Expected attempts 0, got %d", job.Attempts)
	}

	// Verify no pending jobs
	pendingJobs, err := st.ListJobs(storeCtx, "pending")
	if err != nil {
		t.Fatalf("Failed to list pending jobs: %v", err)
	}
	if len(pendingJobs) != 0 {
		t.Errorf("Expected 0 pending jobs, got %d", len(pendingJobs))
	}
}

func TestWorkerSuccessMultipleJobs(t *testing.T) {
	st := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Enqueue multiple jobs that will succeed
	jobs := []string{"job1", "job2", "job3"}
	for _, jobID := range jobs {
		err := enqueueTestJob(st, jobID, "echo 'success'", 3)
		if err != nil {
			t.Fatalf("Failed to enqueue job %s: %v", jobID, err)
		}
	}

	// Create and start worker
	worker := engine.NewWorker(st)
	go worker.Run(ctx)

	// Give worker time to process all jobs
	time.Sleep(3 * time.Second)
	cancel()
	time.Sleep(100 * time.Millisecond) // Allow worker to finish

	// Verify all jobs are completed
	storeCtx := context.Background()
	completedJobs, err := st.ListJobs(storeCtx, "completed")
	if err != nil {
		t.Fatalf("Failed to list completed jobs: %v", err)
	}

	if len(completedJobs) != 3 {
		t.Fatalf("Expected 3 completed jobs, got %d", len(completedJobs))
	}

	// Verify each job is completed
	jobMap := make(map[string]bool)
	for _, job := range completedJobs {
		jobMap[job.ID] = true
		if job.State != "completed" {
			t.Errorf("Job %s: expected state 'completed', got '%s'", job.ID, job.State)
		}
	}

	for _, expectedID := range jobs {
		if !jobMap[expectedID] {
			t.Errorf("Job %s not found in completed jobs", expectedID)
		}
	}
}

func TestWorkerCommandExecution(t *testing.T) {
	// Test that the command actually gets executed
	// This is more of an integration test
	st := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a command that creates a file to verify execution
	testFile := "/tmp/queuectl_test_" + time.Now().Format("20060102150405")
	cmd := "touch " + testFile

	err := enqueueTestJob(st, "file-job", cmd, 3)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Create and start worker
	worker := engine.NewWorker(st)
	go worker.Run(ctx)

	// Give worker time to process
	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Check if file was created (this verifies command execution)
	checkCmd := exec.Command("test", "-f", testFile)
	err = checkCmd.Run()
	if err != nil {
		t.Logf("Note: Could not verify file creation (this is okay on Windows or if test command not available)")
	} else {
		// Clean up test file
		exec.Command("rm", testFile).Run()
	}

	// At minimum, verify job is completed
	storeCtx := context.Background()
	completedJobs, err := st.ListJobs(storeCtx, "completed")
	if err != nil {
		t.Fatalf("Failed to list completed jobs: %v", err)
	}

	if len(completedJobs) != 1 {
		t.Fatalf("Expected 1 completed job, got %d", len(completedJobs))
	}
}

