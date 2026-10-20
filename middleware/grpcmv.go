package middleware

import (
	"context"
	"log"
	"qwrttqr-rate-limiter/core/server/algos"
	"qwrttqr-rate-limiter/core/server/facade"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func newGRPCCheck(
	algo algos.Limiter,
	keyDer GRPCKeyDerivationFunction,
	costDer GRPCCostDerivationFunction,
	failOpen bool,
) func(ctx context.Context) error {
	if algo == nil || keyDer == nil {
		panic("middleware: limiter and key function are required")
	}
	return func(ctx context.Context) error {
		fail := func(stage string, err error) error {
			log.Printf("limiter error on %s: %v", stage, err)
			if failOpen {
				return nil
			}
			return status.Error(codes.Unavailable, "service unavailable")
		}

		tokens := 1.0
		if costDer != nil {
			t, err := costDer(ctx)
			if err != nil {
				return fail("cost derivation", err)
			}
			tokens = t
		}

		key, err := keyDer(ctx)
		if err != nil {
			return fail("key derivation", err)
		}

		req, err := facade.NewRequest(key, tokens)
		if err != nil {
			return fail("request creation", err)
		}

		decision, err := algo.Allow(ctx, req)
		if err != nil {
			return fail("allow", err)
		}
		if !decision.Allowed {
			md := metadata.Pairs("retry-after", strconv.FormatInt(decision.RetryAfter, 10))
			_ = grpc.SetTrailer(ctx, md)
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return nil
	}
}

func NewGRPCUnaryInterceptor(
	algo algos.Limiter,
	keyDer GRPCKeyDerivationFunction,
	costDer GRPCCostDerivationFunction,
	failOpen bool,
) grpc.UnaryServerInterceptor {
	check := newGRPCCheck(algo, keyDer, costDer, failOpen)
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if err := check(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func NewGRPCStreamInterceptor(
	algo algos.Limiter,
	keyDer GRPCKeyDerivationFunction,
	costDer GRPCCostDerivationFunction,
	failOpen bool,
) grpc.StreamServerInterceptor {
	check := newGRPCCheck(algo, keyDer, costDer, failOpen)
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := check(ss.Context()); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}
