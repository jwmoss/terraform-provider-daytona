// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestSafeRetryPolicy locks in which failures are safe to replay. The invariant
// that matters: a non-idempotent mutation (POST/PATCH) must never be retried on
// a 5xx or transport error, because the server may have already applied it and a
// replay would duplicate a create or repeat the mutation. A 429 is always safe
// because the server rejects it before processing.
func TestSafeRetryPolicy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		method     string
		statusCode int
		transport  bool
		wantRetry  bool
	}{
		{name: "GET 503 retries", method: http.MethodGet, statusCode: http.StatusServiceUnavailable, wantRetry: true},
		{name: "GET 429 retries", method: http.MethodGet, statusCode: http.StatusTooManyRequests, wantRetry: true},
		{name: "DELETE 502 retries", method: http.MethodDelete, statusCode: http.StatusBadGateway, wantRetry: true},
		{name: "PUT 500 retries", method: http.MethodPut, statusCode: http.StatusInternalServerError, wantRetry: true},
		{name: "POST 500 does not retry", method: http.MethodPost, statusCode: http.StatusInternalServerError, wantRetry: false},
		{name: "POST 429 retries", method: http.MethodPost, statusCode: http.StatusTooManyRequests, wantRetry: true},
		{name: "PATCH 503 does not retry", method: http.MethodPatch, statusCode: http.StatusServiceUnavailable, wantRetry: false},
		{name: "PATCH 429 retries", method: http.MethodPatch, statusCode: http.StatusTooManyRequests, wantRetry: true},
		{name: "GET 200 does not retry", method: http.MethodGet, statusCode: http.StatusOK, wantRetry: false},
		{name: "POST 400 does not retry", method: http.MethodPost, statusCode: http.StatusBadRequest, wantRetry: false},
		{name: "GET transport error does not retry", method: http.MethodGet, transport: true, wantRetry: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				resp   *http.Response
				reqErr error
			)
			if tc.transport {
				reqErr = errors.New("connection reset")
			} else {
				req, _ := http.NewRequest(tc.method, "https://example.test/resource", nil)
				resp = &http.Response{StatusCode: tc.statusCode, Request: req}
			}

			gotRetry, _ := safeRetryPolicy(context.Background(), resp, reqErr)
			if gotRetry != tc.wantRetry {
				t.Fatalf("safeRetryPolicy(%s, %d, transport=%t) = %t, want %t", tc.method, tc.statusCode, tc.transport, gotRetry, tc.wantRetry)
			}
		})
	}
}

// TestPatchJSONDoesNotRetryServerError proves the wired client does not replay a
// non-idempotent PATCH on a 5xx, even though PATCH flows through the shared
// retrying client.
func TestPatchJSONDoesNotRetryServerError(t *testing.T) {
	prevMin, prevMax := retryWaitMin, retryWaitMax
	retryWaitMin, retryWaitMax = time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() {
		retryWaitMin, retryWaitMax = prevMin, prevMax
	})

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newDaytonaClient(server.URL, "test-key", "", "test")

	_, err := client.patchJSON(context.Background(), "/things/1", map[string]string{"key": "value"})
	if err == nil {
		t.Fatal("expected patchJSON to return an error for a 500 response")
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("expected exactly 1 attempt for a non-idempotent PATCH 500, got %d", got)
	}
}

// TestPatchJSONRetriesRateLimit proves a 429 is retried even for a
// non-idempotent PATCH, because the server rejects it before processing.
func TestPatchJSONRetriesRateLimit(t *testing.T) {
	prevMin, prevMax := retryWaitMin, retryWaitMax
	retryWaitMin, retryWaitMax = time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() {
		retryWaitMin, retryWaitMax = prevMin, prevMax
	})

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if got := attempts.Add(1); got < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
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
		t.Fatalf("expected 3 attempts (2 rate-limited then success), got %d", got)
	}
}

// TestPatchJSONDoesNotRetryClientErrors verifies a non-retryable 4xx fails
// immediately rather than burning the retry budget.
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
