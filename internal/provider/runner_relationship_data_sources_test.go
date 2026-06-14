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

func TestRunnerRelationshipDataSourcesSchema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		factory           func() datasource.DataSource
		expectedTypeName  string
		requiredAttribute string
	}{
		"runner full": {
			factory:           NewRunnerFullDataSource,
			expectedTypeName:  "daytona_runner_full",
			requiredAttribute: "id",
		},
		"runner for sandbox": {
			factory:           NewRunnerForSandboxDataSource,
			expectedTypeName:  "daytona_runner_for_sandbox",
			requiredAttribute: "sandbox_id",
		},
		"runners by snapshot ref": {
			factory:           NewRunnersBySnapshotRefDataSource,
			expectedTypeName:  "daytona_runners_by_snapshot_ref",
			requiredAttribute: "ref",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dataSource := testCase.factory()

			var metadataResp datasource.MetadataResponse
			dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
			if metadataResp.TypeName != testCase.expectedTypeName {
				t.Fatalf("expected type name %q, got %q", testCase.expectedTypeName, metadataResp.TypeName)
			}

			var schemaResp datasource.SchemaResponse
			dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
			}

			requiredAttr, ok := schemaResp.Schema.Attributes[testCase.requiredAttribute].(schema.StringAttribute)
			if !ok {
				t.Fatalf("expected %s to be a string attribute, got %T", testCase.requiredAttribute, schemaResp.Schema.Attributes[testCase.requiredAttribute])
			}
			if !requiredAttr.Required {
				t.Fatalf("expected %s to be required", testCase.requiredAttribute)
			}

			if testCase.expectedTypeName == "daytona_runners_by_snapshot_ref" {
				itemsAttr, ok := schemaResp.Schema.Attributes["items"].(schema.ListNestedAttribute)
				if !ok {
					t.Fatalf("expected items to be a list nested attribute, got %T", schemaResp.Schema.Attributes["items"])
				}
				if !itemsAttr.Computed {
					t.Fatal("expected items to be computed")
				}
				return
			}

			apiKeyAttr, ok := schemaResp.Schema.Attributes["api_key"].(schema.StringAttribute)
			if !ok {
				t.Fatalf("expected api_key to be a string attribute, got %T", schemaResp.Schema.Attributes["api_key"])
			}
			if !apiKeyAttr.Computed || !apiKeyAttr.Sensitive {
				t.Fatal("expected api_key to be computed and sensitive")
			}
		})
	}
}

func TestRunnerFullDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")

		if gotPath != "/runners/runner-1/full" {
			t.Fatalf("unexpected path %q", gotPath)
		}
		_, _ = w.Write([]byte(runnerFullJSON("runner-1")))
	}))
	defer server.Close()

	dataSource := NewRunnerFullDataSource()
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

	var data runnerFullDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "runner-1" {
		t.Fatalf("expected runner ID %q, got %q", "runner-1", data.ID.ValueString())
	}
	if data.APIKey.ValueString() != "runner-api-key" {
		t.Fatalf("expected runner API key %q, got %q", "runner-api-key", data.APIKey.ValueString())
	}
	if data.CurrentCPUUsagePercentage.ValueFloat64() != 25.5 {
		t.Fatalf("expected CPU usage 25.5, got %g", data.CurrentCPUUsagePercentage.ValueFloat64())
	}
	if data.RegionType.ValueString() != "custom" {
		t.Fatalf("expected region type %q, got %q", "custom", data.RegionType.ValueString())
	}
}

func TestRunnerForSandboxDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")

		if gotPath != "/runners/by-sandbox/sandbox-1" {
			t.Fatalf("unexpected path %q", gotPath)
		}
		_, _ = w.Write([]byte(runnerFullJSON("runner-for-sandbox")))
	}))
	defer server.Close()

	dataSource := NewRunnerForSandboxDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"sandbox_id": tftypes.String,
	}, map[string]tftypes.Value{
		"sandbox_id": tftypes.NewValue(tftypes.String, "sandbox-1"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	var data runnerFullDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.SandboxID.ValueString() != "sandbox-1" {
		t.Fatalf("expected sandbox ID %q, got %q", "sandbox-1", data.SandboxID.ValueString())
	}
	if data.ID.ValueString() != "runner-for-sandbox" {
		t.Fatalf("expected runner ID %q, got %q", "runner-for-sandbox", data.ID.ValueString())
	}
}

func TestRunnersBySnapshotRefDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPath, gotRef string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotRef = r.URL.Query().Get("ref")
		w.Header().Set("Content-Type", "application/json")

		if gotPath != "/runners/by-snapshot-ref" {
			t.Fatalf("unexpected path %q", gotPath)
		}
		_, _ = w.Write([]byte(`[{"runnerSnapshotId":"runner-snapshot-1","runnerId":"runner-1","runnerDomain":"runner.example.com"}]`))
	}))
	defer server.Close()

	dataSource := NewRunnersBySnapshotRefDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"ref": tftypes.String,
	}, map[string]tftypes.Value{
		"ref": tftypes.NewValue(tftypes.String, "snapshot-ref"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotRef != "snapshot-ref" {
		t.Fatalf("expected ref query %q, got %q", "snapshot-ref", gotRef)
	}

	var data runnersBySnapshotRefDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if len(data.Items) != 1 {
		t.Fatalf("expected 1 runner snapshot item, got %d", len(data.Items))
	}
	if data.Items[0].RunnerID.ValueString() != "runner-1" {
		t.Fatalf("expected runner ID %q, got %q", "runner-1", data.Items[0].RunnerID.ValueString())
	}
}

func runnerRelationshipDataSourceConfig(t *testing.T, dataSource datasource.DataSource, attributeTypes map[string]tftypes.Type, values map[string]tftypes.Value) *tfsdk.Config {
	t.Helper()

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	return &tfsdk.Config{
		Raw:    tftypes.NewValue(tftypes.Object{AttributeTypes: attributeTypes}, values),
		Schema: schemaResp.Schema,
	}
}

func runnerFullJSON(id string) string {
	return `{
  "id": "` + id + `",
  "domain": "runner.example.com",
  "apiUrl": "https://runner.example.com/api",
  "proxyUrl": "https://proxy.example.com",
  "cpu": 8,
  "memory": 32,
  "disk": 100,
  "gpu": 1,
  "gpuType": "H100",
  "sandboxClass": "container",
  "currentCpuUsagePercentage": 25.5,
  "currentMemoryUsagePercentage": 40.25,
  "currentDiskUsagePercentage": 10.5,
  "currentAllocatedCpu": 2,
  "currentAllocatedMemoryGiB": 8,
  "currentAllocatedDiskGiB": 20,
  "currentSnapshotCount": 3,
  "currentStartedSandboxes": 4,
  "availabilityScore": 0.92,
  "region": "custom-region",
  "name": "runner-name",
  "state": "ready",
  "lastChecked": "2026-06-11T00:00:00Z",
  "unschedulable": false,
  "tags": ["gpu", "custom"],
  "createdAt": "2026-06-10T00:00:00Z",
  "updatedAt": "2026-06-11T00:00:00Z",
  "version": "v0.187.0",
  "apiVersion": "v1",
  "runnerClass": "container",
  "appVersion": "v0.187.0",
  "apiKey": "runner-api-key",
  "regionType": "custom"
}`
}
