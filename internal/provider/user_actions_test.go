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

func TestUserLinkedAccountActionsSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]func() action.Action{
		"daytona_link_account":   NewUserLinkAccountAction,
		"daytona_unlink_account": NewUserUnlinkAccountAction,
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

			accountProviderAttr, ok := schemaResp.Schema.Attributes["account_provider"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected account_provider to be a string attribute, got %T", schemaResp.Schema.Attributes["account_provider"])
			}
			if !accountProviderAttr.Required {
				t.Fatal("expected account_provider to be required")
			}

			providerUserIDAttr, ok := schemaResp.Schema.Attributes["provider_user_id"].(actionschema.StringAttribute)
			if !ok {
				t.Fatalf("expected provider_user_id to be a string attribute, got %T", schemaResp.Schema.Attributes["provider_user_id"])
			}
			if !providerUserIDAttr.Required {
				t.Fatal("expected provider_user_id to be required")
			}
		})
	}
}

func TestUserSmsMFAEnrollmentActionSchema(t *testing.T) {
	t.Parallel()

	actionInstance := NewUserSmsMFAEnrollmentAction()

	var metadataResp action.MetadataResponse
	actionInstance.Metadata(context.Background(), action.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_enroll_sms_mfa" {
		t.Fatalf("expected type name %q, got %q", "daytona_enroll_sms_mfa", metadataResp.TypeName)
	}

	var schemaResp action.SchemaResponse
	actionInstance.Schema(context.Background(), action.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}
	if len(schemaResp.Schema.Attributes) != 0 {
		t.Fatalf("expected no SMS MFA enrollment attributes, got %#v", schemaResp.Schema.Attributes)
	}
}

func TestUserLinkAccountActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "terraform-provider-daytona/test" {
			t.Fatalf("expected provider user agent, got %q", r.Header.Get("User-Agent"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("unable to decode request body: %s", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	actionInstance := NewUserLinkAccountAction()
	configureActionClient(t, actionInstance, server.URL)

	config := userLinkedAccountActionConfig(t, actionInstance, "github", "provider-user-1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/users/linked-accounts" {
		t.Fatalf("expected path %q, got %q", "/users/linked-accounts", gotPath)
	}
	if gotBody["provider"] != "github" {
		t.Fatalf("expected provider %q, got %#v", "github", gotBody["provider"])
	}
	if gotBody["userId"] != "provider-user-1" {
		t.Fatalf("expected userId %q, got %#v", "provider-user-1", gotBody["userId"])
	}
}

func TestUserUnlinkAccountActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	actionInstance := NewUserUnlinkAccountAction()
	configureActionClient(t, actionInstance, server.URL)

	config := userLinkedAccountActionConfig(t, actionInstance, "github", "provider user/1")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodDelete {
		t.Fatalf("expected method %s, got %s", http.MethodDelete, gotMethod)
	}
	if gotPath != "/users/linked-accounts/github/provider%20user%2F1" {
		t.Fatalf("expected path %q, got %q", "/users/linked-accounts/github/provider%20user%2F1", gotPath)
	}
}

func TestUserSmsMFAEnrollmentActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"https://app.daytona.io/mfa/enroll/session-1"`))
	}))
	defer server.Close()

	actionInstance := NewUserSmsMFAEnrollmentAction()
	configureActionClient(t, actionInstance, server.URL)

	config := emptyUserActionConfig(t, actionInstance)
	var progressMessages []string
	var invokeResp action.InvokeResponse
	invokeResp.SendProgress = func(event action.InvokeProgressEvent) {
		progressMessages = append(progressMessages, event.Message)
	}

	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/users/mfa/sms/enroll" {
		t.Fatalf("expected path %q, got %q", "/users/mfa/sms/enroll", gotPath)
	}
	if len(progressMessages) != 2 {
		t.Fatalf("expected two progress messages, got %#v", progressMessages)
	}
	if progressMessages[1] != "Daytona SMS MFA enrollment URL: https://app.daytona.io/mfa/enroll/session-1" {
		t.Fatalf("expected enrollment URL progress message, got %#v", progressMessages)
	}
}

func TestUserLinkedAccountActionRejectsMissingProviderUserID(t *testing.T) {
	t.Parallel()

	actionInstance := NewUserLinkAccountAction()
	configureActionClient(t, actionInstance, "https://daytona.invalid")

	config := userLinkedAccountActionConfig(t, actionInstance, "github", "")

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if !invokeResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for missing provider_user_id")
	}
}

func userLinkedAccountActionConfig(t *testing.T, actionInstance action.Action, accountProvider, providerUserID string) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"account_provider": tftypes.String,
		"provider_user_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"account_provider": terraformValue(t, types.StringValue(accountProvider)),
		"provider_user_id": terraformValue(t, types.StringValue(providerUserID)),
	})
}

func emptyUserActionConfig(t *testing.T, actionInstance action.Action) *tfsdk.Config {
	t.Helper()

	return newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
}
