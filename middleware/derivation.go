package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/netip"
)

type KeyDerivationFunction func(r *http.Request) (string, error)

// The CostDerivationFunction should return more than 0 on each request
type CostDerivationFunction func(r *http.Request) (float64, error)

func KeyRemoteIp(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return host, nil
	}
	ip = ip.Unmap()
	if ip.Is6() {
		if p, err := ip.Prefix(64); err == nil {
			return p.String(), nil
		}
	}
	return ip.String(), nil
}

func keyAuthorizationHeader(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", fmt.Errorf("authorization header is absent")
	}

	sum := sha256.Sum256([]byte(token))
	return "auth:" + hex.EncodeToString(sum[:]), nil
}
