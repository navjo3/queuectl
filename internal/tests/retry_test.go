package tests

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"queuectl/internal/model"
	"queuectl/internal/store"
)

func TestRetryIncrementsAttempts(t *testing.T) {
	st := newStore(t)
	_ = (*store.Store)(nil) // Ensure store package is used
	_ = (*model.Job)(nil)   // Ensure model package is used
	ctx := context.Background()
	now := time.Now().UTC()

	// Enqueue a job
	err := enqueueTestJob(st, "retry-job", "false", 3) // false command will fail
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Claim the job
	job, err := st.ClaimOne(ctx, now)
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job == nil {
		t.Fatal("Expected to claim a job")
	}

	// Simulate failure and retry
	moved, err := st.FailRetry(ctx, job, now, 2, 60, errors.New("test error"))
	if err != nil {
		t.Fatalf("Failed to fail/retry job: %v", err)
	}
	if moved {
		t.Error("Expected job not to be moved to DLQ yet")
	}

	// Verify job was rescheduled with attempts incremented
	updatedJob, err := getJob(st, "retry-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if updatedJob.Attempts != 1 {
		t.Errorf("Expected attempts 1, got %d", updatedJob.Attempts)
	}
	if updatedJob.State != "pending" {
		t.Errorf("Expected state 'pending', got '%s'", updatedJob.State)
	}

	// Verify available_at was set to future (exponential backoff)
	expectedDelay := time.Duration(math.Pow(2, float64(updatedJob.Attempts))) * time.Second
	expectedAvailableAt := now.Add(expectedDelay)
	
	// Allow some tolerance for timing
	if updatedJob.AvailableAt.Before(now) {
		t.Errorf("Expected available_at to be in the future, got %v", updatedJob.AvailableAt)
	}
	
	// Check that available_at is approximately correct (within 1 second)
	diff := updatedJob.AvailableAt.Sub(expectedAvailableAt)
	if diff < -1*time.Second || diff > 1*time.Second {
		t.Errorf("Expected available_at to be approximately %v, got %v (diff: %v)", 
			expectedAvailableAt, updatedJob.AvailableAt, diff)
	}
}

func TestRetryExponentialBackoff(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	base := 2
	capSeconds := 60

	// Enqueue a job
	err := enqueueTestJob(st, "backoff-job", "false", 5)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	job, err := getJob(st, "backoff-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	now := time.Now().UTC()
	attempts := []int{1, 2, 3}
	expectedDelays := []time.Duration{
		time.Duration(math.Pow(float64(base), 1)) * time.Second, // 2^1 = 2s
		time.Duration(math.Pow(float64(base), 2)) * time.Second, // 2^2 = 4s
		time.Duration(math.Pow(float64(base), 3)) * time.Second, // 2^3 = 8s
	}

	for i, attempt := range attempts {
		// Claim job
		job, err = st.ClaimOne(ctx, now.Add(1*time.Hour)) // Use future time to claim
		if err != nil {
			t.Fatalf("Failed to claim job: %v", err)
		}
		if job == nil {
			t.Fatalf("Expected to claim job on attempt %d", attempt)
		}

		// Fail and retry
		now = time.Now().UTC()
		moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
		if err != nil {
			t.Fatalf("Failed to fail/retry job: %v", err)
		}
		if moved {
			t.Fatalf("Expected job not to be moved to DLQ on attempt %d", attempt)
		}

		// Verify attempts and backoff
		updatedJob, err := getJob(st, "backoff-job")
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}

		if updatedJob.Attempts != attempt {
			t.Errorf("Attempt %d: expected attempts %d, got %d", i+1, attempt, updatedJob.Attempts)
		}

		expectedDelay := expectedDelays[i]
		expectedAvailableAt := now.Add(expectedDelay)
		
		// Check available_at is approximately correct (within 1 second)
		diff := updatedJob.AvailableAt.Sub(expectedAvailableAt)
		if diff < -1*time.Second || diff > 1*time.Second {
			t.Errorf("Attempt %d: expected available_at ~%v, got %v (diff: %v, expected delay: %v)",
				i+1, expectedAvailableAt, updatedJob.AvailableAt, diff, expectedDelay)
		}

		// Update now for next iteration (simulate waiting)
		now = updatedJob.AvailableAt
	}
}

func TestRetryBackoffCap(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	base := 2
	capSeconds := 10 // Cap at 10 seconds

	// Enqueue a job with enough retries to exceed the cap
	err := enqueueTestJob(st, "cap-job", "false", 10)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	job, err := getJob(st, "cap-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	now := time.Now().UTC()
	
	// Fail multiple times to exceed the cap
	// 2^4 = 16 seconds > 10 second cap
	for attempt := 1; attempt <= 4; attempt++ {
		// Claim job (use future time to ensure it's available)
		job, err = st.ClaimOne(ctx, now.Add(1*time.Hour))
		if err != nil {
			t.Fatalf("Failed to claim job: %v", err)
		}
		if job == nil {
			t.Fatalf("Expected to claim job on attempt %d", attempt)
		}

		now = time.Now().UTC()
		moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
		if err != nil {
			t.Fatalf("Failed to fail/retry job: %v", err)
		}
		if moved {
			t.Fatalf("Expected job not to be moved to DLQ on attempt %d", attempt)
		}

		updatedJob, err := getJob(st, "cap-job")
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}

		// Calculate expected delay (capped)
		uncappedDelay := time.Duration(math.Pow(float64(base), float64(attempt))) * time.Second
		expectedDelay := uncappedDelay
		capDur := time.Duration(capSeconds) * time.Second
		if expectedDelay > capDur {
			expectedDelay = capDur
		}

		expectedAvailableAt := now.Add(expectedDelay)
		
		// Verify delay is capped
		diff := updatedJob.AvailableAt.Sub(expectedAvailableAt)
		if diff < -1*time.Second || diff > 1*time.Second {
			t.Errorf("Attempt %d: expected available_at ~%v (capped at %v), got %v (diff: %v)",
				attempt, expectedAvailableAt, capDur, updatedJob.AvailableAt, diff)
		}

		if updatedJob.AvailableAt.Sub(now) > capDur+1*time.Second {
			t.Errorf("Attempt %d: delay %v exceeds cap %v",
				attempt, updatedJob.AvailableAt.Sub(now), capDur)
		}

		now = updatedJob.AvailableAt
	}
}

func TestRetryJobNotClaimableUntilAvailableAt(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Enqueue a job
	err := enqueueTestJob(st, "delay-job", "false", 3)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Claim and fail it
	job, err := st.ClaimOne(ctx, now)
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}

	base := 2
	capSeconds := 60
	moved, err := st.FailRetry(ctx, job, now, base, capSeconds, errors.New("test error"))
	if err != nil {
		t.Fatalf("Failed to fail/retry job: %v", err)
	}
	if moved {
		t.Fatal("Expected job not to be moved to DLQ")
	}

	// Verify job is not claimable until available_at
	updatedJob, err := getJob(st, "delay-job")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	// Try to claim before available_at
	job2, err := st.ClaimOne(ctx, now)
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job2 != nil {
		t.Error("Expected job not to be claimable before available_at")
	}

	// Try to claim after available_at
	job3, err := st.ClaimOne(ctx, updatedJob.AvailableAt.Add(1*time.Second))
	if err != nil {
		t.Fatalf("Failed to claim job: %v", err)
	}
	if job3 == nil {
		t.Error("Expected job to be claimable after available_at")
	} else if job3.ID != "delay-job" {
		t.Errorf("Expected job ID 'delay-job', got '%s'", job3.ID)
	}
}

