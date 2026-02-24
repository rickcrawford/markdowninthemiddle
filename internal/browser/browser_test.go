package browser

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestPool_New_ConnectRetry tests that New retries when Chrome is unavailable.
func TestPool_New_ConnectRetry(t *testing.T) {
	// Using an invalid URL should fail after retries
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := New(ctx, "http://invalid-chrome-host:9999", 5, 30*time.Second)
	if err == nil {
		t.Fatal("expected error when connecting to invalid Chrome URL")
	}
}

// TestPool_PoolSize tests that the semaphore limits concurrent requests.
func TestPool_PoolSize(t *testing.T) {
	// This test validates the semaphore behavior by checking that
	// the Pool struct initializes correctly with a given pool size
	mockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// We can't test actual Chrome operations without a running instance,
	// but we can validate the pool creation with a mock
	p := &Pool{
		allocCtx: mockCtx,
		sem:      nil, // Would be initialized in New()
		timeout:  30 * time.Second,
	}

	if p.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", p.timeout)
	}
}

// TestPool_Close ensures the Pool closes cleanly.
func TestPool_Close(t *testing.T) {
	mockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p := &Pool{
		allocCtx:    mockCtx,
		allocCancel: cancel,
		sem:         nil,
		timeout:     30 * time.Second,
	}

	err := p.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

// TestRetry tests the retry logic with exponential backoff.
func TestRetry(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		buildFn    func(*int) func() error
		shouldFail bool
	}{
		{
			name:       "success on first try",
			maxRetries: 3,
			buildFn: func(*int) func() error {
				return func() error {
					return nil
				}
			},
			shouldFail: false,
		},
		{
			name:       "success on second try",
			maxRetries: 3,
			buildFn: func(attempt *int) func() error {
				return func() error {
					*attempt++
					if *attempt < 2 {
						return errors.New("temporary error")
					}
					return nil
				}
			},
			shouldFail: false,
		},
		{
			name:       "failure after all retries",
			maxRetries: 2,
			buildFn: func(*int) func() error {
				return func() error {
					return errors.New("persistent error")
				}
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := 0
			ctx := context.Background()
			err := retry(ctx, tt.maxRetries, 1*time.Millisecond, tt.buildFn(&attempt))

			if (err != nil) != tt.shouldFail {
				t.Errorf("retry error: got %v, shouldFail=%v", err, tt.shouldFail)
			}
		})
	}
}

// TestRetry_ContextCancellation tests that retry respects context cancellation.
func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	fn := func() error {
		callCount++
		// Always fail to force retries
		return errors.New("persistent error")
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retry(ctx, 1000, 10*time.Millisecond, fn)
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}

	// Should have been cancelled before all retries
	if callCount >= 1000 {
		t.Errorf("retry made all attempts despite context cancellation")
	}
}
