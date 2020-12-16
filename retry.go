package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Default backoff
const (
	DefaultMaxTries     = 5
	DefaultInitialDelay = time.Millisecond * 200
	DefaultMaxDelay     = time.Millisecond * 1000
)

// Retrier retries code blocks with or without context using an exponential
// backoff algorithm with jitter. It is intended to be used as a retry policy,
// which means it is safe to create and use concurrently.
type Retrier struct {
	maxTries     int
	initialDelay time.Duration
	maxDelay     time.Duration
}

// NewRetrier returns a retrier for retrying functions with expoential backoff.
// If any of the values are <= 0, they will be set to their respective defaults.
func NewRetrier(maxTries int, initialDelay, maxDelay time.Duration) *Retrier {
	if maxTries <= 0 {
		maxTries = DefaultMaxTries
	}
	if initialDelay <= 0 {
		initialDelay = DefaultInitialDelay
	}
	if maxDelay <= 0 {
		maxDelay = DefaultMaxDelay
	}
	return &Retrier{maxTries, initialDelay, maxDelay}
}

// Run runs a function until it returns nil, until it returns a terminal error,
// or until it has failed the maximum set number of iterations
func (r *Retrier) Run(funcToRetry func() error) error {
	return r.RunContext(context.Background(), func(_ context.Context) error {
		return funcToRetry()
	})
}

// RunContext runs a function until it returns nil, until it returns a terminal
// error, until its context is done, or until it has failed the maximum set
// number of iterations.
//
// Note: it is the responsibility of the called function to do its part in
// honoring context deadlines. retry has no special magic around this, and will
// simply stop the retry loop when the function returns if the context is done.
func (r *Retrier) RunContext(ctx context.Context, funcToRetry func(context.Context) error) error {
	maxTries := r.maxTries
	initialDelay := r.initialDelay
	maxDelay := r.maxDelay
	if maxTries <= 0 {
		maxTries = DefaultMaxTries
	}
	if initialDelay <= 0 {
		initialDelay = DefaultInitialDelay
	}
	if maxDelay <= 0 {
		maxDelay = DefaultMaxDelay
	}
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))
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
		if attempts == maxTries {
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
		case <-time.NewTimer(getnextBackoff(attempts, initialDelay, maxDelay, randSource)).C:
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

func getnextBackoff(attempts int, initialDelay, maxDelay time.Duration, randSource *rand.Rand) time.Duration {
	var backoff time.Duration

	// this complexity is to limit the backoff to values that fit into signed 64 bit numbers
	attemptsLimit := int(math.Log2(float64(initialDelay))) + 1
	if attemptsLimit < 63-attempts {
		backoff = time.Duration(1<<uint64(attempts)) * jitterDuration(initialDelay, randSource)
		if backoff > maxDelay {
			backoff = jitterDuration(maxDelay/2, randSource)
		}
	} else {
		backoff = jitterDuration(maxDelay/2, randSource)
	}
	return backoff + initialDelay
}

func jitterDuration(duration time.Duration, randSource *rand.Rand) time.Duration {
	return time.Duration(randSource.Int63n(int64(duration)) + int64(duration))
}
