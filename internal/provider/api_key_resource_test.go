package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// resourcePlan builds a tfsdk.Plan for a resource from explicit attribute
// values, defaulting every remaining attribute to unknown the way Terraform
// plans computed attributes before create.
func resourcePlan(t *testing.T, r resource.Resource, values map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	objectType, ok := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("expected schema terraform type to be an object, got %T", schemaResp.Schema.Type().TerraformType(context.Background()))
	}

	planValues := map[string]tftypes.Value{}
	for name, attributeType := range objectType.AttributeTypes {
		if value, present := values[name]; present {
			planValues[name] = value
		} else {
			planValues[name] = tftypes.NewValue(attributeType, tftypes.UnknownValue)
		}
	}

	return tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objectType, planValues)}
}

func TestFlattenAPIKeyResponse(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	apiKey := apiclient.NewApiKeyResponse(
		"key-1",
		"dtn_real_secret",
		createdAt,
		[]string{"read:sandboxes", "write:registries"},
		*apiclient.NewNullableTime(&expiresAt),
	)

	prior := apiKeyResourceModel{
		ID:        types.StringUnknown(),
		Value:     types.StringUnknown(),
		ExpiresAt: types.StringUnknown(),
	}
	flattened := flattenAPIKeyResponse(apiKey, prior)

	if flattened.ID.ValueString() != "key-1" {
		t.Fatalf("expected ID %q, got %q", "key-1", flattened.ID.ValueString())
	}
	if flattened.Name.ValueString() != "key-1" {
		t.Fatalf("expected name %q, got %q", "key-1", flattened.Name.ValueString())
	}
	if flattened.Value.ValueString() != "dtn_real_secret" {
		t.Fatalf("expected create-time value %q, got %q", "dtn_real_secret", flattened.Value.ValueString())
	}
	if flattened.CreatedAt.ValueString() != "2026-06-01T12:00:00Z" {
		t.Fatalf("expected created_at %q, got %q", "2026-06-01T12:00:00Z", flattened.CreatedAt.ValueString())
	}
	if flattened.ExpiresAt.ValueString() != "2026-12-31T23:59:59Z" {
		t.Fatalf("expected expires_at %q, got %q", "2026-12-31T23:59:59Z", flattened.ExpiresAt.ValueString())
	}

	permissions := []string{}
	diags := flattened.Permissions.ElementsAs(context.Background(), &permissions, false)
	if diags.HasError() {
		t.Fatalf("unexpected permissions diagnostics: %s", diags)
	}
	if len(permissions) != 2 || permissions[0] != "read:sandboxes" || permissions[1] != "write:registries" {
		t.Fatalf("expected permissions [read:sandboxes write:registries], got %#v", permissions)
	}

	withoutExpiry := apiclient.NewApiKeyResponse("key-1", "dtn_real_secret", createdAt, nil, apiclient.NullableTime{})
	flattened = flattenAPIKeyResponse(withoutExpiry, apiKeyResourceModel{ExpiresAt: types.StringUnknown()})
	if !flattened.ExpiresAt.IsNull() {
		t.Fatalf("expected unknown expires_at to flatten to null, got %#v", flattened.ExpiresAt)
	}

	priorName := types.StringValue("untouched")
	unchanged := flattenAPIKeyResponse(nil, apiKeyResourceModel{Name: priorName})
	if unchanged.Name != priorName {
		t.Fatalf("expected nil response to leave prior model unchanged, got %#v", unchanged.Name)
	}
}

// flattenAPIKeyList must never copy the list/get endpoint's Value into state:
// that endpoint masks the key, and the real key is only returned once at
// create time via flattenAPIKeyResponse. Copying the masked value would
// silently replace a working credential in state with a non-functional string.
func TestFlattenAPIKeyListNeverStoresMaskedValue(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	apiKey := apiclient.NewApiKeyList(
		"key-1",
		"dtn_****masked",
		createdAt,
		[]string{"read:sandboxes"},
		apiclient.NullableTime{},
		apiclient.NullableTime{},
		"user-1",
	)

	imported := flattenAPIKeyList(apiKey, apiKeyResourceModel{Value: types.StringNull()})
	if !imported.Value.IsNull() {
		t.Fatalf("expected null prior value to stay null after flattening, got %q", imported.Value.ValueString())
	}

	unknown := flattenAPIKeyList(apiKey, apiKeyResourceModel{Value: types.StringUnknown()})
	if !unknown.Value.IsNull() {
		t.Fatalf("expected unknown prior value to flatten to null, got %#v", unknown.Value)
	}

	stored := flattenAPIKeyList(apiKey, apiKeyResourceModel{Value: types.StringValue("dtn_real_secret")})
	if stored.Value.ValueString() != "dtn_real_secret" {
		t.Fatalf("expected stored real value to be kept, got %q", stored.Value.ValueString())
	}
}

func TestFlattenAPIKeyListFieldMapping(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	lastUsedAt := time.Date(2026, 6, 10, 8, 30, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	apiKey := apiclient.NewApiKeyList(
		"key-1",
		"dtn_****masked",
		createdAt,
		[]string{"read:sandboxes"},
		*apiclient.NewNullableTime(&lastUsedAt),
		*apiclient.NewNullableTime(&expiresAt),
		"user-1",
	)

	flattened := flattenAPIKeyList(apiKey, apiKeyResourceModel{ExpiresAt: types.StringUnknown()})

	if flattened.ID.ValueString() != "key-1" || flattened.Name.ValueString() != "key-1" {
		t.Fatalf("expected ID and name %q, got %q and %q", "key-1", flattened.ID.ValueString(), flattened.Name.ValueString())
	}
	if flattened.CreatedAt.ValueString() != "2026-06-01T12:00:00Z" {
		t.Fatalf("expected created_at %q, got %q", "2026-06-01T12:00:00Z", flattened.CreatedAt.ValueString())
	}
	if flattened.UserID.ValueString() != "user-1" {
		t.Fatalf("expected user_id %q, got %q", "user-1", flattened.UserID.ValueString())
	}
	if flattened.LastUsedAt.ValueString() != "2026-06-10T08:30:00Z" {
		t.Fatalf("expected last_used_at %q, got %q", "2026-06-10T08:30:00Z", flattened.LastUsedAt.ValueString())
	}
	if flattened.ExpiresAt.ValueString() != "2026-12-31T23:59:59Z" {
		t.Fatalf("expected expires_at %q, got %q", "2026-12-31T23:59:59Z", flattened.ExpiresAt.ValueString())
	}

	permissions := []string{}
	diags := flattened.Permissions.ElementsAs(context.Background(), &permissions, false)
	if diags.HasError() {
		t.Fatalf("unexpected permissions diagnostics: %s", diags)
	}
	if len(permissions) != 1 || permissions[0] != "read:sandboxes" {
		t.Fatalf("expected permissions [read:sandboxes], got %#v", permissions)
	}

	withoutTimestamps := apiclient.NewApiKeyList("key-1", "dtn_****masked", createdAt, nil, apiclient.NullableTime{}, apiclient.NullableTime{}, "user-1")
	flattened = flattenAPIKeyList(withoutTimestamps, apiKeyResourceModel{
		LastUsedAt: types.StringValue("stale"),
		ExpiresAt:  types.StringUnknown(),
	})
	if !flattened.LastUsedAt.IsNull() {
		t.Fatalf("expected unset last_used_at to flatten to null, got %#v", flattened.LastUsedAt)
	}
	if !flattened.ExpiresAt.IsNull() {
		t.Fatalf("expected unset expires_at to flatten to null, got %#v", flattened.ExpiresAt)
	}

	priorName := types.StringValue("untouched")
	unchanged := flattenAPIKeyList(nil, apiKeyResourceModel{Name: priorName})
	if unchanged.Name != priorName {
		t.Fatalf("expected nil API key to leave prior model unchanged, got %#v", unchanged.Name)
	}
}

func TestAPIKeyResourceCreateRequest(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var createPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
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
		if err := json.Unmarshal(body, &createPayload); err != nil {
			t.Fatalf("failed unmarshalling create payload %q: %s", string(body), err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "key-1",
			"value": "dtn_real_secret",
			"createdAt": "2026-06-01T12:00:00Z",
			"permissions": ["read:sandboxes"],
			"expiresAt": "2026-12-31T23:59:59Z"
		}`))
	}))
	defer server.Close()

	apiKeyResource := &APIKeyResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, apiKeyResource, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "key-1"),
		"permissions": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "read:sandboxes"),
		}),
		"expires_at": tftypes.NewValue(tftypes.String, "2026-12-31T23:59:59Z"),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	apiKeyResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/api-keys" {
		t.Fatalf("expected path %q, got %q", "/api-keys", gotPath)
	}
	if createPayload["name"] != "key-1" {
		t.Fatalf("expected payload name key-1, got %#v", createPayload["name"])
	}
	permissions, ok := createPayload["permissions"].([]any)
	if !ok || len(permissions) != 1 || permissions[0] != "read:sandboxes" {
		t.Fatalf("expected payload permissions [read:sandboxes], got %#v", createPayload["permissions"])
	}
	if createPayload["expiresAt"] != "2026-12-31T23:59:59Z" {
		t.Fatalf("expected payload expiresAt 2026-12-31T23:59:59Z, got %#v", createPayload["expiresAt"])
	}

	var data apiKeyResourceModel
	createResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "key-1" {
		t.Fatalf("expected state ID key-1, got %q", data.ID.ValueString())
	}
	if data.Value.ValueString() != "dtn_real_secret" {
		t.Fatalf("expected state value from create response, got %q", data.Value.ValueString())
	}
	if data.CreatedAt.ValueString() != "2026-06-01T12:00:00Z" {
		t.Fatalf("expected state created_at 2026-06-01T12:00:00Z, got %q", data.CreatedAt.ValueString())
	}
}

func TestAPIKeyResourceCreateRejectsInvalidExpiresAt(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request %s %s for invalid expires_at", r.Method, r.URL.EscapedPath())
	}))
	defer server.Close()

	apiKeyResource := &APIKeyResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, apiKeyResource, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "key-1"),
		"permissions": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "read:sandboxes"),
		}),
		"expires_at": tftypes.NewValue(tftypes.String, "tomorrow"),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	apiKeyResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for non-RFC3339 expires_at")
	}
}
