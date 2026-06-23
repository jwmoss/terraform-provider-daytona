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

func TestRunnerHealthcheckActionInvoke(t *testing.T) {
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

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	actionInstance := NewRunnerHealthcheckAction()
	configureActionClient(t, actionInstance, server.URL)

	config := newActionConfig(t, actionInstance, tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"app_version":         tftypes.String,
		"domain":              tftypes.String,
		"proxy_url":           tftypes.String,
		"api_url":             tftypes.String,
		"metrics_json":        tftypes.String,
		"service_health_json": tftypes.String,
	}}, map[string]tftypes.Value{
		"app_version": terraformValue(t, types.StringValue("v1.2.3")),
		"domain":      terraformValue(t, types.StringValue("runner.example.com")),
		"proxy_url":   terraformValue(t, types.StringValue("https://proxy.example.com")),
		"api_url":     terraformValue(t, types.StringValue("https://api.example.com")),
		"metrics_json": terraformValue(t, types.StringValue(`{
			"currentCpuLoadAverage": 1,
			"currentCpuUsagePercentage": 2,
			"currentMemoryUsagePercentage": 3,
			"currentDiskUsagePercentage": 4,
			"currentAllocatedCpu": 5,
			"currentAllocatedMemoryGiB": 6,
			"currentAllocatedDiskGiB": 7,
			"currentSnapshotCount": 8,
			"currentStartedSandboxes": 9,
			"cpu": 10,
			"memoryGiB": 11,
			"diskGiB": 12
		}`)),
		"service_health_json": terraformValue(t, types.StringValue(`[{"serviceName":"docker","healthy":true}]`)),
	})

	var invokeResp action.InvokeResponse
	actionInstance.Invoke(context.Background(), action.InvokeRequest{Config: *config}, &invokeResp)
	if invokeResp.Diagnostics.HasError() {
		t.Fatalf("unexpected invoke diagnostics: %s", invokeResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/runners/healthcheck" {
		t.Fatalf("expected path %q, got %q", "/runners/healthcheck", gotPath)
	}
	if gotAuthorization != "Bearer test-key" {
		t.Fatalf("expected bearer token header, got %q", gotAuthorization)
	}
	if gotUserAgent != "terraform-provider-daytona/test" {
		t.Fatalf("expected provider user agent, got %q", gotUserAgent)
	}
	if gotPayload["appVersion"] != "v1.2.3" || gotPayload["domain"] != "runner.example.com" {
		t.Fatalf("unexpected runner healthcheck payload: %#v", gotPayload)
	}
	metrics, ok := gotPayload["metrics"].(map[string]any)
	if !ok || metrics["cpu"] != float64(10) {
		t.Fatalf("expected metrics payload with cpu=10, got %#v", gotPayload["metrics"])
	}
	serviceHealth, ok := gotPayload["serviceHealth"].([]any)
	if !ok || len(serviceHealth) != 1 {
		t.Fatalf("expected one serviceHealth item, got %#v", gotPayload["serviceHealth"])
	}
}
