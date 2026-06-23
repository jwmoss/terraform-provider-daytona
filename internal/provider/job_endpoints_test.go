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

func TestJobPollDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotPath, gotTimeout, gotLimit string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotTimeout = r.URL.Query().Get("timeout")
		gotLimit = r.URL.Query().Get("limit")
		if r.Method != http.MethodGet {
			t.Fatalf("expected method %s, got %s", http.MethodGet, r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jobs":[{"id":"job-1","type":"snapshot","status":"PENDING","resourceType":"snapshot","resourceId":"snapshot-1","createdAt":"2026-06-22T00:00:00Z"}]}`))
	}))
	defer server.Close()

	dataSource := NewJobPollDataSource()
	configureDataSource(t, dataSource, server.URL)

	config := dataSourceTestConfig(t, dataSource, map[string]tftypes.Value{
		"timeout": tftypes.NewValue(tftypes.Number, 1),
		"limit":   tftypes.NewValue(tftypes.Number, 2),
	})

	var readResp datasource.ReadResponse
	readResp.State = tfsdk.State{Schema: config.Schema}
	dataSource.Read(context.Background(), datasource.ReadRequest{Config: *config}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}

	if gotPath != "/jobs/poll" {
		t.Fatalf("expected path %q, got %q", "/jobs/poll", gotPath)
	}
	if gotTimeout != "1" || gotLimit != "2" {
		t.Fatalf("expected timeout=1 and limit=2, got timeout=%q limit=%q", gotTimeout, gotLimit)
	}

	var data jobPollDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "job_poll" {
		t.Fatalf("expected ID %q, got %q", "job_poll", data.ID.ValueString())
	}
	if len(data.Items) != 1 {
		t.Fatalf("expected 1 job, got %d", len(data.Items))
	}
	if data.Items[0].ID.ValueString() != "job-1" {
		t.Fatalf("expected job ID %q, got %q", "job-1", data.Items[0].ID.ValueString())
	}
}
