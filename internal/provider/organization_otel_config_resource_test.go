package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestExpandOrganizationOtelConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	withHeaders := organizationOtelConfigResourceModel{
		Endpoint: types.StringValue("https://otel.example.com/v1/traces"),
		Headers:  stringMapValue(ctx, map[string]string{"Authorization": "Bearer secret"}),
	}
	var diags diag.Diagnostics
	config, ok := expandOrganizationOtelConfig(ctx, withHeaders, &diags)
	if !ok || diags.HasError() {
		t.Fatalf("unexpected expand diagnostics: %s", diags)
	}
	if config.Endpoint != "https://otel.example.com/v1/traces" {
		t.Fatalf("expected endpoint %q, got %q", "https://otel.example.com/v1/traces", config.Endpoint)
	}
	if config.Headers["Authorization"] != "Bearer secret" {
		t.Fatalf("expected Authorization header %q, got %#v", "Bearer secret", config.Headers)
	}

	withoutHeaders := organizationOtelConfigResourceModel{
		Endpoint: types.StringValue("https://otel.example.com/v1/traces"),
		Headers:  types.MapNull(types.StringType),
	}
	var nullDiags diag.Diagnostics
	config, ok = expandOrganizationOtelConfig(ctx, withoutHeaders, &nullDiags)
	if !ok || nullDiags.HasError() {
		t.Fatalf("unexpected expand diagnostics: %s", nullDiags)
	}
	if len(config.Headers) != 0 {
		t.Fatalf("expected null headers to be omitted, got %#v", config.Headers)
	}
}

func TestFlattenOrganizationOtelConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := apiclient.NewOtelConfig("https://otel.example.com/v1/traces")
	config.SetHeaders(map[string]string{"Authorization": "Bearer secret"})

	prior := organizationOtelConfigResourceModel{
		ID:             types.StringUnknown(),
		OrganizationID: types.StringValue("org-1"),
	}
	flattened := flattenOrganizationOtelConfig(ctx, config, prior)

	if flattened.ID.ValueString() != "org-1" {
		t.Fatalf("expected ID to mirror organization_id org-1, got %q", flattened.ID.ValueString())
	}
	if flattened.Endpoint.ValueString() != "https://otel.example.com/v1/traces" {
		t.Fatalf("expected endpoint %q, got %q", "https://otel.example.com/v1/traces", flattened.Endpoint.ValueString())
	}
	headers := map[string]string{}
	diags := flattened.Headers.ElementsAs(ctx, &headers, false)
	if diags.HasError() {
		t.Fatalf("unexpected headers diagnostics: %s", diags)
	}
	if headers["Authorization"] != "Bearer secret" {
		t.Fatalf("expected Authorization header %q, got %#v", "Bearer secret", headers)
	}

	priorHeaders := stringMapValue(ctx, map[string]string{"X-Custom": "kept"})
	withoutHeaders := flattenOrganizationOtelConfig(ctx, apiclient.NewOtelConfig("https://otel.example.com"), organizationOtelConfigResourceModel{
		OrganizationID: types.StringValue("org-1"),
		Headers:        priorHeaders,
	})
	if !withoutHeaders.Headers.Equal(priorHeaders) {
		t.Fatalf("expected prior headers to be kept when the API omits them, got %#v", withoutHeaders.Headers)
	}

	priorEndpoint := types.StringValue("untouched")
	unchanged := flattenOrganizationOtelConfig(ctx, nil, organizationOtelConfigResourceModel{Endpoint: priorEndpoint})
	if unchanged.Endpoint != priorEndpoint {
		t.Fatalf("expected nil config to leave prior model unchanged, got %#v", unchanged.Endpoint)
	}
}

func TestOrganizationOtelConfigResourceCreateRequests(t *testing.T) {
	t.Parallel()

	var updatePayload map[string]any
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(organizationHeader) != "org-1" {
			t.Fatalf("expected organization header %q, got %q", "org-1", r.Header.Get(organizationHeader))
		}

		path := r.URL.EscapedPath()
		requests[r.Method+" "+path]++
		switch {
		case r.Method == http.MethodPut && path == "/organizations/org-1/otel-config":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed reading body: %s", err)
			}
			if err := json.Unmarshal(body, &updatePayload); err != nil {
				t.Fatalf("failed unmarshalling update payload %q: %s", string(body), err)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && path == "/organizations/org-1/otel-config":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"endpoint":"https://otel.example.com/v1/traces","headers":{"Authorization":"Bearer secret"}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	otelResource := &OrganizationOtelConfigResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, otelResource, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"endpoint":        tftypes.NewValue(tftypes.String, "https://otel.example.com/v1/traces"),
		"headers": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"Authorization": tftypes.NewValue(tftypes.String, "Bearer secret"),
		}),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	otelResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}

	for _, key := range []string{
		"PUT /organizations/org-1/otel-config",
		"GET /organizations/org-1/otel-config",
	} {
		if requests[key] != 1 {
			t.Fatalf("expected one %s request, got %d", key, requests[key])
		}
	}

	if updatePayload["endpoint"] != "https://otel.example.com/v1/traces" {
		t.Fatalf("expected payload endpoint %q, got %#v", "https://otel.example.com/v1/traces", updatePayload["endpoint"])
	}
	headers, ok := updatePayload["headers"].(map[string]any)
	if !ok || headers["Authorization"] != "Bearer secret" {
		t.Fatalf("expected payload Authorization header %q, got %#v", "Bearer secret", updatePayload["headers"])
	}

	var data organizationOtelConfigResourceModel
	createResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "org-1" {
		t.Fatalf("expected state ID org-1, got %q", data.ID.ValueString())
	}
	if data.Endpoint.ValueString() != "https://otel.example.com/v1/traces" {
		t.Fatalf("expected state endpoint from read response, got %q", data.Endpoint.ValueString())
	}
}
