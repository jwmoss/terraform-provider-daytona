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

func TestWebhookActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]func() action.Action{
		"daytona_initialize_webhooks":       NewWebhookInitializeAction,
		"daytona_refresh_webhook_endpoints": NewWebhookRefreshEndpointsAction,
	}

	for name, factory := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actionInstance := factory()

			var metadataResp action.MetadataResponse
			actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != name {
				t.Fatalf("expected type name %q, got %q", name, metadataResp.TypeName)
			}

			var schemaResp action.SchemaResponse
			actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
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

func TestWebhookInitializeActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"organizationId":"org-1","svixApplicationId":"app-1","lastError":null,"retryCount":1,"createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z"}`))
	}))
	defer server.Close()

	actionInstance := NewWebhookInitializeAction()
	configureActionClient(t, actionInstance, server.URL)

	config := webhookActionConfig(t, actionInstance, "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}

	if gotPath != "/webhooks/organizations/org-1/initialize" {
		t.Fatalf("expected path %q, got %q", "/webhooks/organizations/org-1/initialize", gotPath)
	}

	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestWebhookRefreshEndpointsActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	actionInstance := NewWebhookRefreshEndpointsAction()
	configureActionClientWithOrganization(t, actionInstance, server.URL, "org-1")

	config := webhookActionConfig(t, actionInstance, "")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}

	if gotPath != "/webhooks/organizations/org-1/refresh-endpoints" {
		t.Fatalf("expected path %q, got %q", "/webhooks/organizations/org-1/refresh-endpoints", gotPath)
	}

	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func configureActionClientWithOrganization(t *testing.T, actionInstance action.Action, apiURL, organizationID string) {
	t.Helper()

	configurable, ok := actionInstance.(action.ActionWithConfigure)
	if !ok {
		t.Fatal("expected action to implement ActionWithConfigure")
	}

	var configureResp action.ConfigureResponse
	configurable.Configure(context.Background(), action.ConfigureRequest{ProviderData: newDaytonaClient(apiURL, "test-key", organizationID, "test")}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %s", configureResp.Diagnostics)
	}
}

func webhookActionConfig(t *testing.T, actionInstance action.Action, organizationID string) *tfsdk.Config {
	t.Helper()

	var organizationValue tftypes.Value
	if organizationID == "" {
		organizationValue = terraformValue(t, types.StringNull())
	} else {
		organizationValue = terraformValue(t, types.StringValue(organizationID))
	}

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"organization_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"organization_id": organizationValue,
	})
}
