package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"qwrttqr-rate-limiter/core/server/facade"
	"qwrttqr-rate-limiter/core/server/limiting"
	ratelimiterv1 "qwrttqr-rate-limiter/gen/ratelimiter/v1"
	"syscall"
	"time"

	"google.golang.org/grpc"
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

	httpSrv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	grpcSrv := grpc.NewServer()
	ratelimiterv1.RegisterRateLimiterServer(grpcSrv, facade.NewGRPCServer(rl))

	tcpLis, err := net.Listen("tcp", ":8081")
	errCh := make(chan error, 2)
	go func() { errCh <- httpSrv.ListenAndServe() }()
	log.Printf("listening http on %s", httpSrv.Addr)
	go func() { errCh <- grpcSrv.Serve(tcpLis) }()
	log.Printf("listening grpc on %s", tcpLis.Addr())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcDone := make(chan struct{})
	go func() { grpcSrv.GracefulStop(); close(grpcDone) }()

	select {
	case <-grpcDone:
	case <-shutdownCtx.Done():
		grpcSrv.Stop()
	}

	return httpSrv.Shutdown(shutdownCtx)
}
