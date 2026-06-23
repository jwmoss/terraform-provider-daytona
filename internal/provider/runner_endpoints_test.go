package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestAuthenticatedRunnerDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(runnerFullJSON("runner-1")))
	}))
	defer server.Close()

	dataSource := NewAuthenticatedRunnerDataSource()
	configureDataSource(t, dataSource, server.URL)
	config := dataSourceTestConfig(t, dataSource, map[string]tftypes.Value{})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected method %s, got %s", http.MethodGet, gotMethod)
	}
	if gotPath != "/runners/me" {
		t.Fatalf("expected path %q, got %q", "/runners/me", gotPath)
	}

	var data runnerFullDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "runner-1" {
		t.Fatalf("expected runner ID %q, got %q", "runner-1", data.ID.ValueString())
	}
	if data.Name.ValueString() != "runner-name" {
		t.Fatalf("expected runner name %q, got %q", "runner-name", data.Name.ValueString())
	}
}
