// dns_fallback.go - DNS Resolution Fallback
//
// On startup the HTTP client uses the OS resolver. If the OS resolver
// fails to resolve a hostname (broken ISP resolver, misconfigured router,
// etc.), we retry the lookup via Cloudflare DNS (1.1.1.1) before giving up.
//
// Performance: zero overhead in the normal case — system DNS is tried first
// and the result is cached by the HTTP transport's connection pool. The
// fallback only fires when the OS resolver returns an error.
//
// Note: Cloudflare 1.1.1.1 also validates DNSSEC. For DNSSEC-related
// outages on .de (e.g. DENIC RRSIG breakage) this fallback alone would not
// help; that requires a non-validating resolver or a backup hostname on a
// different TLD.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// fallbackResolver queries Cloudflare DNS directly via UDP, bypassing
// any broken local/ISP resolver configuration.
var fallbackResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		// Try Cloudflare primary, then secondary
		conn, err := d.DialContext(ctx, "udp", "1.1.1.1:53")
		if err == nil {
			return conn, nil
		}
		return d.DialContext(ctx, "udp", "1.0.0.1:53")
	},
}

// dialContextWithFallback resolves the hostname using the OS resolver and
// retries via Cloudflare DNS if the OS lookup fails. It then connects to
// the resolved address using a standard TCP dialer.
func dialContextWithFallback(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Try OS resolver via standard dialer first — fast path, no overhead
	conn, err := dialer.DialContext(ctx, network, addr)
	if err == nil {
		return conn, nil
	}

	// Detect DNS-related failures (vs network/connection failures)
	var dnsErr *net.DNSError
	if !asDNSError(err, &dnsErr) {
		return nil, err // not a DNS issue, surface original error
	}

	debugLog("[DNS] OS resolver failed for %s, trying Cloudflare: %v\n", host, err)

	// Fall back to Cloudflare resolver
	resolveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ips, err := fallbackResolver.LookupIP(resolveCtx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("both OS and Cloudflare DNS failed: %w", err)
	}

	debugLog("[DNS] Cloudflare resolved %s to %v\n", host, ips)

	// Try each resolved IP until one connects
	var lastErr error
	for _, ip := range ips {
		target := net.JoinHostPort(ip.String(), port)
		conn, err := dialer.DialContext(ctx, network, target)
		if err == nil {
			debugLog("[DNS] Connected via fallback to %s\n", target)
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("connect failed for all fallback IPs: %w", lastErr)
}

// asDNSError reports whether err wraps a *net.DNSError and assigns it to target.
// Avoids a hard dependency on errors.As for clarity.
func asDNSError(err error, target **net.DNSError) bool {
	for e := err; e != nil; {
		if d, ok := e.(*net.DNSError); ok {
			*target = d
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// newFallbackTransport returns an http.Transport that uses our DNS fallback.
func newFallbackTransport() *http.Transport {
	return &http.Transport{
		DialContext:           dialContextWithFallback,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
