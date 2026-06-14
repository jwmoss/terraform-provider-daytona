package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAdminWebhookDataSourcesSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory  func() datasource.DataSource
		attrName string
	}{
		"daytona_admin_webhook_status": {
			factory:  NewAdminWebhookStatusDataSource,
			attrName: "enabled",
		},
		"daytona_admin_webhook_message_attempts": {
			factory:  NewAdminWebhookMessageAttemptsDataSource,
			attrName: "attempts_json",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dataSource := testCase.factory()

			var metadataResp datasource.MetadataResponse
			dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != name {
				t.Fatalf("expected type name %q, got %q", name, metadataResp.TypeName)
			}

			var schemaResp datasource.SchemaResponse
			dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			if _, ok := schemaResp.Schema.Attributes[testCase.attrName]; !ok {
				t.Fatalf("expected %s attribute in schema", testCase.attrName)
			}
		})
	}
}

func TestAdminWebhookStatusDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true}`))
	}))
	defer server.Close()

	dataSource := NewAdminWebhookStatusDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{}, map[string]tftypes.Value{})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/admin/webhooks/status" {
		t.Fatalf("expected path %q, got %q", "/admin/webhooks/status", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminWebhookStatusDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if !data.Enabled.ValueBool() {
		t.Fatal("expected enabled=true")
	}
}

func TestAdminWebhookMessageAttemptsDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"attempt-1","status":200}]`))
	}))
	defer server.Close()

	dataSource := NewAdminWebhookMessageAttemptsDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"organization_id": tftypes.String,
		"message_id":      tftypes.String,
	}, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"message_id":      tftypes.NewValue(tftypes.String, "msg-1"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/admin/webhooks/organizations/org-1/messages/msg-1/attempts" {
		t.Fatalf("expected path %q, got %q", "/admin/webhooks/organizations/org-1/messages/msg-1/attempts", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminWebhookMessageAttemptsDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "org-1:msg-1:admin_webhook_message_attempts" {
		t.Fatalf("expected state ID %q, got %q", "org-1:msg-1:admin_webhook_message_attempts", data.ID.ValueString())
	}
	if data.AttemptsJSON.ValueString() != `[{"id":"attempt-1","status":200}]` {
		t.Fatalf("unexpected attempts JSON %q", data.AttemptsJSON.ValueString())
	}
}

func TestAdminWebhookMessageAttemptsSensitiveSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewAdminWebhookMessageAttemptsDataSource()

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	attemptsAttr, ok := schemaResp.Schema.Attributes["attempts_json"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected attempts_json to be a string attribute, got %T", schemaResp.Schema.Attributes["attempts_json"])
	}
	if !attemptsAttr.Sensitive {
		t.Fatal("expected attempts_json to be sensitive")
	}
}
