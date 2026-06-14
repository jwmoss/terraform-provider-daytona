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

func TestOrganizationOtelConfigBySandboxAuthTokenDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewOrganizationOtelConfigBySandboxAuthTokenDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_organization_otel_config_by_sandbox_auth_token" {
		t.Fatalf("expected type name %q, got %q", "daytona_organization_otel_config_by_sandbox_auth_token", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	authTokenAttr, ok := schemaResp.Schema.Attributes["auth_token"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected auth_token to be a string attribute, got %T", schemaResp.Schema.Attributes["auth_token"])
	}
	if !authTokenAttr.Required {
		t.Fatal("expected auth_token to be required")
	}
	if !authTokenAttr.Sensitive {
		t.Fatal("expected auth_token to be sensitive")
	}

	headersAttr, ok := schemaResp.Schema.Attributes["headers"].(schema.MapAttribute)
	if !ok {
		t.Fatalf("expected headers to be a map attribute, got %T", schemaResp.Schema.Attributes["headers"])
	}
	if !headersAttr.Computed {
		t.Fatal("expected headers to be computed")
	}
	if !headersAttr.Sensitive {
		t.Fatal("expected headers to be sensitive")
	}
}

func TestOrganizationOtelConfigBySandboxAuthTokenDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"endpoint":"https://otel.example.com/v1/traces","headers":{"Authorization":"Bearer secret"}}`))
	}))
	defer server.Close()

	dataSource := NewOrganizationOtelConfigBySandboxAuthTokenDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"auth_token": tftypes.String,
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, "sandbox-auth-token"),
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
	if gotPath != "/organizations/otel-config/by-sandbox-auth-token/sandbox-auth-token" {
		t.Fatalf("expected path %q, got %q", "/organizations/otel-config/by-sandbox-auth-token/sandbox-auth-token", gotPath)
	}

	var data organizationOtelConfigBySandboxAuthTokenDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.Endpoint.ValueString() != "https://otel.example.com/v1/traces" {
		t.Fatalf("expected endpoint %q, got %q", "https://otel.example.com/v1/traces", data.Endpoint.ValueString())
	}

	headers := map[string]string{}
	diags := data.Headers.ElementsAs(context.Background(), &headers, false)
	if diags.HasError() {
		t.Fatalf("unexpected headers diagnostics: %s", diags)
	}
	if headers["Authorization"] != "Bearer secret" {
		t.Fatalf("expected Authorization header %q, got %q", "Bearer secret", headers["Authorization"])
	}
}

func TestOrganizationOtelConfigBySandboxAuthTokenDataSourceRejectsEmptyToken(t *testing.T) {
	t.Parallel()

	dataSource := NewOrganizationOtelConfigBySandboxAuthTokenDataSource()
	configureDataSource(t, dataSource, "https://daytona.invalid")

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"auth_token": tftypes.String,
	}, map[string]tftypes.Value{
		"auth_token": tftypes.NewValue(tftypes.String, " "),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for empty auth token")
	}
}
