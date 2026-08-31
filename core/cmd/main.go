package main

import (
	"log"
	"net/http"
	"qwrttqr-rate-limiter/core/server/handlers"
)

func main() {
	http.HandleFunc("GET /health", handlers.HealthHandler)
	http.HandleFunc("GET /getConfig", handlers.ParseConfig)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
