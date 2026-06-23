package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type daytonaHTTPExpectation struct {
	method      string
	path        string
	response    string
	statusCode  int
	check       func(*testing.T, *http.Request)
	captureJSON *map[string]any
}

type daytonaHTTPRecorder struct {
	mu     sync.Mutex
	counts map[string]int
}

func newDaytonaHTTPServer(t *testing.T, expectations ...daytonaHTTPExpectation) (*httptest.Server, *daytonaHTTPRecorder) {
	t.Helper()

	recorder := &daytonaHTTPRecorder{counts: map[string]int{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()

		for _, expectation := range expectations {
			if expectation.method != r.Method || expectation.path != path {
				continue
			}

			recorder.record(r.Method, path)
			if expectation.check != nil {
				expectation.check(t, r)
			}
			if expectation.captureJSON != nil {
				decodeTestPayload(t, r.Body, expectation.captureJSON)
			}

			w.Header().Set("Content-Type", "application/json")
			if expectation.statusCode != 0 {
				w.WriteHeader(expectation.statusCode)
			}
			_, _ = w.Write([]byte(expectation.response))
			return
		}

		t.Fatalf("unexpected request %s %s", r.Method, path)
	}))

	return server, recorder
}

func (r *daytonaHTTPRecorder) Count(method, path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counts[method+" "+path]
}

func (r *daytonaHTTPRecorder) Counts() map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()

	counts := make(map[string]int, len(r.counts))
	for key, count := range r.counts {
		counts[key] = count
	}
	return counts
}

func (r *daytonaHTTPRecorder) record(method, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[method+" "+path]++
}

func terraformValue(t *testing.T, value interface {
	ToTerraformValue(context.Context) (tftypes.Value, error)
}) tftypes.Value {
	t.Helper()

	terraformValue, err := value.ToTerraformValue(context.Background())
	if err != nil {
		t.Fatalf("unable to convert value to Terraform value: %s", err)
	}

	return terraformValue
}
