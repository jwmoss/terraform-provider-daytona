// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestPatchJSONRetriesTransientErrors verifies the shared retrying HTTP client
// recovers from transient 5xx responses instead of failing the operation on the
// first blip. It exercises patchJSON because that path uses the same client the
// generated Daytona client is configured with.
func TestPatchJSONRetriesTransientErrors(t *testing.T) {
	prevMin, prevMax := retryWaitMin, retryWaitMax
	retryWaitMin, retryWaitMax = time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() {
		retryWaitMin, retryWaitMax = prevMin, prevMax
	})

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected method %s, got %s", http.MethodPatch, r.Method)
		}
		if got := attempts.Add(1); got < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := newDaytonaClient(server.URL, "test-key", "", "test")

	resp, err := client.patchJSON(context.Background(), "/things/1", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected patchJSON error: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected final status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts (2 transient failures then success), got %d", got)
	}
}

// TestPatchJSONDoesNotRetryClientErrors verifies a non-retryable 4xx response
// fails immediately rather than burning the retry budget on an unrecoverable
// error.
func TestPatchJSONDoesNotRetryClientErrors(t *testing.T) {
	prevMin, prevMax := retryWaitMin, retryWaitMax
	retryWaitMin, retryWaitMax = time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() {
		retryWaitMin, retryWaitMax = prevMin, prevMax
	})

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newDaytonaClient(server.URL, "test-key", "", "test")

	_, err := client.patchJSON(context.Background(), "/things/1", map[string]string{"key": "value"})
	if err == nil {
		t.Fatal("expected patchJSON to return an error for a 400 response")
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("expected exactly 1 attempt for a non-retryable 400, got %d", got)
	}
}
