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

func TestCurrentUserOrganizationInvitationsDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewCurrentUserOrganizationInvitationsDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_current_user_organization_invitations" {
		t.Fatalf("expected type name %q, got %q", "daytona_current_user_organization_invitations", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	countAttr, ok := schemaResp.Schema.Attributes["total_count"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected total_count to be an int64 attribute, got %T", schemaResp.Schema.Attributes["total_count"])
	}
	if !countAttr.Computed {
		t.Fatal("expected count to be computed")
	}

	itemsAttr, ok := schemaResp.Schema.Attributes["items"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected items to be a list nested attribute, got %T", schemaResp.Schema.Attributes["items"])
	}
	if !itemsAttr.Computed {
		t.Fatal("expected items to be computed")
	}
}

func TestCurrentUserOrganizationInvitationsDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
		}

		gotPaths = append(gotPaths, r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.EscapedPath() {
		case "/organizations/invitations/count":
			_, _ = w.Write([]byte(`1`))
		case "/organizations/invitations":
			_, _ = w.Write([]byte(`[{"id":"inv-1","email":"user@example.com","invitedBy":"owner@example.com","organizationId":"org-1","organizationName":"Automation","expiresAt":"2026-06-20T00:00:00Z","status":"pending","role":"member","assignedRoles":[{"id":"role-1","name":"Member","description":"Member role","permissions":["organization:read"],"isGlobal":false,"createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z"}],"createdAt":"2026-06-11T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z"}]`))
		default:
			t.Fatalf("unexpected path %q", r.URL.EscapedPath())
		}
	}))
	defer server.Close()

	dataSource := NewCurrentUserOrganizationInvitationsDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := emptyDataSourceConfig(t, dataSource)

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 requests, got %#v", gotPaths)
	}
	if gotPaths[0] != "/organizations/invitations/count" || gotPaths[1] != "/organizations/invitations" {
		t.Fatalf("unexpected request paths: %#v", gotPaths)
	}

	var data currentUserOrganizationInvitationsDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.TotalCount.ValueInt64() != 1 {
		t.Fatalf("expected total_count 1, got %d", data.TotalCount.ValueInt64())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(data.Items))
	}
	if data.Items[0].ID.ValueString() != "inv-1" {
		t.Fatalf("expected invitation ID %q, got %q", "inv-1", data.Items[0].ID.ValueString())
	}
	if data.Items[0].OrganizationName.ValueString() != "Automation" {
		t.Fatalf("expected organization name %q, got %q", "Automation", data.Items[0].OrganizationName.ValueString())
	}
}

func configureDataSource(t *testing.T, dataSource datasource.DataSource, apiURL string) {
	t.Helper()

	configurable, ok := dataSource.(datasource.DataSourceWithConfigure)
	if !ok {
		t.Fatal("expected data source to implement DataSourceWithConfigure")
	}

	var configureResp datasource.ConfigureResponse
	configurable.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: newDaytonaClient(apiURL, "test-key", "", "test")}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %s", configureResp.Diagnostics)
	}
}

func emptyDataSourceConfig(t *testing.T, dataSource datasource.DataSource) *tfsdk.Config {
	t.Helper()

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	return &tfsdk.Config{
		Raw:    tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{}),
		Schema: schemaResp.Schema,
	}
}
