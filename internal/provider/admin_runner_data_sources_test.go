package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAdminRunnerDataSourcesSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory  func() datasource.DataSource
		attrName string
	}{
		"daytona_admin_runner": {
			factory:  NewAdminRunnerDataSource,
			attrName: "id",
		},
		"daytona_admin_runners": {
			factory:  NewAdminRunnersDataSource,
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

func TestAdminRunnerDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(adminRunnerFullJSON("runner-1", false)))
	}))
	defer server.Close()

	dataSource := NewAdminRunnerDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"id": tftypes.String,
	}, map[string]tftypes.Value{
		"id": tftypes.NewValue(tftypes.String, "runner-1"),
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
	if gotPath != "/admin/runners/runner-1" {
		t.Fatalf("expected path %q, got %q", "/admin/runners/runner-1", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data runnerFullDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "runner-1" {
		t.Fatalf("expected runner ID %q, got %q", "runner-1", data.ID.ValueString())
	}
	if data.APIKey.ValueString() != "runner-secret" {
		t.Fatalf("expected runner API key to be flattened")
	}
}

func TestAdminRunnersDataSourceReadWithRegionFilter(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotRegionID, gotAuthorization, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotRegionID = r.URL.Query().Get("regionId")
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[" + adminRunnerFullJSON("runner-1", true) + "]"))
	}))
	defer server.Close()

	dataSource := NewAdminRunnersDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"region_id": tftypes.String,
	}, map[string]tftypes.Value{
		"region_id": tftypes.NewValue(tftypes.String, "region-1"),
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
	if gotPath != "/admin/runners" {
		t.Fatalf("expected path %q, got %q", "/admin/runners", gotPath)
	}
	if gotRegionID != "region-1" {
		t.Fatalf("expected regionId query %q, got %q", "region-1", gotRegionID)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}

	var data adminRunnersDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "admin_runners:region-1" {
		t.Fatalf("expected data source ID %q, got %q", "admin_runners:region-1", data.ID.ValueString())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected one runner, got %d", len(data.Items))
	}
	if data.Items[0].Unschedulable.ValueBool() != true {
		t.Fatalf("expected flattened unschedulable=true")
	}
}

func TestAdminRunnersDataSourceItemsSchemaMarksAPIKeySensitive(t *testing.T) {
	t.Parallel()

	dataSource := NewAdminRunnersDataSource()

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	itemsAttr, ok := schemaResp.Schema.Attributes["items"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected items to be a nested list attribute, got %T", schemaResp.Schema.Attributes["items"])
	}
	apiKeyAttr, ok := itemsAttr.NestedObject.Attributes["api_key"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected api_key to be a string attribute, got %T", itemsAttr.NestedObject.Attributes["api_key"])
	}
	if !apiKeyAttr.Sensitive {
		t.Fatal("expected nested api_key to be sensitive")
	}
}

func adminRunnerFullJSON(id string, unschedulable bool) string {
	return fmt.Sprintf(`{"id":%q,"domain":"runner.example.com","apiUrl":"https://api.runner.example.com","proxyUrl":"https://proxy.runner.example.com","cpu":8,"memory":16,"disk":100,"gpu":0,"gpuType":null,"sandboxClass":"container","currentCpuUsagePercentage":1,"currentMemoryUsagePercentage":2,"currentDiskUsagePercentage":3,"currentAllocatedCpu":4,"currentAllocatedMemoryGiB":5,"currentAllocatedDiskGiB":6,"currentSnapshotCount":7,"currentStartedSandboxes":8,"availabilityScore":99,"region":"us","name":"admin-runner","state":"ready","lastChecked":"2026-06-11T21:00:00Z","unschedulable":%t,"tags":["terraform"],"createdAt":"2026-06-11T20:00:00Z","updatedAt":"2026-06-11T21:00:00Z","version":"0","apiVersion":"2","runnerClass":"container","appVersion":"v1","apiKey":"runner-secret","regionType":"custom"}`, id, unschedulable)
}
