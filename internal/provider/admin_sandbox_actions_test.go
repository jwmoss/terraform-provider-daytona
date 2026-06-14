package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAdminRecoverSandboxActionSchema(t *testing.T) {
	t.Parallel()

	actionInstance := NewAdminRecoverSandboxAction()

	var metadataResp action.MetadataResponse
	actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_admin_recover_sandbox" {
		t.Fatalf("expected type name %q, got %q", "daytona_admin_recover_sandbox", metadataResp.TypeName)
	}

	var schemaResp action.SchemaResponse
	actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	sandboxIDAttr, ok := schemaResp.Schema.Attributes["sandbox_id"].(actionschema.StringAttribute)
	if !ok {
		t.Fatalf("expected sandbox_id to be a string attribute, got %T", schemaResp.Schema.Attributes["sandbox_id"])
	}
	if !sandboxIDAttr.Required {
		t.Fatal("expected sandbox_id to be required")
	}
}

func TestAdminRecoverSandboxActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		writeSandboxActionResponse(t, w)
	}))
	defer server.Close()

	actionInstance := NewAdminRecoverSandboxAction()
	configureActionClient(t, actionInstance, server.URL)

	config := adminRecoverSandboxActionConfig(t, actionInstance, "sandbox/1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/sandbox/sandbox%2F1/recover" {
		t.Fatalf("expected path %q, got %q", "/admin/sandbox/sandbox%2F1/recover", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
}

func TestAdminRecoverSandboxActionRejectsMissingID(t *testing.T) {
	t.Parallel()

	actionInstance := NewAdminRecoverSandboxAction()
	configureActionClient(t, actionInstance, "https://daytona.invalid")

	config := adminRecoverSandboxActionConfig(t, actionInstance, " ")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if !invokeResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for missing sandbox ID")
	}
}

func adminRecoverSandboxActionConfig(t *testing.T, actionInstance action.Action, sandboxID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id": terraformValue(t, types.StringValue(sandboxID)),
	})
}
