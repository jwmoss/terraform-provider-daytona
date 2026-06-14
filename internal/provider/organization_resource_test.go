package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func resourceTestSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}
	return schemaResp.Schema
}

// resourceTestObject builds a full terraform object value for a resource schema,
// filling every attribute missing from values with null so framework request
// types can round-trip the model.
func resourceTestObject(t *testing.T, resourceSchema schema.Schema, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	objectType, ok := resourceSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("expected resource schema object type, got %T", resourceSchema.Type().TerraformType(context.Background()))
	}

	for name := range values {
		if _, ok := objectType.AttributeTypes[name]; !ok {
			t.Fatalf("unknown attribute %q in test values", name)
		}
	}

	allValues := map[string]tftypes.Value{}
	for name, attributeType := range objectType.AttributeTypes {
		value, ok := values[name]
		if !ok {
			value = tftypes.NewValue(attributeType, nil)
		}
		allValues[name] = value
	}
	return tftypes.NewValue(objectType, allValues)
}

func resourceTestState(t *testing.T, resourceSchema schema.Schema, values map[string]tftypes.Value) tfsdk.State {
	t.Helper()
	return tfsdk.State{Raw: resourceTestObject(t, resourceSchema, values), Schema: resourceSchema}
}

func resourceTestPlan(t *testing.T, resourceSchema schema.Schema, values map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()
	return tfsdk.Plan{Raw: resourceTestObject(t, resourceSchema, values), Schema: resourceSchema}
}

func tftypesStringSet(values ...string) tftypes.Value {
	elements := make([]tftypes.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, tftypes.NewValue(tftypes.String, value))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elements)
}

type resourceReadContractCase struct {
	name        string
	statusCode  int
	body        string
	wantError   bool
	wantRemoved bool
}

// runResourceReadContractTest verifies the Read not-found-versus-error contract:
// only an HTTP 404 or a successful lookup without the object may remove state.
// A transient API failure must surface as an error diagnostic, because removing
// state would make Terraform re-create an object that still exists remotely.
func runResourceReadContractTest(t *testing.T, readPath string, newResource func(*daytonaClient) resource.Resource, stateValues map[string]tftypes.Value, testCases []resourceReadContractCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.EscapedPath() != readPath {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(testCase.statusCode)
				_, _ = w.Write([]byte(testCase.body))
			}))
			defer server.Close()

			testResource := newResource(newDaytonaClient(server.URL, "test-token", "org-1", "test"))
			state := resourceTestState(t, resourceTestSchema(t, testResource), stateValues)

			readResp := resource.ReadResponse{State: state}
			testResource.Read(context.Background(), resource.ReadRequest{State: state}, &readResp)

			if readResp.Diagnostics.HasError() != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics: %s", testCase.wantError, readResp.Diagnostics)
			}
			if readResp.State.Raw.IsNull() != testCase.wantRemoved {
				t.Fatalf("expected state removed=%t, got state %s", testCase.wantRemoved, readResp.State.Raw.String())
			}
		})
	}
}

func TestOrganizationResourceReadContract(t *testing.T) {
	t.Parallel()

	runResourceReadContractTest(t, "/organizations/org-1",
		func(client *daytonaClient) resource.Resource {
			return &OrganizationResource{client: client}
		},
		map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.String, "org-1"),
		},
		[]resourceReadContractCase{
			{name: "api error keeps state", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantError: true},
			{name: "not found removes state", statusCode: http.StatusNotFound, body: `{"message":"missing"}`, wantRemoved: true},
			{name: "found refreshes state", statusCode: http.StatusOK, body: organizationJSON(4)},
		})
}

func TestFlattenOrganization(t *testing.T) {
	t.Parallel()

	var organization apiclient.Organization
	if err := json.Unmarshal([]byte(organizationJSON(4)), &organization); err != nil {
		t.Fatalf("failed unmarshalling organization fixture: %s", err)
	}

	data := flattenOrganization(&organization, organizationResourceModel{})

	if data.ID.ValueString() != "org-1" {
		t.Fatalf("expected ID org-1, got %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "engineering" {
		t.Fatalf("expected name engineering, got %q", data.Name.ValueString())
	}
	if data.CreatedBy.ValueString() != "user-1" {
		t.Fatalf("expected created_by user-1, got %q", data.CreatedBy.ValueString())
	}
	if data.DefaultRegionID.ValueString() != "region-1" {
		t.Fatalf("expected default_region_id region-1, got %q", data.DefaultRegionID.ValueString())
	}
	// Empty API strings and zero timestamps must map to null, not "", so
	// unsuspended organizations never show a suspension diff.
	if !data.SuspensionReason.IsNull() {
		t.Fatalf("expected null suspension_reason, got %q", data.SuspensionReason.ValueString())
	}
	if !data.SuspendedAt.IsNull() {
		t.Fatalf("expected null suspended_at, got %q", data.SuspendedAt.ValueString())
	}
	// Nullable rate limits must distinguish unset (null) from configured values.
	if !data.AuthenticatedRateLimit.IsNull() {
		t.Fatalf("expected null authenticated_rate_limit, got %f", data.AuthenticatedRateLimit.ValueFloat64())
	}
	if data.SandboxCreateRateLimit.ValueFloat64() != 10 {
		t.Fatalf("expected sandbox_create_rate_limit 10, got %f", data.SandboxCreateRateLimit.ValueFloat64())
	}
	if data.MaxCPUPerSandbox.ValueFloat64() != 4 {
		t.Fatalf("expected max_cpu_per_sandbox 4, got %f", data.MaxCPUPerSandbox.ValueFloat64())
	}
	if !data.SandboxLimitedNetworkEgress.ValueBool() {
		t.Fatal("expected sandbox_limited_network_egress true")
	}
	if data.ExperimentalConfigJSON.ValueString() != `{"flag":true}` {
		t.Fatalf("expected experimental config JSON string, got %q", data.ExperimentalConfigJSON.ValueString())
	}
	if data.CreatedAt.ValueString() != "2026-06-10T00:00:00Z" {
		t.Fatalf("expected RFC3339 created_at, got %q", data.CreatedAt.ValueString())
	}
}

func TestOrganizationResourceCreatePersistsStateBeforeSettings(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		requests[r.Method+" "+path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && path == "/organizations":
			_, _ = w.Write([]byte(organizationJSON(4)))
		case r.Method == http.MethodPatch && path == "/organizations/org-1/quota":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"quota service unavailable"}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	organizationResource := &OrganizationResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	organizationSchema := resourceTestSchema(t, organizationResource)
	plan := resourceTestPlan(t, organizationSchema, map[string]tftypes.Value{
		"name":                tftypes.NewValue(tftypes.String, "engineering"),
		"default_region_id":   tftypes.NewValue(tftypes.String, "region-1"),
		"max_cpu_per_sandbox": tftypes.NewValue(tftypes.Number, 4),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: organizationSchema}}
	organizationResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)

	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected quota failure diagnostics")
	}
	// The organization exists remotely as soon as POST succeeds; state must
	// already be persisted so a failed follow-up settings call cannot orphan it.
	if createResp.State.Raw.IsNull() {
		t.Fatal("expected created organization to be persisted to state despite quota failure")
	}

	var persisted organizationResourceModel
	if diags := createResp.State.Get(context.Background(), &persisted); diags.HasError() {
		t.Fatalf("unexpected state get diagnostics: %s", diags)
	}
	if persisted.ID.ValueString() != "org-1" {
		t.Fatalf("expected persisted organization ID org-1, got %q", persisted.ID.ValueString())
	}

	if requests["POST /organizations"] != 1 {
		t.Fatalf("expected one create request, got %d", requests["POST /organizations"])
	}
	// A PATCH is non-idempotent, so a 500 from the quota endpoint is not retried;
	// the failure surfaces after one request and state must still be persisted.
	if got := requests["PATCH /organizations/org-1/quota"]; got != 1 {
		t.Fatalf("expected one quota request, got %d", got)
	}
}

func TestOrganizationResourceCreateAppliesSettings(t *testing.T) {
	t.Parallel()

	var createPayload, quotaPayload, egressPayload, experimentalPayload map[string]any
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(organizationHeader) != "org-1" {
			t.Errorf("expected organization header %q, got %q", "org-1", r.Header.Get(organizationHeader))
		}

		path := r.URL.EscapedPath()
		requests[r.Method+" "+path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && path == "/organizations":
			decodeTestPayload(t, r.Body, &createPayload)
			_, _ = w.Write([]byte(organizationJSON(4)))
		case r.Method == http.MethodPatch && path == "/organizations/org-1/quota":
			decodeTestPayload(t, r.Body, &quotaPayload)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && path == "/organizations/org-1/sandbox-default-limited-network-egress":
			decodeTestPayload(t, r.Body, &egressPayload)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && path == "/organizations/org-1/experimental-config":
			decodeTestPayload(t, r.Body, &experimentalPayload)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && path == "/organizations/org-1":
			_, _ = w.Write([]byte(organizationJSON(6)))
		default:
			t.Errorf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	organizationResource := &OrganizationResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	organizationSchema := resourceTestSchema(t, organizationResource)
	plan := resourceTestPlan(t, organizationSchema, map[string]tftypes.Value{
		"name":                           tftypes.NewValue(tftypes.String, "engineering"),
		"default_region_id":              tftypes.NewValue(tftypes.String, "region-1"),
		"max_cpu_per_sandbox":            tftypes.NewValue(tftypes.Number, 4),
		"sandbox_limited_network_egress": tftypes.NewValue(tftypes.Bool, true),
		"experimental_config_json":       tftypes.NewValue(tftypes.String, `{"flag":true}`),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: organizationSchema}}
	organizationResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}

	if createPayload["name"] != "engineering" || createPayload["defaultRegionId"] != "region-1" {
		t.Fatalf("expected create payload name/defaultRegionId, got %#v", createPayload)
	}
	if quotaPayload["maxCpuPerSandbox"] != float64(4) {
		t.Fatalf("expected quota maxCpuPerSandbox 4, got %#v", quotaPayload["maxCpuPerSandbox"])
	}
	if egressPayload["sandboxDefaultLimitedNetworkEgress"] != true {
		t.Fatalf("expected egress payload true, got %#v", egressPayload)
	}
	if experimentalPayload["flag"] != true {
		t.Fatalf("expected experimental payload flag true, got %#v", experimentalPayload)
	}

	var persisted organizationResourceModel
	if diags := createResp.State.Get(context.Background(), &persisted); diags.HasError() {
		t.Fatalf("unexpected state get diagnostics: %s", diags)
	}
	// The final read-back wins so computed attributes reflect server truth
	// after the settings calls, not the initial create response.
	if persisted.MaxCPUPerSandbox.ValueFloat64() != 6 {
		t.Fatalf("expected max_cpu_per_sandbox 6 from final read, got %f", persisted.MaxCPUPerSandbox.ValueFloat64())
	}

	for _, key := range []string{
		"POST /organizations",
		"PATCH /organizations/org-1/quota",
		"POST /organizations/org-1/sandbox-default-limited-network-egress",
		"PUT /organizations/org-1/experimental-config",
		"GET /organizations/org-1",
	} {
		if requests[key] != 1 {
			t.Fatalf("expected one %s request, got %d", key, requests[key])
		}
	}
}

func TestExperimentalConfigPayloadValidation(t *testing.T) {
	t.Parallel()

	var invalidDiags diag.Diagnostics
	if _, ok := experimentalConfigPayload(types.StringValue("{"), &invalidDiags); ok || !invalidDiags.HasError() {
		t.Fatal("expected malformed JSON to fail validation")
	}

	// "null" parses as JSON but is not an object; sending it would clear the
	// config without the user expressing that with {}.
	var nullDiags diag.Diagnostics
	if _, ok := experimentalConfigPayload(types.StringValue("null"), &nullDiags); ok || !nullDiags.HasError() {
		t.Fatal("expected JSON null to fail validation")
	}

	var validDiags diag.Diagnostics
	payload, ok := experimentalConfigPayload(types.StringValue(`{"flag":true}`), &validDiags)
	if !ok || validDiags.HasError() {
		t.Fatalf("unexpected payload diagnostics: %s", validDiags)
	}
	if payload["flag"] != true {
		t.Fatalf("expected payload flag true, got %#v", payload)
	}
}

func decodeTestPayload(t *testing.T, body io.Reader, target *map[string]any) {
	t.Helper()

	raw, err := io.ReadAll(body)
	if err != nil {
		t.Errorf("failed reading request body: %s", err)
		return
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Errorf("failed unmarshalling request body %q: %s", string(raw), err)
	}
}

func organizationJSON(maxCPUPerSandbox float64) string {
	return fmt.Sprintf(`{
		"id": "org-1",
		"name": "engineering",
		"createdBy": "user-1",
		"personal": false,
		"createdAt": "2026-06-10T00:00:00Z",
		"updatedAt": "2026-06-11T00:00:00Z",
		"suspended": false,
		"suspendedAt": null,
		"suspensionReason": "",
		"suspendedUntil": null,
		"suspensionCleanupGracePeriodHours": 24,
		"maxCpuPerSandbox": %g,
		"maxMemoryPerSandbox": 8,
		"maxDiskPerSandbox": 50,
		"snapshotDeactivationTimeoutMinutes": 30,
		"sandboxLimitedNetworkEgress": true,
		"defaultRegionId": "region-1",
		"authenticatedRateLimit": null,
		"sandboxCreateRateLimit": 10,
		"sandboxLifecycleRateLimit": null,
		"experimentalConfig": {"flag": true},
		"otelConfig": null,
		"authenticatedRateLimitTtlSeconds": null,
		"sandboxCreateRateLimitTtlSeconds": 60,
		"sandboxLifecycleRateLimitTtlSeconds": null
	}`, maxCPUPerSandbox)
}
