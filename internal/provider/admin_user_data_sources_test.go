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

func TestAdminUserDataSourcesSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory  func() datasource.DataSource
		attrName string
	}{
		"daytona_admin_user": {
			factory:  NewAdminUserDataSource,
			attrName: "user_id",
		},
		"daytona_admin_users": {
			factory:  NewAdminUsersDataSource,
			attrName: "items",
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

func TestAdminUserDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-1","name":"Automation User","email":"automation@example.com","publicKeys":[{"name":"default","key":"ssh-ed25519 AAA"}],"createdAt":"2026-06-11T21:00:00Z"}`))
	}))
	defer server.Close()

	dataSource := NewAdminUserDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"user_id": tftypes.String,
	}, map[string]tftypes.Value{
		"user_id": tftypes.NewValue(tftypes.String, "user-1"),
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
	if gotPath != "/admin/users/user-1" {
		t.Fatalf("expected path %q, got %q", "/admin/users/user-1", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminUserDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "user-1" {
		t.Fatalf("expected user ID %q, got %q", "user-1", data.ID.ValueString())
	}
	if data.Name.ValueString() != "Automation User" {
		t.Fatalf("expected user name %q, got %q", "Automation User", data.Name.ValueString())
	}
	if data.CreatedAt.ValueString() != "2026-06-11T21:00:00Z" {
		t.Fatalf("expected created_at %q, got %q", "2026-06-11T21:00:00Z", data.CreatedAt.ValueString())
	}
}

func TestAdminUsersDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"user-1","name":"Automation User","email":"automation@example.com","publicKeys":[],"createdAt":"2026-06-11T21:00:00Z"}]`))
	}))
	defer server.Close()

	dataSource := NewAdminUsersDataSource()
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
	if gotPath != "/admin/users" {
		t.Fatalf("expected path %q, got %q", "/admin/users", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminUsersDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "admin_users" {
		t.Fatalf("expected data source ID %q, got %q", "admin_users", data.ID.ValueString())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected one user, got %d", len(data.Items))
	}
	if data.Items[0].ID.ValueString() != "user-1" {
		t.Fatalf("expected user ID %q, got %q", "user-1", data.Items[0].ID.ValueString())
	}
}

func TestAdminUsersDataSourceItemsSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewAdminUsersDataSource()

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	itemsAttr, ok := schemaResp.Schema.Attributes["items"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected items to be a nested list attribute, got %T", schemaResp.Schema.Attributes["items"])
	}
	if _, ok := itemsAttr.NestedObject.Attributes["public_keys"].(schema.ListNestedAttribute); !ok {
		t.Fatalf("expected public_keys to be a nested list attribute, got %T", itemsAttr.NestedObject.Attributes["public_keys"])
	}
}
