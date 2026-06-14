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

func TestAdminAuditLogsDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewAdminAuditLogsDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_admin_audit_logs" {
		t.Fatalf("expected type name %q, got %q", "daytona_admin_audit_logs", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	limitAttr, ok := schemaResp.Schema.Attributes["limit"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected limit to be an int64 attribute, got %T", schemaResp.Schema.Attributes["limit"])
	}
	if !limitAttr.Optional || !limitAttr.Computed {
		t.Fatal("expected limit to be optional and computed")
	}

	itemsAttr, ok := schemaResp.Schema.Attributes["items"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected items to be a nested list attribute, got %T", schemaResp.Schema.Attributes["items"])
	}
	if !itemsAttr.Computed {
		t.Fatal("expected items to be computed")
	}
}

func TestAdminAuditLogsDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotPage, gotLimit, gotFrom, gotTo, gotNextToken, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotPage = r.URL.Query().Get("page")
		gotLimit = r.URL.Query().Get("limit")
		gotFrom = r.URL.Query().Get("from")
		gotTo = r.URL.Query().Get("to")
		gotNextToken = r.URL.Query().Get("nextToken")
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"audit-1","actorId":"user-1","actorEmail":"admin@example.com","actorApiKeyPrefix":"dtn_abc","actorApiKeySuffix":"xyz","organizationId":"org-1","action":"CREATE","targetType":"USER","targetId":"user-2","statusCode":200,"errorMessage":null,"ipAddress":"127.0.0.1","userAgent":"terraform-provider-daytona/test","source":"api","metadata":{"key":"value"},"createdAt":"2026-06-11T00:00:00Z"}],"total":1,"page":2,"totalPages":1,"nextToken":"next-page"}`))
	}))
	defer server.Close()

	dataSource := NewAdminAuditLogsDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"page":   tftypes.Number,
		"limit":  tftypes.Number,
		"from":   tftypes.String,
		"to":     tftypes.String,
		"cursor": tftypes.String,
	}, map[string]tftypes.Value{
		"page":   tftypes.NewValue(tftypes.Number, 2),
		"limit":  tftypes.NewValue(tftypes.Number, 50),
		"from":   tftypes.NewValue(tftypes.String, "2026-06-11T00:00:00Z"),
		"to":     tftypes.NewValue(tftypes.String, "2026-06-12T00:00:00Z"),
		"cursor": tftypes.NewValue(tftypes.String, "cursor-1"),
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
	if gotPath != "/admin/audit" {
		t.Fatalf("expected path %q, got %q", "/admin/audit", gotPath)
	}
	if gotPage != "2" {
		t.Fatalf("expected page query %q, got %q", "2", gotPage)
	}
	if gotLimit != "50" {
		t.Fatalf("expected limit query %q, got %q", "50", gotLimit)
	}
	if gotFrom != "2026-06-11T00:00:00Z" {
		t.Fatalf("expected from query %q, got %q", "2026-06-11T00:00:00Z", gotFrom)
	}
	if gotTo != "2026-06-12T00:00:00Z" {
		t.Fatalf("expected to query %q, got %q", "2026-06-12T00:00:00Z", gotTo)
	}
	if gotNextToken != "cursor-1" {
		t.Fatalf("expected nextToken query %q, got %q", "cursor-1", gotNextToken)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminAuditLogsDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "admin_audit_logs" {
		t.Fatalf("expected state ID %q, got %q", "admin_audit_logs", data.ID.ValueString())
	}
	if data.Total.ValueInt64() != 1 {
		t.Fatalf("expected total 1, got %d", data.Total.ValueInt64())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected one audit log item, got %d", len(data.Items))
	}
	if data.Items[0].OrganizationID.ValueString() != "org-1" {
		t.Fatalf("expected organization_id %q, got %q", "org-1", data.Items[0].OrganizationID.ValueString())
	}
}
