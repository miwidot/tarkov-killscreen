package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIURLs_OCRPath(t *testing.T) {
	got := apiURLs(false)
	if len(got) == 0 {
		t.Fatal("expected at least primary URL")
	}
	if !strings.HasSuffix(got[0], "/api/ocr") {
		t.Fatalf("primary should end /api/ocr, got %s", got[0])
	}
	if APIBackupURL != "" {
		if len(got) != 2 {
			t.Fatalf("expected 2 URLs when backup configured, got %d", len(got))
		}
		if !strings.HasSuffix(got[1], "/api/ocr") {
			t.Fatalf("backup should end /api/ocr, got %s", got[1])
		}
	}
}

func TestAPIURLs_SavePath(t *testing.T) {
	got := apiURLs(true)
	for _, u := range got {
		if !strings.HasSuffix(u, "/api/kills/save") {
			t.Fatalf("save URL should end /api/kills/save, got %s", u)
		}
	}
}

func TestDoWithFallback_PrimarySucceeds(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"primary":true}`)
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("backup should not be called when primary works")
	}))
	defer backup.Close()

	status, body, err := doWithFallback([]string{primary.URL, backup.URL}, []byte("test"), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}
	if !strings.Contains(string(body), `"primary":true`) {
		t.Fatalf("expected primary response, got %s", body)
	}
}

func TestDoWithFallback_PrimaryDownUsesBackup(t *testing.T) {
	primary := "http://127.0.0.1:1" // refused

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"backup":true}`)
	}))
	defer backup.Close()

	status, body, err := doWithFallback([]string{primary, backup.URL}, []byte("test"), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Fatalf("expected 200 from backup, got %d", status)
	}
	if !strings.Contains(string(body), `"backup":true`) {
		t.Fatalf("expected backup response, got %s", body)
	}
}

func TestDoWithFallback_BothDownReturnsError(t *testing.T) {
	_, _, err := doWithFallback([]string{"http://127.0.0.1:1", "http://127.0.0.1:2"}, []byte("test"), http.Header{})
	if err == nil {
		t.Fatal("expected error when both hosts down")
	}
}

func TestDoWithFallback_HeadersForwarded(t *testing.T) {
	gotAuth := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer test-token")
	_, _, err := doWithFallback([]string{server.URL}, []byte("test"), headers)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected Authorization header, got %q", gotAuth)
	}
}

func TestDoWithFallback_5xxNotRetried(t *testing.T) {
	primaryHits := 0
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits++
		w.WriteHeader(500)
		fmt.Fprint(w, "server error")
	}))
	defer primary.Close()

	backupHits := 0
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupHits++
	}))
	defer backup.Close()

	status, _, err := doWithFallback([]string{primary.URL, backup.URL}, []byte("test"), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 500 {
		t.Fatalf("expected 500 from primary, got %d", status)
	}
	if primaryHits != 1 {
		t.Fatalf("primary should be hit once, got %d", primaryHits)
	}
	if backupHits != 0 {
		t.Fatalf("backup should NOT be hit on 5xx (server is up), got %d hits", backupHits)
	}
}
