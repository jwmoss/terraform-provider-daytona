package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAdminUserActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory  func() action.Action
		attrName string
	}{
		"daytona_admin_create_user": {
			factory:  NewAdminCreateUserAction,
			attrName: "name",
		},
		"daytona_admin_regenerate_user_key_pair": {
			factory:  NewAdminRegenerateUserKeyPairAction,
			attrName: "user_id",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actionInstance := testCase.factory()

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

			if _, ok := schemaResp.Schema.Attributes[testCase.attrName]; !ok {
				t.Fatalf("expected %s attribute in schema", testCase.attrName)
			}
		})
	}
}

func TestAdminCreateUserActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string
	var gotBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("unable to decode request body: %s", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	actionInstance := NewAdminCreateUserAction()
	configureActionClient(t, actionInstance, server.URL)

	config := adminCreateUserActionConfig(t, actionInstance, map[string]tftypes.Value{
		"user_id":                          terraformValue(t, types.StringValue("user-1")),
		"name":                             terraformValue(t, types.StringValue("Automation User")),
		"email":                            terraformValue(t, types.StringValue("automation@example.com")),
		"personal_organization_quota_json": terraformValue(t, types.StringValue(`{"totalCpuQuota":4,"snapshotQuota":10}`)),
		"personal_organization_default_region_id": terraformValue(t, types.StringValue("region-1")),
		"role":           terraformValue(t, types.StringValue("admin")),
		"email_verified": terraformValue(t, types.BoolValue(true)),
	})

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/users" {
		t.Fatalf("expected path %q, got %q", "/admin/users", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
	if gotBody["id"] != "user-1" || gotBody["name"] != "Automation User" || gotBody["role"] != "admin" {
		t.Fatalf("unexpected request body %#v", gotBody)
	}
	if gotBody["emailVerified"] != true {
		t.Fatalf("expected emailVerified=true, got %#v", gotBody["emailVerified"])
	}
	quota, ok := gotBody["personalOrganizationQuota"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected personalOrganizationQuota object, got %#v", gotBody["personalOrganizationQuota"])
	}
	if quota["totalCpuQuota"] != float64(4) {
		t.Fatalf("expected quota totalCpuQuota=4, got %#v", quota["totalCpuQuota"])
	}
}

func TestAdminRegenerateUserKeyPairActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	actionInstance := NewAdminRegenerateUserKeyPairAction()
	configureActionClient(t, actionInstance, server.URL)

	config := adminRegenerateUserKeyPairActionConfig(t, actionInstance, "user/1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/admin/users/user%2F1/regenerate-key-pair" {
		t.Fatalf("expected path %q, got %q", "/admin/users/user%2F1/regenerate-key-pair", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
}

func TestAdminCreateUserActionRejectsInvalidRole(t *testing.T) {
	t.Parallel()

	actionInstance := NewAdminCreateUserAction()
	configureActionClient(t, actionInstance, "https://daytona.invalid")

	config := adminCreateUserActionConfig(t, actionInstance, map[string]tftypes.Value{
		"user_id":                          terraformValue(t, types.StringValue("user-1")),
		"name":                             terraformValue(t, types.StringValue("Automation User")),
		"email":                            terraformValue(t, types.StringNull()),
		"personal_organization_quota_json": terraformValue(t, types.StringNull()),
		"personal_organization_default_region_id": terraformValue(t, types.StringNull()),
		"role":           terraformValue(t, types.StringValue("owner")),
		"email_verified": terraformValue(t, types.BoolNull()),
	})

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if !invokeResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for invalid role")
	}
}

func adminCreateUserActionConfig(t *testing.T, actionInstance action.Action, values map[string]tftypes.Value) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"user_id":                          tftypes.String,
		"name":                             tftypes.String,
		"email":                            tftypes.String,
		"personal_organization_quota_json": tftypes.String,
		"personal_organization_default_region_id": tftypes.String,
		"role":           tftypes.String,
		"email_verified": tftypes.Bool,
	}}, values)
}

func adminRegenerateUserKeyPairActionConfig(t *testing.T, actionInstance action.Action, userID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"user_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"user_id": terraformValue(t, types.StringValue(userID)),
	})
}

func TestAdminCreateUserOptionalQuotaSchema(t *testing.T) {
	t.Parallel()

	actionInstance := NewAdminCreateUserAction()

	var schemaResp action.SchemaResponse
	actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	quotaAttr, ok := schemaResp.Schema.Attributes["personal_organization_quota_json"].(actionschema.StringAttribute)
	if !ok {
		t.Fatalf("expected personal_organization_quota_json to be a string attribute, got %T", schemaResp.Schema.Attributes["personal_organization_quota_json"])
	}
	if !quotaAttr.Optional {
		t.Fatal("expected personal_organization_quota_json to be optional")
	}
}
