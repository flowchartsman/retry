# retry [![Build Status](https://travis-ci.org/flowchartsman/retry.svg?branch=master)](https://travis-ci.org/flowchartsman/v8) [![Go Report Card](https://goreportcard.com/badge/github.com/flowchartsman/retry)](https://goreportcard.com/report/github.com/flowchartsman/retry) [![GoDoc](https://godoc.org/github.com/flowchartsman/retry?status.svg)](https://godoc.org/github.com/flowchartsman/retry)
`♻️¯\_ʕ◔ϖ◔ʔ_/¯`

**retry** is a simple retrier for golang with exponential backoff and context support.

It exists mainly because I found the other libraries either too heavy in implementation or too tedious to use. **retry** is simple and opinionated; it retries your code with an expoential backoff and it lets you bail early. It does not implement constant backoff or any alternative jitter schemes. Retrier objects are intended to be re-used, which means you define them once and then run functions with them whenever you want, as many times as you want. It is safe for concurrent use.

If you're mostly doing HTTP work and you are comfortable with the requirements of converting everything to *io.ReadSeeker*, I highly recommend [hashicorp/go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) (which I didn't use because I didn't want to cache my large POSTs) or [facebookgo/httpcontrol](https://github.com/facebookgo/httpcontrol) (which I didn't use because it only retries GETs).

## Usage

### Simple
```go
package main

import (
	"fmt"
	"net/http"

	"github.com/flowchartsman/retry"
)

func main() {
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
		// Do your thang
	}
}
```

### With context
```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flowchartsman/retry"
)

func main() {
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
		// Do your thang
	}
}
```

## Reference

See:
* https://en.wikipedia.org/wiki/Exponential_backoff
* https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
