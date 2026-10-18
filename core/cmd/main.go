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
	"qwrttqr-rate-limiter/core/server/metrics"
	ratelimiterv1 "qwrttqr-rate-limiter/gen/ratelimiter/v1"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	reg := prometheus.NewRegistry()
	rec := metrics.NewPromRecorder(reg)

	rl, err := limiting.NewRateLimiter(rec)
	if err != nil {
		return fmt.Errorf("create rate limiter: %w", err)
	}
	defer rl.Close()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	metricsSrv := &http.Server{Addr: ":9091", Handler: metricsMux, ReadHeaderTimeout: 5 * time.Second}

	limitingMux := http.NewServeMux()
	limitingMux.HandleFunc("GET /health", facade.HealthHandler)
	limitingMux.HandleFunc("POST /limit", facade.LimitHandler(rl))

	limitingSrv := &http.Server{Addr: ":8080", Handler: limitingMux, ReadHeaderTimeout: 5 * time.Second}
	grpcSrv := grpc.NewServer()
	ratelimiterv1.RegisterRateLimiterServer(grpcSrv, facade.NewGRPCServer(rl))

	tcpLis, err := net.Listen("tcp", ":8081")
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	errCh := make(chan error, 3)

	go func() { errCh <- metricsSrv.ListenAndServe() }()
	log.Printf("metrics available on %s", metricsSrv.Addr)
	go func() { errCh <- limitingSrv.ListenAndServe() }()
	log.Printf("listening http on %s", limitingSrv.Addr)
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

	_ = metricsSrv.Shutdown(shutdownCtx)
	return limitingSrv.Shutdown(shutdownCtx)
}
