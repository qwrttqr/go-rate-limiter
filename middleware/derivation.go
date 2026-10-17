package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// normalizeIP accepts string "host:port"
func normalizeIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr // no port present
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	ip = ip.Unmap()
	if ip.Is6() {
		if p, err := ip.Prefix(64); err == nil {
			return p.String()
		}
	}
	return ip.String()
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "auth:" + hex.EncodeToString(sum[:])
}

type HTTPKeyDerivationFunction func(r *http.Request) (string, error)

// The HTTPCostDerivationFunction should return more than 0 on each request
type HTTPCostDerivationFunction func(r *http.Request) (float64, error)

func HTTPKeyRemoteIP(r *http.Request) (string, error) {
	return normalizeIP(r.RemoteAddr), nil
}

func HTTPKeyAuthorizationHeader(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", errors.New("authorization header is absent")
	}

	return hashToken(token), nil
}

type GRPCKeyDerivationFunction func(ctx context.Context) (string, error)

// The GRPCCostDerivationFunction should return more than 0 on each request
type GRPCCostDerivationFunction func(ctx context.Context) (float64, error)

func GRPCKeyRemoteIP(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("no peer info in context")
	}
	return normalizeIP(p.Addr.String()), nil
}

func GRPCKeyAuthorizationHeader(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("no metadata in context")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 || vals[0] == "" {
		return "", errors.New("authorization metadata is absent")
	}
	return hashToken(vals[0]), nil
}
