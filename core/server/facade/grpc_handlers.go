package facade

import (
	"context"
	"qwrttqr-rate-limiter/core/server/algos"
	ratelimiterv1 "qwrttqr-rate-limiter/gen/ratelimiter/v1"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

type GRPCServer struct {
	ratelimiterv1.UnimplementedRateLimiterServer
	l algos.Limiter
}

func NewGRPCServer(l algos.Limiter) *GRPCServer {
	return &GRPCServer{l: l}
}

func (s *GRPCServer) Allow(ctx context.Context, in *ratelimiterv1.AllowRequest) (*ratelimiterv1.AllowResponse, error) {
	tokens := in.GetTokens()
	if tokens == 0.0 {
		tokens = 1.0
	}
	req, err := NewRequest(in.GetKey(), tokens)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	d, err := s.l.Allow(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "rate limiter error")
	}

	return &ratelimiterv1.AllowResponse{
		Allowed:    d.Allowed,
		RetryAfter: durationpb.New(time.Duration(d.RetryAfter) * time.Second),
	}, nil
}
