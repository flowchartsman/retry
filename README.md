# retry [![Build Status](https://travis-ci.org/flowchartsman/retry.svg?branch=master)](https://travis-ci.org/flowchartsman/v8) [![Go Report Card](https://goreportcard.com/badge/github.com/flowchartsman/retry)](https://goreportcard.com/report/github.com/flowchartsman/retry) [![GoDoc](https://godoc.org/github.com/flowchartsman/retry?status.svg)](https://godoc.org/github.com/flowchartsman/retry)

**retry** is a simple retrier for golang with exponential backoff and context support.

## Usage

### Simple
```go
err := retry.Do(retry.ExponentialBackoff(5, 100*time.Millisecond, 1*time.Second), func() error {
	resp, err := http.Get("http://golang.org")
	switch {
	case err != nil:
		return err
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("HTTP status: %s", http.StatusText(resp.StatusCode))
	}
	return nil
})
```

### With context
```go
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
```

## Reference

See:
* https://en.wikipedia.org/wiki/Exponential_backoff
* https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
