package main

import (
	"log"
	"net/http"
	"qwrttqr-rate-limiter/core/server/handlers"
)

func main() {
	http.HandleFunc("/health", handlers.HealthHandler)
	http.HandleFunc("/getConfig", handlers.ParseConfig)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
