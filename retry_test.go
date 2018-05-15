package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

var (
	errTest = errors.New("test error")
)

func TestBackoffBacksOff(t *testing.T) {
	t.Run("r.Run", func(t *testing.T) {
		tries := 0
		start := time.Now()
		var last time.Time
		retrier := NewRetrier(5, 50, 50)
		err := retrier.Run(func() error {
			tries++
			last = time.Now()
			return errTest
		})

		if tries != 5 {
			t.Errorf("expected 5 tries, got %d", tries)
		}
		if err != errTest {
			t.Errorf("err should equal errTest, got: %v", err)
		}
		if last.Sub(start) > 250*time.Millisecond {
			t.Errorf("should have taken less than 250 milliseconds, took %d", last.Sub(start).Nanoseconds()/1000000)
		}
	})
	t.Run("r.RunContext", func(t *testing.T) {
		tries := 0
		start := time.Now()
		var last time.Time
		retrier := NewRetrier(5, 50, 50)
		err := retrier.RunContext(context.Background(), func(ctx context.Context) error {
			tries++
			last = time.Now()
			return errTest
		})

		if tries != 5 {
			t.Errorf("expected 5 tries, got %d", tries)
		}
		if err != errTest {
			t.Errorf("err should equal errTest, got: %v", err)
		}
		if last.Sub(start) > 250*time.Millisecond {
			t.Errorf("should have taken less than 250 milliseconds, took %d", last.Sub(start).Nanoseconds()/1000000)
		}
	})
}

func TestEventualSuccessSucceedsTransparently(t *testing.T) {
	t.Run("r.Run", func(t *testing.T) {
		tries := 0
		retrier := NewRetrier(5, 50, 50)
		err := retrier.Run(func() error {
			tries++
			if tries == 2 {
				return nil
			}
			return errTest
		})
		if tries != 2 {
			t.Errorf("expected 2 tries, got %d", tries)
		}
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
	t.Run("r.RunContext", func(t *testing.T) {
		tries := 0
		retrier := NewRetrier(5, 50, 50)
		err := retrier.RunContext(context.Background(), func(ctx context.Context) error {
			tries++
			if tries == 2 {
				return nil
			}
			return errTest
		})
		if tries != 2 {
			t.Errorf("expected 2 tries, got %d", tries)
		}
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestRunContextExitsEarlyWhenContextCanceled(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	tries := 0
	ctx, canceler := context.WithCancel(context.Background())
	retrier := NewRetrier(5, 50, 50)

	wg.Add(1)
	go func() {
		err = retrier.RunContext(ctx, func(ctx context.Context) error {
			tries++
			return errTest
		})
		wg.Done()
	}()
	time.Sleep(200 * time.Millisecond)
	canceler()
	wg.Wait()

	if tries < 1 {
		t.Errorf("expected at least one retry, got %d", tries)
	}
	if tries >= 100 {
		t.Error("reached MaxTries, but should not have")
	}
	if err != errTest {
		t.Errorf("err should equal errTest, got: %v", err)
	}
}

func TestStopStopsImmediately(t *testing.T) {
	t.Run("r.Run", func(t *testing.T) {
		tries := 0
		retrier := NewRetrier(5, 50, 50)
		err := retrier.Run(func() error {
			tries++
			return Stop(errTest)
		})

		if tries != 1 {
			t.Errorf("expected 1 tries, got %d", tries)
		}
		if err != errTest {
			t.Errorf("err should equal errTest, got: %v", err)
		}
	})
	t.Run("r.RunContext", func(t *testing.T) {
		tries := 0
		retrier := NewRetrier(5, 50, 50)
		err := retrier.RunContext(context.Background(), func(ctx context.Context) error {
			tries++
			return Stop(errTest)
		})

		if tries != 1 {
			t.Errorf("expected 1 tries, got %d", tries)
		}
		if err != errTest {
			t.Errorf("err should equal errTest, got: %v", err)
		}
	})
}

func TestRetrierGetsDefaultsIfLessThanZero(t *testing.T) {
	r := NewRetrier(-1, -1, -1)
	if r.maxTries != DefaultMaxTries {
		t.Errorf("expected maxTries to be %d, got %d", DefaultMaxTries, r.maxTries)
	}
	if r.initialDelay != DefaultInitialDelayMS {
		t.Errorf("expected initialDelay to be %d, got %d", DefaultInitialDelayMS, r.initialDelay)
	}
	if r.maxDelay != DefaultMaxDelayMS {
		t.Errorf("expected maxDelay to be %d, got %d", DefaultMaxDelayMS, r.maxDelay)
	}
}

func ExampleRetrier_Run() {
	retrier := NewRetrier(5, 50, 50)
	err := retrier.Run(func() error {
		resp, err := http.Get("http://golang.org")
		switch {
		case err != nil:
			return err
		case resp.StatusCode == 0 || resp.StatusCode >= 500:
			return fmt.Errorf("Retryable HTTP status: %s", http.StatusText(resp.StatusCode))
		case resp.StatusCode != 200:
			return Stop(fmt.Errorf("Non-retryable HTTP status: %s", http.StatusText(resp.StatusCode)))
		}
		return nil
	})
	fmt.Println(err)
}

func ExampleRetrier_RunContext_output() {
	retrier := NewRetrier(5, 50, 50)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := retrier.RunContext(ctx, func(ctx context.Context) error {
		req, _ := http.NewRequest("GET", "http://golang.org/notfastenough", nil)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("OMG AWFUL CODE %d", resp.StatusCode)
			// or decide not to retry
		}
		return nil
	})
	fmt.Println(err)
	// Output: Get http://golang.org/notfastenough: context deadline exceeded
}
