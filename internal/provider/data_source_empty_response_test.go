package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDataSourcesRejectEmptySuccessfulResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		source    datasource.DataSource
		configure func(*testing.T, datasource.DataSource) *tfsdk.Config
	}{
		{
			name:   "current API key",
			source: NewCurrentAPIKeyDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, nil)
			},
		},
		{
			name:   "sandboxes collection",
			source: NewSandboxesDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, nil)
			},
		},
		{
			name:   "snapshots collection",
			source: NewSnapshotsDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, nil)
			},
		},
		{
			name:   "configuration",
			source: NewConfigDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, nil)
			},
		},
		{
			name:   "current user",
			source: NewCurrentUserDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, nil)
			},
		},
		{
			name:   "organization usage",
			source: NewOrganizationUsageDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, map[string]tftypes.Value{
					"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
				})
			},
		},
		{
			name:   "organization audit logs",
			source: NewOrganizationAuditLogsDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, map[string]tftypes.Value{
					"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
				})
			},
		},
		{
			name:   "job",
			source: NewJobDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, map[string]tftypes.Value{
					"id": tftypes.NewValue(tftypes.String, "job-1"),
				})
			},
		},
		{
			name:   "jobs",
			source: NewJobsDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, nil)
			},
		},
		{
			name:   "organization OpenTelemetry config",
			source: NewOrganizationOtelConfigDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return dataSourceTestConfig(t, source, map[string]tftypes.Value{
					"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
				})
			},
		},
		{
			name:   "admin audit logs",
			source: NewAdminAuditLogsDataSource(),
			configure: func(t *testing.T, source datasource.DataSource) *tfsdk.Config {
				return runnerRelationshipDataSourceConfig(t, source, map[string]tftypes.Type{
					"page":   tftypes.Number,
					"limit":  tftypes.Number,
					"from":   tftypes.String,
					"to":     tftypes.String,
					"cursor": tftypes.String,
				}, map[string]tftypes.Value{
					"page":   tftypes.NewValue(tftypes.Number, 1),
					"limit":  tftypes.NewValue(tftypes.Number, 100),
					"from":   tftypes.NewValue(tftypes.String, "2026-06-11T00:00:00Z"),
					"to":     tftypes.NewValue(tftypes.String, "2026-06-12T00:00:00Z"),
					"cursor": tftypes.NewValue(tftypes.String, "cursor-1"),
				})
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var requestCount atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("expected GET request, got %s", r.Method)
				}
				requestCount.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			configureDataSource(t, testCase.source, server.URL)
			config := testCase.configure(t, testCase.source)

			var readResp datasource.ReadResponse
			readResp.State = tfsdk.State{Schema: config.Schema}
			testCase.source.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected diagnostics for empty successful API response")
			}
			if requestCount.Load() == 0 {
				t.Fatal("expected data source to call the API")
			}
		})
	}
}

func dataSourceTestConfig(t *testing.T, source datasource.DataSource, values map[string]tftypes.Value) *tfsdk.Config {
	t.Helper()

	var schemaResp datasource.SchemaResponse
	source.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	objectType, ok := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("expected data source schema object type, got %T", schemaResp.Schema.Type().TerraformType(context.Background()))
	}

	configValues := map[string]tftypes.Value{}
	for name, attributeType := range objectType.AttributeTypes {
		value, ok := values[name]
		if !ok {
			value = tftypes.NewValue(attributeType, nil)
		}
		configValues[name] = value
	}
	for name := range values {
		if _, ok := objectType.AttributeTypes[name]; !ok {
			t.Fatalf("unknown data source attribute %q", name)
		}
	}

	return &tfsdk.Config{
		Raw:    tftypes.NewValue(objectType, configValues),
		Schema: schemaResp.Schema,
	}
}
