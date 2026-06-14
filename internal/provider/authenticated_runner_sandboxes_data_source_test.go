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

func TestAuthenticatedRunnerSandboxesDataSourceSchema(t *testing.T) {
	t.Parallel()

	dataSource := NewAuthenticatedRunnerSandboxesDataSource()

	var metadataResp datasource.MetadataResponse
	dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_authenticated_runner_sandboxes" {
		t.Fatalf("expected type name %q, got %q", "daytona_authenticated_runner_sandboxes", metadataResp.TypeName)
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

	statesAttr, ok := schemaResp.Schema.Attributes["states"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected states to be a string attribute, got %T", schemaResp.Schema.Attributes["states"])
	}
	if !statesAttr.Optional {
		t.Fatal("expected states to be optional")
	}
}

func TestAuthenticatedRunnerSandboxesDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPath, gotOrganizationID, gotStates, gotSkip string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get("X-Daytona-Organization-ID")
		gotStates = r.URL.Query().Get("states")
		gotSkip = r.URL.Query().Get("skipReconcilingSandboxes")
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodGet {
			t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
		}
		if gotPath != "/sandbox/for-runner" {
			t.Fatalf("unexpected path %q", gotPath)
		}

		_, _ = w.Write([]byte(`[{"id":"sandbox-1","organizationId":"org-1","name":"agent-runtime","snapshot":"snapshot-1","user":"user-1","env":{"TOKEN":"secret"},"labels":{"managed-by":"terraform"},"public":false,"networkBlockAll":false,"networkAllowList":"10.0.0.0/8","target":"region-1","cpu":2,"gpu":0,"memory":4,"disk":20,"state":"started","autoStopInterval":15,"autoArchiveInterval":60,"autoDeleteInterval":120,"runnerId":"runner-1","toolboxProxyUrl":"https://toolbox.example.com","createdAt":"2026-06-10T00:00:00Z","updatedAt":"2026-06-11T00:00:00Z"}]`))
	}))
	defer server.Close()

	dataSource := NewAuthenticatedRunnerSandboxesDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := runnerRelationshipDataSourceConfig(t, dataSource, map[string]tftypes.Type{
		"request_organization_id":    tftypes.String,
		"states":                     tftypes.String,
		"skip_reconciling_sandboxes": tftypes.Bool,
	}, map[string]tftypes.Value{
		"request_organization_id":    tftypes.NewValue(tftypes.String, "org-1"),
		"states":                     tftypes.NewValue(tftypes.String, "started,stopped"),
		"skip_reconciling_sandboxes": tftypes.NewValue(tftypes.Bool, true),
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
	if gotStates != "started,stopped" {
		t.Fatalf("expected states query %q, got %q", "started,stopped", gotStates)
	}
	if gotSkip != "true" {
		t.Fatalf("expected skip query %q, got %q", "true", gotSkip)
	}

	var data authenticatedRunnerSandboxesDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "authenticated_runner_sandboxes" {
		t.Fatalf("expected data source ID %q, got %q", "authenticated_runner_sandboxes", data.ID.ValueString())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected 1 sandbox, got %d", len(data.Items))
	}
	if data.Items[0].ID.ValueString() != "sandbox-1" {
		t.Fatalf("expected sandbox ID %q, got %q", "sandbox-1", data.Items[0].ID.ValueString())
	}
	if data.Items[0].RunnerID.ValueString() != "runner-1" {
		t.Fatalf("expected runner ID %q, got %q", "runner-1", data.Items[0].RunnerID.ValueString())
	}
}
