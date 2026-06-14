package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestRunnerResourceSchemaOperationalFields(t *testing.T) {
	t.Parallel()

	runnerResource := NewRunnerResource()

	var metadataResp resource.MetadataResponse
	runnerResource.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_runner" {
		t.Fatalf("expected type name %q, got %q", "daytona_runner", metadataResp.TypeName)
	}

	var schemaResp resource.SchemaResponse
	runnerResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	unschedulableAttr, ok := schemaResp.Schema.Attributes["unschedulable"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected unschedulable to be a bool attribute, got %T", schemaResp.Schema.Attributes["unschedulable"])
	}
	if !unschedulableAttr.Optional || !unschedulableAttr.Computed {
		t.Fatal("expected unschedulable to be optional and computed")
	}

	drainingAttr, ok := schemaResp.Schema.Attributes["draining"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected draining to be a bool attribute, got %T", schemaResp.Schema.Attributes["draining"])
	}
	// draining must be tracked in state: write-only values are never diffed by
	// Terraform core, so toggling draining alone would produce an empty plan and
	// the drain request would never be sent.
	if !drainingAttr.Optional || drainingAttr.WriteOnly {
		t.Fatal("expected draining to be optional and tracked in state (not write-only)")
	}
}

func TestRunnerResourceOperationalPatchRequests(t *testing.T) {
	t.Parallel()

	requests := map[string]map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected method %s, got %s", http.MethodPatch, r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(organizationHeader) != "org-1" {
			t.Fatalf("expected organization header %q, got %q", "org-1", r.Header.Get(organizationHeader))
		}
		if r.Header.Get("User-Agent") != "terraform-provider-daytona/test" {
			t.Fatalf("expected provider user agent, got %q", r.Header.Get("User-Agent"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %s", err)
		}
		var payload map[string]bool
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed unmarshalling body %q: %s", string(body), err)
		}
		requests[r.URL.EscapedPath()] = payload

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(runnerResourceJSON("runner-1", payload["unschedulable"])))
	}))
	defer server.Close()

	runnerResource := &RunnerResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}

	if _, err := runnerResource.updateRunnerScheduling(context.Background(), "runner-1", true); err != nil {
		t.Fatalf("unexpected scheduling update error: %s", err)
	}
	if _, err := runnerResource.updateRunnerDraining(context.Background(), "runner-1", true); err != nil {
		t.Fatalf("unexpected draining update error: %s", err)
	}

	scheduling := requests["/runners/runner-1/scheduling"]
	if scheduling == nil || !scheduling["unschedulable"] {
		t.Fatalf("expected scheduling payload to set unschedulable=true, got %#v", scheduling)
	}

	draining := requests["/runners/runner-1/draining"]
	if draining == nil || !draining["draining"] {
		t.Fatalf("expected draining payload to set draining=true, got %#v", draining)
	}
}

func runnerResourceJSON(id string, unschedulable bool) string {
	if id == "" {
		id = "runner-1"
	}
	payload := map[string]any{
		"id":            id,
		"cpu":           4,
		"memory":        16,
		"disk":          100,
		"region":        "us",
		"name":          "runner",
		"state":         "ready",
		"unschedulable": unschedulable,
		"tags":          []string{"terraform"},
		"createdAt":     "2026-06-11T00:00:00Z",
		"updatedAt":     "2026-06-11T00:00:00Z",
		"version":       "0",
		"apiVersion":    "0",
		"runnerClass":   "container",
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(raw)
}
