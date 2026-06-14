package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAdminActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory      func() action.Action
		requiredAttr string
		boolAttr     string
	}{
		"daytona_admin_set_default_docker_registry": {
			factory:      NewAdminSetDefaultDockerRegistryAction,
			requiredAttr: "registry_id",
		},
		"daytona_admin_set_snapshot_general_status": {
			factory:      NewAdminSetSnapshotGeneralStatusAction,
			requiredAttr: "snapshot_id",
			boolAttr:     "general",
		},
	}

	for expectedTypeName, testCase := range testCases {
		t.Run(expectedTypeName, func(t *testing.T) {
			t.Parallel()

			actionInstance := testCase.factory()

			var metadataResp action.MetadataResponse
			actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != expectedTypeName {
				t.Fatalf("expected type name %q, got %q", expectedTypeName, metadataResp.TypeName)
			}

			var schemaResp action.SchemaResponse
			actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			stringAttr, ok := schemaResp.Schema.Attributes[testCase.requiredAttr].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected %s to be a string attribute, got %T", testCase.requiredAttr, schemaResp.Schema.Attributes[testCase.requiredAttr])
			}
			if !stringAttr.Required {
				t.Fatalf("expected %s to be required", testCase.requiredAttr)
			}

			if testCase.boolAttr != "" {
				boolAttr, ok := schemaResp.Schema.Attributes[testCase.boolAttr].(actionschema.BoolAttribute)
				if !ok {
					t.Fatalf("expected %s to be a bool attribute, got %T", testCase.boolAttr, schemaResp.Schema.Attributes[testCase.boolAttr])
				}
				if !boolAttr.Required {
					t.Fatalf("expected %s to be required", testCase.boolAttr)
				}
			}
		})
	}
}

func TestAdminSetDefaultDockerRegistryActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"registry-1","name":"registry","url":"https://registry.example.com","username":"user","project":"project","registryType":"docker","createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z"}`))
	}))
	defer server.Close()

	actionInstance := NewAdminSetDefaultDockerRegistryAction()
	configureActionClient(t, actionInstance, server.URL)

	config := adminSetDefaultDockerRegistryActionConfig(t, actionInstance, "registry-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/docker-registry/registry-1/set-default" {
		t.Fatalf("expected path %q, got %q", "/admin/docker-registry/registry-1/set-default", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
}

func TestAdminSetSnapshotGeneralStatusActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %s", err)
		}
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("failed unmarshalling request body %q: %s", string(body), err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","general":true,"name":"base","state":"active","size":null,"entrypoint":[],"cpu":2,"gpu":0,"mem":4,"disk":20,"errorReason":null,"createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z","lastUsedAt":null}`))
	}))
	defer server.Close()

	actionInstance := NewAdminSetSnapshotGeneralStatusAction()
	configureActionClient(t, actionInstance, server.URL)

	config := adminSetSnapshotGeneralStatusActionConfig(t, actionInstance, "550e8400-e29b-41d4-a716-446655440000", true)

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected method %s, got %s", http.MethodPatch, gotMethod)
	}
	if gotPath != "/admin/snapshots/550e8400-e29b-41d4-a716-446655440000/general" {
		t.Fatalf("expected path %q, got %q", "/admin/snapshots/550e8400-e29b-41d4-a716-446655440000/general", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
	if gotPayload["general"] != true {
		t.Fatalf("expected general=true payload, got %#v", gotPayload["general"])
	}
}

func adminSetDefaultDockerRegistryActionConfig(t *testing.T, actionInstance action.Action, registryID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"registry_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"registry_id": terraformValue(t, types.StringValue(registryID)),
	})
}

func adminSetSnapshotGeneralStatusActionConfig(t *testing.T, actionInstance action.Action, snapshotID string, general bool) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"snapshot_id": tftypes.String,
		"general":     tftypes.Bool,
	}}, map[string]tftypes.Value{
		"snapshot_id": terraformValue(t, types.StringValue(snapshotID)),
		"general":     terraformValue(t, types.BoolValue(general)),
	})
}
