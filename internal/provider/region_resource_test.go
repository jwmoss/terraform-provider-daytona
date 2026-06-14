package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRegionResourceSchemaRotationFields(t *testing.T) {
	t.Parallel()

	regionResource := NewRegionResource()

	var metadataResp resource.MetadataResponse
	regionResource.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_region" {
		t.Fatalf("expected type name %q, got %q", "daytona_region", metadataResp.TypeName)
	}

	var schemaResp resource.SchemaResponse
	regionResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	for _, attributeName := range []string{
		"proxy_api_key_rotation_id",
		"ssh_gateway_api_key_rotation_id",
		"snapshot_manager_credentials_rotation_id",
	} {
		attribute, ok := schemaResp.Schema.Attributes[attributeName].(schema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute, got %T", attributeName, schemaResp.Schema.Attributes[attributeName])
		}
		if !attribute.Optional {
			t.Fatalf("expected %s to be optional", attributeName)
		}
	}

	for _, attributeName := range []string{
		"proxy_api_key",
		"ssh_gateway_api_key",
		"snapshot_manager_username",
		"snapshot_manager_password",
	} {
		attribute, ok := schemaResp.Schema.Attributes[attributeName].(schema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute, got %T", attributeName, schemaResp.Schema.Attributes[attributeName])
		}
		if !attribute.Computed || !attribute.Sensitive {
			t.Fatalf("expected %s to be computed and sensitive", attributeName)
		}
	}
}

func TestRegionResourceCredentialRotations(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method %s, got %s", http.MethodPost, r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(organizationHeader) != "org-1" {
			t.Fatalf("expected organization header %q, got %q", "org-1", r.Header.Get(organizationHeader))
		}

		path := r.URL.EscapedPath()
		requests[path]++
		w.Header().Set("Content-Type", "application/json")
		switch path {
		case "/regions/region-1/regenerate-proxy-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"new-proxy-key"}`))
		case "/regions/region-1/regenerate-ssh-gateway-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"new-ssh-key"}`))
		case "/regions/region-1/regenerate-snapshot-manager-credentials":
			_, _ = w.Write([]byte(`{"username":"new-snapshot-user","password":"new-snapshot-pass"}`))
		default:
			t.Fatalf("unexpected path %q", path)
		}
	}))
	defer server.Close()

	regionResource := &RegionResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	prior := regionResourceModel{
		ProxyAPIKey:               types.StringValue("old-proxy-key"),
		SSHGatewayAPIKey:          types.StringValue("old-ssh-key"),
		SnapshotManagerUsername:   types.StringValue("old-snapshot-user"),
		SnapshotManagerPassword:   types.StringValue("old-snapshot-pass"),
		ProxyAPIKeyRotationID:     types.StringValue("rotation-1"),
		SSHGatewayRotationID:      types.StringValue("rotation-1"),
		SnapshotManagerRotationID: types.StringValue("rotation-1"),
	}
	planned := prior
	planned.ID = types.StringValue("region-1")
	planned.ProxyAPIKeyRotationID = types.StringValue("rotation-2")
	planned.SSHGatewayRotationID = types.StringValue("rotation-2")
	planned.SnapshotManagerRotationID = types.StringValue("rotation-2")
	data := prior

	persistCalls := 0
	persist := func() bool {
		persistCalls++
		return true
	}

	if _, err := regionResource.applyRegionCredentialRotations(context.Background(), planned, prior, &data, persist); err != nil {
		t.Fatalf("unexpected rotation error: %s", err)
	}

	// Each rotation must persist as soon as its credential lands, so a later
	// rotation failure cannot lose an already-regenerated credential.
	if persistCalls != 3 {
		t.Fatalf("expected state to persist after each of 3 rotations, got %d", persistCalls)
	}
	if data.ProxyAPIKeyRotationID.ValueString() != "rotation-2" {
		t.Fatalf("expected proxy rotation ID to advance, got %q", data.ProxyAPIKeyRotationID.ValueString())
	}

	if data.ProxyAPIKey.ValueString() != "new-proxy-key" {
		t.Fatalf("expected rotated proxy key, got %q", data.ProxyAPIKey.ValueString())
	}
	if data.SSHGatewayAPIKey.ValueString() != "new-ssh-key" {
		t.Fatalf("expected rotated SSH gateway key, got %q", data.SSHGatewayAPIKey.ValueString())
	}
	if data.SnapshotManagerUsername.ValueString() != "new-snapshot-user" {
		t.Fatalf("expected rotated snapshot manager username, got %q", data.SnapshotManagerUsername.ValueString())
	}
	if data.SnapshotManagerPassword.ValueString() != "new-snapshot-pass" {
		t.Fatalf("expected rotated snapshot manager password, got %q", data.SnapshotManagerPassword.ValueString())
	}
	for _, path := range []string{
		"/regions/region-1/regenerate-proxy-api-key",
		"/regions/region-1/regenerate-ssh-gateway-api-key",
		"/regions/region-1/regenerate-snapshot-manager-credentials",
	} {
		if requests[path] != 1 {
			t.Fatalf("expected one request to %s, got %d", path, requests[path])
		}
	}
}
