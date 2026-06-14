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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAdminRunnerResourceSchema(t *testing.T) {
	t.Parallel()

	runnerResource := NewAdminRunnerResource()

	var metadataResp resource.MetadataResponse
	runnerResource.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_admin_runner" {
		t.Fatalf("expected type name %q, got %q", "daytona_admin_runner", metadataResp.TypeName)
	}

	var schemaResp resource.SchemaResponse
	runnerResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	apiKeyAttr, ok := schemaResp.Schema.Attributes["api_key"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected api_key to be a string attribute, got %T", schemaResp.Schema.Attributes["api_key"])
	}
	if !apiKeyAttr.Required || !apiKeyAttr.Sensitive {
		t.Fatal("expected api_key to be required and sensitive")
	}

	unschedulableAttr, ok := schemaResp.Schema.Attributes["unschedulable"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected unschedulable to be a bool attribute, got %T", schemaResp.Schema.Attributes["unschedulable"])
	}
	if !unschedulableAttr.Optional || !unschedulableAttr.Computed {
		t.Fatal("expected unschedulable to be optional and computed")
	}
}

func TestAdminRunnerResourceCreateRequest(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotOrganization, gotUserAgent string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotOrganization = r.Header.Get(organizationHeader)
		gotUserAgent = r.Header.Get("User-Agent")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %s", err)
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("failed unmarshalling request body %q: %s", string(body), err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"runner-1","apiKey":"runner-secret"}`))
	}))
	defer server.Close()

	runnerResource := &AdminRunnerResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	data := adminRunnerResourceModel{
		RegionID:   types.StringValue("region-1"),
		Name:       types.StringValue("runner-1"),
		APIKey:     types.StringValue("runner-secret"),
		APIVersion: types.StringValue("2"),
		Domain:     types.StringValue("runner.example.com"),
		APIURL:     types.StringValue("https://api.runner.example.com"),
		ProxyURL:   types.StringValue("https://proxy.runner.example.com"),
		CPU:        types.Float64Value(8),
		MemoryGiB:  types.Float64Value(16),
		DiskGiB:    types.Float64Value(100),
		Tags:       listStringValue(context.Background(), []string{"terraform"}),
	}

	created, _, err := runnerResource.createAdminRunner(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected create error: %s", err)
	}
	if created.Id != "runner-1" {
		t.Fatalf("expected created runner ID %q, got %q", "runner-1", created.Id)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/runners" {
		t.Fatalf("expected path %q, got %q", "/admin/runners", gotPath)
	}
	if gotAuthorization != "Bearer test-token" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotOrganization != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	if gotBody["regionId"] != "region-1" || gotBody["name"] != "runner-1" || gotBody["apiKey"] != "runner-secret" || gotBody["apiVersion"] != "2" {
		t.Fatalf("unexpected create payload: %#v", gotBody)
	}
	if gotBody["domain"] != "runner.example.com" || gotBody["apiUrl"] != "https://api.runner.example.com" || gotBody["proxyUrl"] != "https://proxy.runner.example.com" {
		t.Fatalf("unexpected runner URL payload: %#v", gotBody)
	}
	if gotBody["cpu"] != float64(8) || gotBody["memoryGiB"] != float64(16) || gotBody["diskGiB"] != float64(100) {
		t.Fatalf("unexpected capacity payload: %#v", gotBody)
	}
}

func TestAdminRunnerResourceSchedulingPatchRequest(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotOrganization, gotUserAgent string
	var gotPayload map[string]bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotOrganization = r.Header.Get(organizationHeader)
		gotUserAgent = r.Header.Get("User-Agent")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %s", err)
		}
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("failed unmarshalling request body %q: %s", string(body), err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	runnerResource := &AdminRunnerResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	if _, err := runnerResource.updateAdminRunnerScheduling(context.Background(), "runner/1", true); err != nil {
		t.Fatalf("unexpected scheduling update error: %s", err)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected method %s, got %s", http.MethodPatch, gotMethod)
	}
	if gotPath != "/admin/runners/runner%2F1/scheduling" {
		t.Fatalf("expected path %q, got %q", "/admin/runners/runner%2F1/scheduling", gotPath)
	}
	if gotAuthorization != "Bearer test-token" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotOrganization != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
	if !gotPayload["unschedulable"] {
		t.Fatalf("expected unschedulable payload, got %#v", gotPayload)
	}
}

func TestAdminCreateRunnerPayloadRejectsInvalidAPIVersion(t *testing.T) {
	t.Parallel()

	_, err := adminCreateRunnerPayload(context.Background(), adminRunnerResourceModel{
		RegionID:   types.StringValue("region-1"),
		Name:       types.StringValue("runner-1"),
		APIKey:     types.StringValue("runner-secret"),
		APIVersion: types.StringValue("1"),
	})
	if err == nil {
		t.Fatal("expected invalid api_version error")
	}
}
