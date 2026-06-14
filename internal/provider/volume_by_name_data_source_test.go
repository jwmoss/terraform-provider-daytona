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

func TestVolumeByNameDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewVolumeByNameDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_volume_by_name" {
		t.Fatalf("expected type name %q, got %q", "daytona_volume_by_name", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	nameAttr, ok := schemaResp.Schema.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected name to be a string attribute, got %T", schemaResp.Schema.Attributes["name"])
	}
	if !nameAttr.Required {
		t.Fatal("expected name to be required")
	}

	idAttr, ok := schemaResp.Schema.Attributes["id"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected id to be a string attribute, got %T", schemaResp.Schema.Attributes["id"])
	}
	if !idAttr.Computed {
		t.Fatal("expected id to be computed")
	}
}

func TestVolumeByNameDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPath, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get("X-Daytona-Organization-ID")
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
		}
		if gotPath != "/volumes/by-name/workspace-cache" {
			t.Fatalf("unexpected path %q", gotPath)
		}

		_, _ = w.Write([]byte(`{"id":"vol-1","name":"workspace-cache","organizationId":"org-1","state":"ready","createdAt":"2026-06-10T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z","lastUsedAt":null,"errorReason":null}`))
	}))
	defer server.Close()

	dataSource := NewVolumeByNameDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"name":                    tftypes.String,
		"request_organization_id": tftypes.String,
	}, map[string]tftypes.Value{
		"name":                    tftypes.NewValue(tftypes.String, "workspace-cache"),
		"request_organization_id": tftypes.NewValue(tftypes.String, "org-1"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}

	var data volumeByNameDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "vol-1" {
		t.Fatalf("expected volume ID %q, got %q", "vol-1", data.ID.ValueString())
	}
	if data.OrganizationID.ValueString() != "org-1" {
		t.Fatalf("expected organization ID %q, got %q", "org-1", data.OrganizationID.ValueString())
	}
	if data.State.ValueString() != "ready" {
		t.Fatalf("expected state %q, got %q", "ready", data.State.ValueString())
	}
}
