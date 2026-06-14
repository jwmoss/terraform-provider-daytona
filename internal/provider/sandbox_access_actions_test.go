package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSandboxAccessActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory          func() action.Action
		expectedTypeName string
		expectPort       bool
		tokenRequired    bool
		tokenOptional    bool
	}{
		"expire signed port preview URL": {
			factory:          NewSandboxExpireSignedPortPreviewURLAction,
			expectedTypeName: "daytona_expire_sandbox_signed_port_preview_url",
			expectPort:       true,
			tokenRequired:    true,
		},
		"revoke SSH access": {
			factory:          NewSandboxRevokeSSHAccessAction,
			expectedTypeName: "daytona_revoke_sandbox_ssh_access",
			tokenOptional:    true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actionInstance := testCase.factory()

			var metadataResp action.MetadataResponse
			actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != testCase.expectedTypeName {
				t.Fatalf("expected type name %q, got %q", testCase.expectedTypeName, metadataResp.TypeName)
			}

			var schemaResp action.SchemaResponse
			actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			sandboxAttr, ok := schemaResp.Schema.Attributes["sandbox_id_or_name"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected sandbox_id_or_name to be a string attribute, got %T", schemaResp.Schema.Attributes["sandbox_id_or_name"])
			}
			if !sandboxAttr.Required {
				t.Fatal("expected sandbox_id_or_name to be required")
			}

			tokenAttr, ok := schemaResp.Schema.Attributes["token"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected token to be a string attribute, got %T", schemaResp.Schema.Attributes["token"])
			}
			if tokenAttr.Required != testCase.tokenRequired {
				t.Fatalf("expected token required=%t, got %t", testCase.tokenRequired, tokenAttr.Required)
			}
			if tokenAttr.Optional != testCase.tokenOptional {
				t.Fatalf("expected token optional=%t, got %t", testCase.tokenOptional, tokenAttr.Optional)
			}
			if !tokenAttr.WriteOnly {
				t.Fatal("expected token to be write-only")
			}

			if testCase.expectPort {
				portAttr, ok := schemaResp.Schema.Attributes["port"].(actionschema.Int64Attribute)
				if !ok {
					t.Fatalf("expected port to be an int64 attribute, got %T", schemaResp.Schema.Attributes["port"])
				}
				if !portAttr.Required {
					t.Fatal("expected port to be required")
				}
			}

			organizationAttr, ok := schemaResp.Schema.Attributes["organization_id"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected organization_id to be a string attribute, got %T", schemaResp.Schema.Attributes["organization_id"])
			}
			if !organizationAttr.Optional {
				t.Fatal("expected organization_id to be optional")
			}
		})
	}
}

func TestSandboxExpireSignedPortPreviewURLActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	actionInstance := NewSandboxExpireSignedPortPreviewURLAction()
	configureActionClient(t, actionInstance, server.URL)

	config := expireSignedPortPreviewURLActionConfig(t, actionInstance, "sandbox-1", 3000, "preview-token", "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}

	expectedPath := "/sandbox/sandbox-1/ports/3000/signed-preview-url/preview-token/expire"
	if gotPath != expectedPath {
		t.Fatalf("expected path %q, got %q", expectedPath, gotPath)
	}

	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestSandboxRevokeSSHAccessActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotToken, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotToken = r.URL.Query().Get("token")
		gotOrganizationID = r.Header.Get(organizationHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sandbox-1","organizationId":"org-1","name":"sandbox-1","user":"user-1","env":{},"labels":{},"public":false,"networkBlockAll":false,"target":"region-1","cpu":1,"gpu":0,"memory":2,"disk":10,"toolboxProxyUrl":"https://toolbox.example.com"}`))
	}))
	defer server.Close()

	actionInstance := NewSandboxRevokeSSHAccessAction()
	configureActionClient(t, actionInstance, server.URL)

	config := revokeSSHAccessActionConfig(t, actionInstance, "sandbox-1", "ssh-token", "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("expected method %s, got %s", http.MethodDelete, gotMethod)
	}

	if gotPath != "/sandbox/sandbox-1/ssh-access" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox-1/ssh-access", gotPath)
	}

	if gotToken != "ssh-token" {
		t.Fatalf("expected token query value %q, got %q", "ssh-token", gotToken)
	}

	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestInt32Port(t *testing.T) {
	t.Parallel()

	testCases := map[int64]bool{
		0:     false,
		1:     true,
		3000:  true,
		65535: true,
		65536: false,
	}

	for port, valid := range testCases {
		t.Run(fmt.Sprintf("%d", port), func(t *testing.T) {
			t.Parallel()

			_, ok := int32Port(port)
			if ok != valid {
				t.Fatalf("expected valid=%t, got %t", valid, ok)
			}
		})
	}
}

func configureActionClient(t *testing.T, actionInstance action.Action, apiURL string) {
	t.Helper()

	configurable, ok := actionInstance.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("expected action to implement ActionWithConfigure")
	}

	var configureResp action.ConfigureResponse
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: newDaytonaClient(apiURL, "test-key", "", "test")}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %s", configureResp.Diagnostics)
	}
}

func expireSignedPortPreviewURLActionConfig(t *testing.T, actionInstance action.Action, sandboxIDOrName string, port int64, token, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id_or_name": tftypes.String,
		"port":               tftypes.Number,
		"token":              tftypes.String,
		"organization_id":    tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id_or_name": terraformValue(t, types.StringValue(sandboxIDOrName)),
		"port":               terraformValue(t, types.Int64Value(port)),
		"token":              terraformValue(t, types.StringValue(token)),
		"organization_id":    terraformValue(t, types.StringValue(organizationID)),
	})
}

func revokeSSHAccessActionConfig(t *testing.T, actionInstance action.Action, sandboxIDOrName, token, organizationID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id_or_name": tftypes.String,
		"token":              tftypes.String,
		"organization_id":    tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id_or_name": terraformValue(t, types.StringValue(sandboxIDOrName)),
		"token":              terraformValue(t, types.StringValue(token)),
		"organization_id":    terraformValue(t, types.StringValue(organizationID)),
	})
}

func newActionConfig(t *testing.T, actionInstance action.Action, objectType tftypes.Object, values map[string]tftypes.Value) *tfsdk.Config {
	t.Helper()

	var schemaResp action.SchemaResponse
	actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	return &tfsdk.Config{
		Raw:    tftypes.NewValue(objectType, values),
		Schema: schemaResp.Schema,
	}
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
