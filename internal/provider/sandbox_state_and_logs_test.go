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

func TestSandboxUpdateStateActionInvoke(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotOrganizationID string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotOrganizationID = r.Header.Get(organizationHeader)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %s", err)
		}
		if err := json.Unmarshal(body, &gotPayload); err != nil {
			t.Fatalf("failed unmarshalling request body %q: %s", string(body), err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	actionInstance := NewSandboxUpdateStateAction()
	configureActionClient(t, actionInstance, server.URL)

	config := newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"sandbox_id":      tftypes.String,
		"state":           tftypes.String,
		"error_reason":    tftypes.String,
		"recoverable":     tftypes.Bool,
		"organization_id": tftypes.String,
	}}, map[string]tftypes.Value{
		"sandbox_id":      terraformValue(t, types.StringValue("sandbox/1")),
		"state":           terraformValue(t, types.StringValue("error")),
		"error_reason":    terraformValue(t, types.StringValue("runner failed")),
		"recoverable":     terraformValue(t, types.BoolValue(true)),
		"organization_id": terraformValue(t, types.StringValue("org-1")),
	})

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("expected method %s, got %s", http.MethodPut, gotMethod)
	}
	if gotPath != "/sandbox/sandbox%2F1/state" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox%2F1/state", gotPath)
	}
	if gotOrganizationID != "org-1" {
		t.Fatalf("expected organization header %q, got %q", "org-1", gotOrganizationID)
	}
	if gotPayload["state"] != "error" || gotPayload["errorReason"] != "runner failed" || gotPayload["recoverable"] != true {
		t.Fatalf("unexpected sandbox state payload: %#v", gotPayload)
	}
}

func TestSandboxBuildLogsDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotFollow, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotFollow = r.URL.Query().Get("follow")
		gotOrganizationID = r.Header.Get(organizationHeader)
		_, _ = w.Write([]byte("sandbox log line\n"))
	}))
	defer server.Close()

	dataSource := NewSandboxBuildLogsDataSource()
	configureDataSource(t, dataSource, server.URL)
	config := dataSourceTestConfig(t, dataSource, map[string]tftypes.Value{
		"sandbox_id_or_name": tftypes.NewValue(tftypes.String, "sandbox/1"),
		"organization_id":    tftypes.NewValue(tftypes.String, "org-1"),
		"follow":             tftypes.NewValue(tftypes.Bool, false),
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
	if gotPath != "/sandbox/sandbox%2F1/build-logs" {
		t.Fatalf("expected path %q, got %q", "/sandbox/sandbox%2F1/build-logs", gotPath)
	}
	if gotFollow != "false" || gotOrganizationID != "org-1" {
		t.Fatalf("expected follow=false and org header, got follow=%q org=%q", gotFollow, gotOrganizationID)
	}

	var data sandboxBuildLogsDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.Logs.ValueString() != "sandbox log line\n" {
		t.Fatalf("unexpected sandbox logs %q", data.Logs.ValueString())
	}
}

func TestSnapshotBuildLogsDataSourceRead(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotFollow, gotOrganizationID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotFollow = r.URL.Query().Get("follow")
		gotOrganizationID = r.Header.Get(organizationHeader)
		_, _ = w.Write([]byte("snapshot log line\n"))
	}))
	defer server.Close()

	dataSource := NewSnapshotBuildLogsDataSource()
	configureDataSource(t, dataSource, server.URL)
	config := dataSourceTestConfig(t, dataSource, map[string]tftypes.Value{
		"snapshot_id":     tftypes.NewValue(tftypes.String, "snapshot/1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"follow":          tftypes.NewValue(tftypes.Bool, true),
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
	if gotPath != "/snapshots/snapshot%2F1/build-logs" {
		t.Fatalf("expected path %q, got %q", "/snapshots/snapshot%2F1/build-logs", gotPath)
	}
	if gotFollow != "true" || gotOrganizationID != "org-1" {
		t.Fatalf("expected follow=true and org header, got follow=%q org=%q", gotFollow, gotOrganizationID)
	}

	var data snapshotBuildLogsDataSourceModel
	readResp.State.Get(context.Background(), &data)
	if data.Logs.ValueString() != "snapshot log line\n" {
		t.Fatalf("unexpected snapshot logs %q", data.Logs.ValueString())
	}
}
