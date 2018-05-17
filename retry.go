package retry

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// Default backoff
const (
	DefaultMaxTries       = 5
	DefaultInitialDelayMS = 200
	DefaultMaxDelayMS     = 1000
)

// Retrier retries code blocks with or without context using an exponential
// backoff algorithm with jitter. It is intended to be used as a retry policy,
// which means it is safe to create and use concurrently.
type Retrier struct {
	maxTries     int
	initialDelay int
	maxDelay     int
}

// NewRetrier returns a retrier for retrying functions with expoential backoff.
// If any of the values are <= 0, they will be set to their respective defaults.
func NewRetrier(maxTries, initialDelay, maxDelay int) *Retrier {
	if maxTries <= 0 {
		maxTries = DefaultMaxTries
	}
	if initialDelay <= 0 {
		initialDelay = DefaultInitialDelayMS
	}
	if maxDelay <= 0 {
		maxDelay = DefaultMaxDelayMS
	}
	return &Retrier{maxTries, initialDelay, maxDelay}
}

// Run runs a function until it returns nil, until it returns a terminal error,
// or until it has failed the maximum set number of iterations
func (r *Retrier) Run(funcToRetry func() error) error {
	attempts := 0
	for {
		// Attempt to run the function
		err := funcToRetry()
		// If there's no error, we're done!
		if err == nil {
			return nil
		}

		attempts++
		// If we've just run our last attempt, return the error we got
		if attempts == r.maxTries {
			return err
		}

		// Check if the error is a terminal error. If so, stop!
		switch v := err.(type) {
		case terminalError:
			return v.e
		}
		// Otherwise wait for the next duration
		time.Sleep(getnextBackoff(attempts, r.initialDelay, r.maxDelay))
	}
}

// RunContext runs a function until it returns nil, until it returns a terminal
// error, until its context is done, or until it has failed the maximum set
// number of iterations.
//
// Note: it is the responsibility of the called function to do its part in
// honoring context deadlines. retry has no special magic around this, and will
// simply stop the retry loop when the function returns if the context is done.
func (r *Retrier) RunContext(ctx context.Context, funcToRetry func(context.Context) error) error {
	attempts := 0
	for {
		// Attempt to run the function
		err := funcToRetry(ctx)
		// If there's no error, we're done!
		if err == nil {
			return nil
		}

		attempts++
		// If we've just run our last attempt, return the error we got
		if attempts == r.maxTries {
			return err
		}

		// Check if the error is a terminal error. If so, stop!
		switch v := err.(type) {
		case terminalError:
			return v.e
		}
		// Otherwise wait for the next duration or until the context is done,
		// whichever comes first
		select {
		case <-time.NewTimer(getnextBackoff(attempts, r.initialDelay, r.maxDelay)).C:
			// duration elapsed, loop
		case <-ctx.Done():
			// context cancelled, return the last error we got
			return err
		}
	}
}

// Stop signals retry that the error we are returning is a terminal error, which
// means we no longer wish to continue retrying the code
func Stop(err error) error {
	return terminalError{err}
}

// terminalError represents and error that we don't wish to retry from.
type terminalError struct {
	e error
}

// Error implements error
func (t terminalError) Error() string {
	return t.e.Error()
}

func getnextBackoff(attempts, initialDelay, maxDelay int) time.Duration {
	return min(
		time.Duration(maxDelay)*time.Millisecond,
		time.Duration(randInt63n(int64(initialDelay)*(1<<uint(attempts))))*time.Millisecond,
	)
}

func min(a, b time.Duration) time.Duration {
	if a > b {
		return b
	}
	return a
}

var (
	randSource = rand.New(rand.NewSource(time.Now().UnixNano()))
	randMux    sync.Mutex
)

func randInt63n(i int64) int64 {
	randMux.Lock()
	defer randMux.Unlock()
	return randSource.Int63n(i)
}
