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

func TestAPIKeyForUserRevokeActionSchema(t *testing.T) {
	t.Parallel()

	actionInstance := NewAPIKeyForUserRevokeAction()

	var metadataResp action.MetadataResponse
	actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_revoke_api_key_for_user" {
		t.Fatalf("expected type name %q, got %q", "daytona_revoke_api_key_for_user", metadataResp.TypeName)
	}

	var schemaResp action.SchemaResponse
	actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	for _, attribute := range []string{"user_id", "name"} {
		attr, ok := schemaResp.Schema.Attributes[attribute].(actionschema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute, got %T", attribute, schemaResp.Schema.Attributes[attribute])
		}
		if !attr.Required {
			t.Fatalf("expected %s to be required", attribute)
		}
	}

	organizationIDAttr, ok := schemaResp.Schema.Attributes["organization_id"].(actionschema.StringAttribute)
	if !ok {
		t.Fatalf("expected organization_id to be a string attribute, got %T", schemaResp.Schema.Attributes["organization_id"])
	}
	if !organizationIDAttr.Optional {
		t.Fatal("expected organization_id to be optional")
	}
}

func TestAPIKeyForUserRevokeActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "terraform-provider-daytona/test" {
			t.Fatalf("expected provider user agent, got %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	actionInstance := NewAPIKeyForUserRevokeAction()
	configureActionClient(t, actionInstance, server.URL)

	config := apiKeyForUserRevokeActionConfig(t, actionInstance, "user 1", "key/name", "org-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("expected method %s, got %s", http.MethodDelete, gotMethod)
	}
	if gotPath != "/api-keys/user%201/key%2Fname" {
		t.Fatalf("expected path %q, got %q", "/api-keys/user%201/key%2Fname", gotPath)
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
}

func TestAPIKeyForUserRevokeActionRejectsMissingName(t *testing.T) {
	t.Parallel()

	actionInstance := NewAPIKeyForUserRevokeAction()
	configureActionClient(t, actionInstance, "https://daytona.invalid")

	config := apiKeyForUserRevokeActionConfig(t, actionInstance, "user-1", "", "")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if !invokeResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for missing name")
	}
}

func apiKeyForUserRevokeActionConfig(t *testing.T, actionInstance action.Action, userID, name, organizationID string) *tfsdk.Config {
	t.Helper()

	var organizationValue tftypes.Value
	if organizationID == "" {
		organizationValue = terraformValue(t, types.StringNull())
	} else {
		organizationValue = terraformValue(t, types.StringValue(organizationID))
	}

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"user_id":         tftypes.String,
		"name":            tftypes.String,
		"organization_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"user_id":         terraformValue(t, types.StringValue(userID)),
		"name":            terraformValue(t, types.StringValue(name)),
		"organization_id": organizationValue,
	})
}
