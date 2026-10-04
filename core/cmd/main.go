package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"qwrttqr-rate-limiter/core/server/facade"
	"qwrttqr-rate-limiter/core/server/limiting"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rl, err := limiting.NewRateLimiter()
	if err != nil {
		return fmt.Errorf("create rate limiter: %w", err)
	}
	defer rl.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", facade.HealthHandler)
	mux.HandleFunc("POST /limit", facade.LimitHandler(rl))

	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Printf("listening on %s", srv.Addr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
