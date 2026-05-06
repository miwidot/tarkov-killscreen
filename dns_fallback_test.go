package main

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestAsDNSError_DirectDNSError(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "example.invalid"}
	var got *net.DNSError
	if !asDNSError(err, &got) {
		t.Fatal("should detect direct *net.DNSError")
	}
	if got.Name != "example.invalid" {
		t.Fatalf("wrong DNSError, got %v", got)
	}
}

func TestAsDNSError_WrappedDNSError(t *testing.T) {
	dnsErr := &net.DNSError{Err: "timeout", Name: "test.local"}
	wrapped := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: dnsErr,
	}
	var got *net.DNSError
	if !asDNSError(wrapped, &got) {
		t.Fatal("should detect wrapped *net.DNSError")
	}
	if got.Err != "timeout" {
		t.Fatalf("wrong DNSError, got %v", got)
	}
}

func TestAsDNSError_NonDNSError(t *testing.T) {
	err := errors.New("connection refused")
	var got *net.DNSError
	if asDNSError(err, &got) {
		t.Fatal("plain error should not be classified as DNS")
	}
}

func TestAsDNSError_NetworkError(t *testing.T) {
	// Connection refused (not DNS-related)
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}
	var got *net.DNSError
	if asDNSError(err, &got) {
		t.Fatal("network errors must NOT trigger DNS fallback")
	}
}

func TestFallbackResolver_RealLookup(t *testing.T) {
	// Skip if no internet; this is a smoke test against the real fallback path
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	ips, err := fallbackResolver.LookupIP(ctx, "ip", "cloudflare.com")
	if err != nil {
		t.Skipf("fallback resolver unreachable (no internet?): %v", err)
	}
	if len(ips) == 0 {
		t.Fatal("expected at least one IP")
	}
	t.Logf("fallback resolver returned %d IPs for cloudflare.com", len(ips))
}

func TestDialContext_NoDNSFallbackForRefusedConnection(t *testing.T) {
	// 127.0.0.1:1 should always be refused (port 1 not listening).
	// This must NOT trigger DNS fallback — it's a connection error, not DNS.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := dialContextWithFallback(ctx, "tcp", "127.0.0.1:1")
	if err == nil {
		t.Fatal("expected connection error to 127.0.0.1:1")
	}

	var dnsErr *net.DNSError
	if asDNSError(err, &dnsErr) {
		t.Fatalf("connection refusal should not surface as DNS error: %v", err)
	}
}
