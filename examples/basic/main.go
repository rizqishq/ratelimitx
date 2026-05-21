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
		_, _ = w.Write([]byte("ok\n"))
	})

	handler := ratelimitx.HTTPMiddleware(limiter, ratelimitx.ByIP())(mux)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
