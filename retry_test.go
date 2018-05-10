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
	tries := 0
	start := time.Now()
	var last time.Time
	err := Do(ConstantBackoff(5, 50*time.Millisecond), func() error {
		tries++
		last = time.Now()
		return errTest
	})

	if tries != 4 {
		t.Errorf("expected 4 tries, got %d", tries)
	}
	if err != errTest {
		t.Errorf("err should equal errTest, got: %v", err)
	}
	if last.Sub(start) > 200*time.Millisecond {
		t.Errorf("should have taken less than 200 milliseconds, took %d", last.Sub(start).Nanoseconds()/1000000)
	}
	if last.Sub(start) < 150*time.Millisecond {
		t.Errorf("should have taken at least 150 milliseconds, took %d", last.Sub(start).Nanoseconds()/1000000)
	}
}

func TestEventualSuccessSucceedsTransparently(t *testing.T) {
	tries := 0
	err := Do(ConstantBackoff(5, 50*time.Millisecond), func() error {
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
}

func TestDoWithContextExitsEarlyWhenContextCanceled(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	tries := 0
	ctx, canceler := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		err = DoWithContext(ctx, ConstantBackoff(100, 100*time.Millisecond), func(ctx context.Context) error {
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

func TestCeaseStopsImmediately(t *testing.T) {
	tries := 0
	err := Do(ConstantBackoff(5, 50*time.Millisecond), func() error {
		tries++
		return Cease(errTest)
	})

	if tries != 1 {
		t.Errorf("expected 1 tries, got %d", tries)
	}
	if err != errTest {
		t.Errorf("err should equal errTest, got: %v", err)
	}
}

func ExampleDo() {
	err := Do(ExponentialBackoff(5, 100*time.Millisecond, 1*time.Second), func() error {
		resp, err := http.Get("http://golang.org")
		switch {
		case err != nil:
			return err
		case resp.StatusCode == 0 || resp.StatusCode >= 500:
			return fmt.Errorf("Retryable HTTP status: %s", http.StatusText(resp.StatusCode))
		case resp.StatusCode != 200:
			return Cease(fmt.Errorf("Non-retryable HTTP status: %s", http.StatusText(resp.StatusCode)))
		}
		return nil
	})
	fmt.Println(err)
}

func ExampleDoWithContext_output() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := DoWithContext(ctx, ConstantBackoff(5, 100*time.Millisecond), func(ctx context.Context) error {
		req, _ := http.NewRequest("GET", "http://golang.org/notfastenough", nil)
		req = req.WithContext(ctx)
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err == nil {
			fmt.Println(resp.StatusCode)
		}
		return err
	})
	fmt.Println(err)
	// Output: Get http://golang.org/notfastenough: context deadline exceeded
}
