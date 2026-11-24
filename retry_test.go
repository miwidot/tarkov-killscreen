package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// shortRetryDelays makes withRetry instant for the duration of a test.
func shortRetryDelays(t *testing.T) {
	t.Helper()
	saved := retryDelays
	retryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { retryDelays = saved })
}

func TestWithRetry_RecoversFromTransientFailure(t *testing.T) {
	shortRetryDelays(t)
	calls := 0
	err := withRetry("test", isRetryableUpload, func() error {
		calls++
		if calls < 3 {
			return &apiError{Status: 503, text: "API error: 503"}
		}
		return nil
	})
	if err != nil {
		t.Fatal("expected success after retries, got:", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_GivesUpAfterAllAttempts(t *testing.T) {
	shortRetryDelays(t)
	calls := 0
	err := withRetry("test", isRetryableUpload, func() error {
		calls++
		return &apiError{Status: 502, text: "API error: 502"}
	})
	if err == nil {
		t.Fatal("expected the last error")
	}
	if want := len(retryDelays) + 1; calls != want {
		t.Fatalf("expected %d calls, got %d", want, calls)
	}
}

func TestWithRetry_DoesNotRepeatPermanentErrors(t *testing.T) {
	shortRetryDelays(t)
	for _, permanent := range []error{
		ErrDeviceLocked,
		&apiError{Status: 401, text: "API error: 401"},
		fmt.Errorf("no API token configured"),
	} {
		calls := 0
		withRetry("test", isRetryableUpload, func() error {
			calls++
			return permanent
		})
		if calls != 1 {
			t.Errorf("%v: expected 1 call, got %d", permanent, calls)
		}
	}
}

func TestIsRetryableSave_OnlyWhenRequestDidNotLand(t *testing.T) {
	dialErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"never sent", &transportError{err: dialErr, notSent: true}, true},
		{"sent, response lost", &transportError{err: errors.New("timeout awaiting headers")}, false},
		{"proxy 502", &apiError{Status: 502}, true},
		{"proxy 503", &apiError{Status: 503}, true},
		{"app 500 may have saved", &apiError{Status: 500}, false},
		{"gateway timeout may have saved", &apiError{Status: 504}, false},
		{"validation error", &apiError{Status: 400}, false},
	}
	for _, c := range cases {
		if got := isRetryableSave(c.err); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// An upload is idempotent, so a lost response is worth repeating there.
	if !isRetryableUpload(&transportError{err: errors.New("timeout awaiting headers")}) {
		t.Error("upload with lost response should be retryable")
	}
}

func TestDoWithFallback_ConnectFailureIsMarkedNotSent(t *testing.T) {
	// A server that is started and immediately closed leaves a port nothing
	// listens on: the connect fails, so the request is provably never sent.
	srv := httptest.NewServer(http.NotFoundHandler())
	deadURL := srv.URL
	srv.Close()

	_, _, err := doWithFallback([]string{deadURL, deadURL}, []byte("{}"), http.Header{})
	var te *transportError
	if !errors.As(err, &te) {
		t.Fatalf("expected *transportError, got %T: %v", err, err)
	}
	if !te.notSent {
		t.Error("connect failure should be marked notSent")
	}
}

func TestDoWithFallback_DroppedResponseIsNotMarkedNotSent(t *testing.T) {
	// The server receives the request, then drops the connection.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}))
	defer srv.Close()

	_, _, err := doWithFallback([]string{srv.URL}, []byte("{}"), http.Header{})
	var te *transportError
	if !errors.As(err, &te) {
		t.Fatalf("expected *transportError, got %T: %v", err, err)
	}
	if te.notSent {
		t.Error("a request the server received must not be marked notSent")
	}
}

func TestMaskToken_ShortTokenDoesNotPanic(t *testing.T) {
	for _, token := range []string{"", "a", "abc", "12345678"} {
		if got := maskToken(token); got != "(short)" {
			t.Errorf("maskToken(%q) = %q", token, got)
		}
	}
	if got := maskToken("abcd1234wxyz"); got != "abcd...wxyz" {
		t.Errorf("unexpected preview %q", got)
	}
}
