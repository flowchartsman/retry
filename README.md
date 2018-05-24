# retry [![GoDoc](https://godoc.org/github.com/flowchartsman/retry?status.svg)](https://godoc.org/github.com/flowchartsman/retry) [![Build Status](https://img.shields.io/travis/flowchartsman/retry.svg)](https://travis-ci.org/flowchartsman/v8) [![Go Report Card](https://goreportcard.com/badge/github.com/flowchartsman/retry)](https://goreportcard.com/report/github.com/flowchartsman/retry) [![Coverage Status](https://img.shields.io/coveralls/github/flowchartsman/retry.svg)](https://coveralls.io/github/flowchartsman/retry?branch=master)
`♻️¯\_ʕ◔ϖ◔ʔ_/¯`

**retry** is a simple retrier for golang with exponential backoff and context support.

It exists mainly because I found the other libraries either too heavy in implementation or not to my liking.

**retry** is simple and opinionated; it re-runs your code with a [particular](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/) ("full jitter") [exponential backoff](https://en.wikipedia.org/wiki/Exponential_backoff) implementation, it supports context, and it lets you bail early on non-retryable errors. It does not implement constant backoff or alternative jitter schemes.

Retrier objects are intended to be re-used, which means you define them once and then run functions with them whenever you want, as many times as you want. This is safe for concurrent use.

# Usage

## Simple
```go
// create a new retrier that will try a maximum of five times, with
// an initial delay of 100 ms and a maximum delay of 1 second
retrier := retry.NewRetrier(5, 100 * time.Millisecond, time.Second)

err := retrier.Run(func() error {
    resp, err := http.Get("http://golang.org")
    switch {
    case err != nil:
        // request error - return it
        return err
    case resp.StatusCode == 0 || resp.StatusCode >= 500:
        // retryable StatusCode - return it
        return fmt.Errorf("Retryable HTTP status: %s", http.StatusText(resp.StatusCode))
    case resp.StatusCode != 200:
        // non-retryable error - stop now
        return retry.Stop(fmt.Errorf("Non-retryable HTTP status: %s", http.StatusText(resp.StatusCode)))
    }
    return nil
})
if err != nil {
    // handle error
}
```

## With context
```go
// create a new retrier that will try a maximum of five times, with
// an initial delay of 100 ms and a maximum delay of 1 second
retrier := retry.NewRetrier(5, 100 * time.Millisecond, time.Second)
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
if err != nil {
    // handle error
}
```

# Alternatives

Some of the other libs I considered:
- [jpillora/backoff](https://github.com/jpillora/backoff)
  - A little more bare bones than I wanted and no builtin concurrency safety. No context support.
- [cenkalti/backoff](https://github.com/cenkalti/backoff)
  - A good library, but has some issues with context deadlines/timeouts. Can't "share" backup strategies / not thread-safe.
- [gopkg.in/retry.v1](https://gopkg.in/retry.v1)
  - Iterator-based and a little awkward for me to use personally. I preferred to abstract the loop away.
- [eapache/go-resiliency/retrier](https://godoc.org/github.com/eapache/go-resiliency/retrier)
  - Of the alternatives, I like this one the most, but I found the slice of `time.Duration` odd
  - No context support
  - Classifier pattern is not a bad idea, but it really comes down to "do I want to retry or stop?" and I thought it would be more flexible to simply allow the user to implement this logic how they saw fit. I could be open to changing my mind, if the solution is right. PRs welcome ;)

If you're doing HTTP work there are some good alternatives out there that add a layer on top of the standard libraries as well as providing special logic to help you automatically determine whether or not to retry a request.

- [hashicorp/go-retryablehttp](https://github.com/hashicorp/go-retryablehttp)
  - A very good library, but it requires conversion of all `io.Reader`s to `io.ReadSeeker`s, and one of my use-cases didn't allow for unconstrained cacheing of `POST` bodies.
- [facebookgo/httpcontrol](https://github.com/facebookgo/httpcontrol)
  - A great fully-featured transport. Only retries `GET`s, though :(
- [sethgrid/pester](https://github.com/sethgrid/pester)
  - Another good client, but had more options than I needed and also caches request bodies transparently.

# Reference

See:
* https://en.wikipedia.org/wiki/Exponential_backoff
* https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
