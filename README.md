# ratelimitx

`ratelimitx` is a small Go rate limiter library for `net/http` applications.

It currently provides:
- an in-memory fixed-window limiter
- request key helpers
- an HTTP middleware that returns `429 Too Many Requests`

## Install

```bash
go get github.com/rizqishq/ratelimitx
```

## Usage

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/rizqishq/ratelimitx"
)

func main() {
    limiter, err := ratelimitx.NewFixedWindowLimiter(5, time.Minute)
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })

    middleware := ratelimitx.HTTPMiddleware(limiter, ratelimitx.ByIP())

    if err := http.ListenAndServe(":8080", middleware(mux)); err != nil {
        log.Fatal(err)
    }
}
```

## Key Functions

Use the built-in key helpers:

```go
ratelimitx.ByIP()
ratelimitx.ByHeader("X-API-Key")
```

You can also provide your own `KeyFunc`.

## Response Behavior

Successful and blocked requests include these headers:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

Blocked requests also include:
- status `429 Too Many Requests`
- header `Retry-After`
- JSON body:

```json
{"error":"rate limit exceeded"}
```

## Current Scope

This library is intentionally small.

Current limitations:
- in-memory only
- single-process only
- fixed-window algorithm only
- no distributed/shared backend
