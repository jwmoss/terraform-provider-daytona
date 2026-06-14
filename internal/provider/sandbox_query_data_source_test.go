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

func TestSandboxQueryDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewSandboxQueryDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_sandbox_query" {
		t.Fatalf("expected type name %q, got %q", "daytona_sandbox_query", metadataResp.TypeName)
	}

	var schemaResp datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	itemsAttr, ok := schemaResp.Schema.Attributes["items"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected items to be a list nested attribute, got %T", schemaResp.Schema.Attributes["items"])
	}
	if !itemsAttr.Computed {
		t.Fatal("expected items to be computed")
	}

	statesAttr, ok := schemaResp.Schema.Attributes["states"].(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected states to be a list attribute, got %T", schemaResp.Schema.Attributes["states"])
	}
	if !statesAttr.Optional {
		t.Fatal("expected states to be optional")
	}
}

func TestSandboxQueryDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPath, gotOrganizationID string
	gotQuery := map[string][]string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get("X-Daytona-Organization-ID")
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
		}
		if gotPath != "/sandbox" {
			t.Fatalf("unexpected path %q", gotPath)
		}

		_, _ = w.Write([]byte(`{"items":[{"id":"sandbox-1","organizationId":"org-1","name":"agent-runtime","target":"region-1","runnerId":"runner-1","sandboxClass":"container","state":"started","desiredState":"started","snapshot":"snapshot-1","user":"user-1","errorReason":null,"recoverable":true,"public":false,"cpu":2,"gpu":0,"memory":4,"disk":20,"labels":{"managed-by":"terraform"},"backupState":"ready","autoStopInterval":15,"autoArchiveInterval":60,"autoDeleteInterval":120,"createdAt":"2026-06-10T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z","lastActivityAt":"2026-06-11T00:30:00Z","daemonVersion":"v0.187.0","toolboxProxyUrl":"https://toolbox.example.com"}],"nextCursor":"cursor-next"}`))
	}))
	defer server.Close()

	dataSource := NewSandboxQueryDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := sandboxQueryConfig(t, dataSource)

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
	assertQueryValue(t, gotQuery, "cursor", "cursor-1")
	assertQueryValue(t, gotQuery, "limit", "25")
	assertQueryValue(t, gotQuery, "id", "sand")
	assertQueryValue(t, gotQuery, "name", "agent")
	assertQueryValue(t, gotQuery, "labels", `{"managed-by":"terraform"}`)
	assertQueryValue(t, gotQuery, "includeErroredDeleted", "true")
	assertQueryValues(t, gotQuery, "states", []string{"started", "stopped"})
	assertQueryValues(t, gotQuery, "snapshots", []string{"snapshot-1"})
	assertQueryValues(t, gotQuery, "regionIds", []string{"region-1"})
	assertQueryValues(t, gotQuery, "sandboxClasses", []string{"container"})
	assertQueryValue(t, gotQuery, "minCpu", "1.5")
	assertQueryValue(t, gotQuery, "maxCpu", "4.5")
	assertQueryValue(t, gotQuery, "minMemoryGiB", "2")
	assertQueryValue(t, gotQuery, "maxMemoryGiB", "8")
	assertQueryValue(t, gotQuery, "minDiskGiB", "10")
	assertQueryValue(t, gotQuery, "maxDiskGiB", "100")
	assertQueryValue(t, gotQuery, "isPublic", "false")
	assertQueryValue(t, gotQuery, "isRecoverable", "true")
	assertQueryValue(t, gotQuery, "createdAtAfter", "2026-06-10T00:00:00Z")
	assertQueryValue(t, gotQuery, "createdAtBefore", "2026-06-12T00:00:00Z")
	assertQueryValue(t, gotQuery, "lastEventAfter", "2026-06-10T01:00:00Z")
	assertQueryValue(t, gotQuery, "lastEventBefore", "2026-06-12T01:00:00Z")
	assertQueryValue(t, gotQuery, "sort", "createdAt")
	assertQueryValue(t, gotQuery, "order", "desc")

	var data sandboxQueryDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.NextCursor.ValueString() != "cursor-next" {
		t.Fatalf("expected next cursor %q, got %q", "cursor-next", data.NextCursor.ValueString())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected 1 sandbox, got %d", len(data.Items))
	}
	if data.Items[0].ID.ValueString() != "sandbox-1" {
		t.Fatalf("expected sandbox ID %q, got %q", "sandbox-1", data.Items[0].ID.ValueString())
	}
	if data.Items[0].CPU.ValueFloat64() != 2 {
		t.Fatalf("expected CPU %g, got %g", 2.0, data.Items[0].CPU.ValueFloat64())
	}
	if !data.Items[0].Recoverable.ValueBool() {
		t.Fatal("expected recoverable to be true")
	}
}

func TestSandboxQueryDataSourceRejectsInvalidFilters(t *testing.T) {
	t.Parallel()

	dataSource := NewSandboxQueryDataSource()
	configureDataSource(t, dataSource, "https://example.com")

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"states":           tftypes.List{ElementType: tftypes.String},
		"sandbox_classes":  tftypes.List{ElementType: tftypes.String},
		"created_at_after": tftypes.String,
		"sort":             tftypes.String,
		"order":            tftypes.String,
	}, map[string]tftypes.Value{
		"states":           tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "bad-state")}),
		"sandbox_classes":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "bad-class")}),
		"created_at_after": tftypes.NewValue(tftypes.String, "not-a-time"),
		"sort":             tftypes.NewValue(tftypes.String, "bad-sort"),
		"order":            tftypes.NewValue(tftypes.String, "bad-order"),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for invalid filters")
	}
}

func sandboxQueryConfig(t *testing.T, dataSource datasource.DataSource) *tfsdk.Config {
	t.Helper()

	return runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"request_organization_id": tftypes.String,
		"cursor":                  tftypes.String,
		"limit":                   tftypes.Number,
		"id_prefix":               tftypes.String,
		"name_prefix":             tftypes.String,
		"labels_json":             tftypes.String,
		"include_errored_deleted": tftypes.Bool,
		"states":                  tftypes.List{ElementType: tftypes.String},
		"snapshots":               tftypes.List{ElementType: tftypes.String},
		"region_ids":              tftypes.List{ElementType: tftypes.String},
		"sandbox_classes":         tftypes.List{ElementType: tftypes.String},
		"min_cpu":                 tftypes.Number,
		"max_cpu":                 tftypes.Number,
		"min_memory_gib":          tftypes.Number,
		"max_memory_gib":          tftypes.Number,
		"min_disk_gib":            tftypes.Number,
		"max_disk_gib":            tftypes.Number,
		"is_public":               tftypes.Bool,
		"is_recoverable":          tftypes.Bool,
		"created_at_after":        tftypes.String,
		"created_at_before":       tftypes.String,
		"last_event_after":        tftypes.String,
		"last_event_before":       tftypes.String,
		"sort":                    tftypes.String,
		"order":                   tftypes.String,
	}, map[string]tftypes.Value{
		"request_organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"cursor":                  tftypes.NewValue(tftypes.String, "cursor-1"),
		"limit":                   tftypes.NewValue(tftypes.Number, 25),
		"id_prefix":               tftypes.NewValue(tftypes.String, "sand"),
		"name_prefix":             tftypes.NewValue(tftypes.String, "agent"),
		"labels_json":             tftypes.NewValue(tftypes.String, `{"managed-by":"terraform"}`),
		"include_errored_deleted": tftypes.NewValue(tftypes.Bool, true),
		"states":                  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "started"), tftypes.NewValue(tftypes.String, "stopped")}),
		"snapshots":               tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "snapshot-1")}),
		"region_ids":              tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "region-1")}),
		"sandbox_classes":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "container")}),
		"min_cpu":                 tftypes.NewValue(tftypes.Number, 1.5),
		"max_cpu":                 tftypes.NewValue(tftypes.Number, 4.5),
		"min_memory_gib":          tftypes.NewValue(tftypes.Number, 2),
		"max_memory_gib":          tftypes.NewValue(tftypes.Number, 8),
		"min_disk_gib":            tftypes.NewValue(tftypes.Number, 10),
		"max_disk_gib":            tftypes.NewValue(tftypes.Number, 100),
		"is_public":               tftypes.NewValue(tftypes.Bool, false),
		"is_recoverable":          tftypes.NewValue(tftypes.Bool, true),
		"created_at_after":        tftypes.NewValue(tftypes.String, "2026-06-10T00:00:00Z"),
		"created_at_before":       tftypes.NewValue(tftypes.String, "2026-06-12T00:00:00Z"),
		"last_event_after":        tftypes.NewValue(tftypes.String, "2026-06-10T01:00:00Z"),
		"last_event_before":       tftypes.NewValue(tftypes.String, "2026-06-12T01:00:00Z"),
		"sort":                    tftypes.NewValue(tftypes.String, "createdAt"),
		"order":                   tftypes.NewValue(tftypes.String, "desc"),
	})
}

func assertQueryValue(t *testing.T, query map[string][]string, key string, expected string) {
	t.Helper()
	values := query[key]
	if len(values) != 1 || values[0] != expected {
		t.Fatalf("expected query %s=%q, got %#v", key, expected, values)
	}
}

func assertQueryValues(t *testing.T, query map[string][]string, key string, expected []string) {
	t.Helper()
	values := query[key]
	if len(values) != len(expected) {
		t.Fatalf("expected query %s=%#v, got %#v", key, expected, values)
	}
	for i := range expected {
		if values[i] != expected[i] {
			t.Fatalf("expected query %s=%#v, got %#v", key, expected, values)
		}
	}
}
