// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

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

func TestAdminSnapshotImageCleanupStatusDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewAdminSnapshotImageCleanupStatusDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_admin_snapshot_image_cleanup_status" {
		t.Fatalf("expected type name %q, got %q", "daytona_admin_snapshot_image_cleanup_status", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	imageNameAttr, ok := schemaResp.Schema.Attributes["image_name"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected image_name to be a string attribute, got %T", schemaResp.Schema.Attributes["image_name"])
	}
	if !imageNameAttr.Required {
		t.Fatal("expected image_name to be required")
	}

	canCleanupAttr, ok := schemaResp.Schema.Attributes["can_cleanup"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected can_cleanup to be a bool attribute, got %T", schemaResp.Schema.Attributes["can_cleanup"])
	}
	if !canCleanupAttr.Computed {
		t.Fatal("expected can_cleanup to be computed")
	}
}

func TestAdminSnapshotImageCleanupStatusDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotImageName, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotImageName = r.URL.Query().Get("imageName")
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`true`))
	}))
	defer server.Close()

	dataSource := NewAdminSnapshotImageCleanupStatusDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"image_name": tftypes.String,
	}, map[string]tftypes.Value{
		"image_name": tftypes.NewValue(tftypes.String, "registry.example.com/base:latest"),
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
	if gotPath != "/admin/snapshots/can-cleanup-image" {
		t.Fatalf("expected path %q, got %q", "/admin/snapshots/can-cleanup-image", gotPath)
	}
	if gotImageName != "registry.example.com/base:latest" {
		t.Fatalf("expected imageName query %q, got %q", "registry.example.com/base:latest", gotImageName)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminSnapshotImageCleanupStatusDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if !data.CanCleanup.ValueBool() {
		t.Fatal("expected image to be cleanable")
	}
}
