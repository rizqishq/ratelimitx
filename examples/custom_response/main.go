package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rizqishq/ratelimitx"
)

func main() {
	limiter, err := ratelimitx.NewFixedWindowLimiter(2, time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})

	handler := ratelimitx.WrapHTTPWith(
		mux,
		limiter,
		ratelimitx.ByIP(),
		ratelimitx.HTTPMiddlewareOptions{
			OnRejected: func(w http.ResponseWriter, r *http.Request, result ratelimitx.Result) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = fmt.Fprintf(w, "rate limit exceeded, try again in %s\n", result.RetryAfter.Round(time.Second))
			},
		},
	)

	log.Println("listening on :8081")
	if err := http.ListenAndServe(":8081", handler); err != nil {
		log.Fatal(err)
	}
}
