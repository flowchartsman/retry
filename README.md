# retry [![Build Status](https://travis-ci.org/flowchartsman/retry.svg?branch=master)](https://travis-ci.org/flowchartsman/v8) [![Go Report Card](https://goreportcard.com/badge/github.com/flowchartsman/retry)](https://goreportcard.com/report/github.com/flowchartsman/retry) [![GoDoc](https://godoc.org/github.com/flowchartsman/retry?status.svg)](https://godoc.org/github.com/flowchartsman/retry)[![Coverage Status](https://coveralls.io/repos/github/flowchartsman/retry/badge.svg?branch=master)](https://coveralls.io/github/flowchartsman/retry?branch=master)
`♻️¯\_ʕ◔ϖ◔ʔ_/¯`

**retry** is a simple retrier for golang with exponential backoff and context support.

It exists mainly because I found the other libraries either too heavy in implementation or not to my liking. **retry** is simple and opinionated; it retries your code with an expoential backoff and it lets you bail early. It does not implement constant backoff or any alternative jitter schemes. Retrier objects are intended to be re-used, which means you define them once and then run functions with them whenever you want, as many times as you want. It is safe for concurrent use, and it supports `context`.

# Usage

## Simple
```go
retrier := retry.NewRetrier(5, 100, 1000)
err := retrier.Run(func() error {
    resp, err := http.Get("http://golang.org")
    switch {
    case err != nil:
        return err
    case resp.StatusCode == 0 || resp.StatusCode >= 500:
        return fmt.Errorf("Retryable HTTP status: %s", http.StatusText(resp.StatusCode))
    case resp.StatusCode != 200:
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
retrier := retry.NewRetrier(5, 100, 1000)
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

If you're doing HTTP work there are some good alternatives out there that add a layer on top of the standard libraries as well as providing special logic to help you automatically determine whether or not to retry a request.

- [hashicorp/go-retryablehttp](https://github.com/hashicorp/go-retryablehttp)
  - A very good library, but it requires conversion of all `io.Reader`s to `io.ReadSeeker`s, and one of my use-cases didn't allow for unconstrained cacheing of `POST` bodies.
- [facebookgo/httpcontrol](https://github.com/facebookgo/httpcontrol)
  - A great fully-featured transport. Only retries `GET`s, though :(
- [sethgrid/pester](https://github.com/sethgrid/pester)
  - Another good client, but had more options than I needed and also caches request bodies transparently.

Some of the other libs I considered:
- [jpillora/backoff](https://github.com/jpillora/backoff)
  - A little more bare bones than I wanted and no builtin concurrency safety. No context support.
- [cenkalti/backoff](https://github.com/cenkalti/backoff)
  - A good library, but has some issues with context deadlines/timeouts. Can't "share" backup strategies / not thread-safe.
- [gopkg.in/retry.v1](https://gopkg.in/retry.v1)
  - Iterator-based and a little awkward for me to use personally. I preferred to abstract the loop away.

# Reference

See:
* https://en.wikipedia.org/wiki/Exponential_backoff
* https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
