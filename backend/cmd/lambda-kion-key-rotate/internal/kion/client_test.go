package kion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRotate_Success(t *testing.T) {
	var gotPath, gotAuth, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"data":{"key":"new-key-xyz"}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	newKey, err := c.Rotate(context.Background(), "current-key-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newKey != "new-key-xyz" {
		t.Errorf("got new key %q, want %q", newKey, "new-key-xyz")
	}
	if gotPath != "/api/v3/app-api-key/rotate" {
		t.Errorf("path = %q, want /api/v3/app-api-key/rotate", gotPath)
	}
	if gotAuth != "Bearer current-key-abc" {
		t.Errorf("auth = %q, want Bearer current-key-abc", gotAuth)
	}
	var parsed rotateRequest
	if err := json.Unmarshal([]byte(gotBody), &parsed); err != nil || parsed.Key != "current-key-abc" {
		t.Errorf("body = %q, want JSON {key:current-key-abc}", gotBody)
	}
}

func TestRotate_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"upstream"}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"key":"eventual-key"}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	// Shrink backoff for the test by using a short context deadline for the final attempt cushion.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	newKey, err := c.Rotate(ctx, "current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newKey != "eventual-key" {
		t.Errorf("got %q, want eventual-key", newKey)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("got %d calls, want 3", calls)
	}
}

func TestRotate_DoesNotRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid key"}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	_, err := c.Rotate(context.Background(), "dead-key")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("got %d calls, want 1 (no retry on 4xx)", calls)
	}
}

func TestRotate_EmptyCurrentKey(t *testing.T) {
	c := NewClient("http://nowhere")
	_, err := c.Rotate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}

func TestRotate_MissingDataKeyInResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	_, err := c.Rotate(context.Background(), "current")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing data.key") {
		t.Errorf("expected missing-data-key error, got %v", err)
	}
}
