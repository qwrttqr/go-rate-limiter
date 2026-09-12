package main

import (
	"log"
	"net/http"
	"qwrttqr-rate-limiter/core/server/handlers"
	"qwrttqr-rate-limiter/core/server/limiting"
)

func main() {
	http.HandleFunc("GET /health", handlers.HealthHandler)
	http.HandleFunc("GET /getConfig", handlers.ParseConfig)

	rateLimiter, err := limiting.NewRateLimiter()
	if err != nil {
		log.Fatalf("failed to create rate limiter: %v", err)
	}
	rateLimiter.Configure()

	http.HandleFunc("POST /limitHttp", rateLimiter.LimitHTTP)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
