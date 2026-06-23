package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestJobStatusUpdateActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotAuthorization, gotUserAgent string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuthorization = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %s", err)
		}
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("failed unmarshalling request body %q: %s", string(body), err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"job-1","type":"snapshot","status":"COMPLETED","resourceType":"snapshot","resourceId":"snapshot-1","createdAt":"2026-06-22T00:00:00Z"}`))
	}))
	defer server.Close()

	actionInstance := NewJobStatusUpdateAction()
	configureActionClient(t, actionInstance, server.URL)

	config := newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"job_id":          tftypes.String,
		"status":          tftypes.String,
		"error_message":   tftypes.String,
		"result_metadata": tftypes.String,
	}}, map[string]tftypes.Value{
		"job_id":          terraformValue(t, types.StringValue("job-1")),
		"status":          terraformValue(t, types.StringValue("COMPLETED")),
		"error_message":   terraformValue(t, types.StringValue("")),
		"result_metadata": terraformValue(t, types.StringValue(`{"artifact":"snapshot-1"}`)),
	})

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/jobs/job-1/status" {
		t.Fatalf("expected path %q, got %q", "/jobs/job-1/status", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
	if gotPayload["status"] != "COMPLETED" {
		t.Fatalf("expected status payload %q, got %#v", "COMPLETED", gotPayload["status"])
	}
	if _, ok := gotPayload["errorMessage"]; ok {
		t.Fatalf("expected empty error_message to be omitted, got %#v", gotPayload)
	}
	if gotPayload["resultMetadata"] != `{"artifact":"snapshot-1"}` {
		t.Fatalf("expected result metadata payload, got %#v", gotPayload)
	}
}

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
