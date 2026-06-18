package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSandboxResourceSchemaServerDefaults(t *testing.T) {
	t.Parallel()

	sandboxResource := NewSandboxResource()

	var schemaResp resource.SchemaResponse
	sandboxResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	for _, attrName := range []string{"name", "user", "target"} {
		attr, ok := schemaResp.Schema.Attributes[attrName].(schema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute, got %T", attrName, schemaResp.Schema.Attributes[attrName])
		}
		if !attr.Optional || !attr.Computed {
			t.Fatalf("expected %s to be optional and computed", attrName)
		}
		if !hasStringPlanModifierDescription(attr, "Once set, the value of this attribute in state will not change.") {
			t.Fatalf("expected %s to use state for unknown values", attrName)
		}
		if !hasStringPlanModifierDescription(attr, "If the value of this attribute changes, Terraform will destroy and recreate the resource.") {
			t.Fatalf("expected %s to require replacement on changes", attrName)
		}
	}

	for _, attrName := range []string{"cpu", "memory", "disk", "auto_stop_interval", "auto_archive_interval", "auto_delete_interval"} {
		attr, ok := schemaResp.Schema.Attributes[attrName].(schema.Int64Attribute)
		if !ok {
			t.Fatalf("expected %s to be an int64 attribute, got %T", attrName, schemaResp.Schema.Attributes[attrName])
		}
		if !attr.Optional || !attr.Computed {
			t.Fatalf("expected %s to be optional and computed", attrName)
		}
		if !hasInt64PlanModifierDescription(attr, "Once set, the value of this attribute in state will not change.") {
			t.Fatalf("expected %s to use state for unknown values", attrName)
		}
	}

	gpuAttr, ok := schemaResp.Schema.Attributes["gpu"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected gpu to be an int64 attribute, got %T", schemaResp.Schema.Attributes["gpu"])
	}
	if !gpuAttr.Optional || !gpuAttr.Computed {
		t.Fatal("expected gpu to be optional and computed")
	}
	if !hasInt64PlanModifierDescription(gpuAttr, "Once set, the value of this attribute in state will not change.") {
		t.Fatal("expected gpu to use state for unknown values")
	}
	if !hasInt64PlanModifierDescription(gpuAttr, "If the value of this attribute changes, Terraform will destroy and recreate the resource.") {
		t.Fatal("expected gpu to require replacement on changes")
	}

	if _, ok := schemaResp.Schema.Attributes["gpu_type"].(schema.StringAttribute); !ok {
		t.Fatalf("expected gpu_type to be a string attribute, got %T", schemaResp.Schema.Attributes["gpu_type"])
	}
	if _, ok := schemaResp.Schema.Attributes["gpu_types"].(schema.ListAttribute); !ok {
		t.Fatalf("expected gpu_types to be a list attribute, got %T", schemaResp.Schema.Attributes["gpu_types"])
	}
	if _, ok := schemaResp.Schema.Attributes["volumes"].(schema.ListNestedAttribute); !ok {
		t.Fatalf("expected volumes to be a list nested attribute, got %T", schemaResp.Schema.Attributes["volumes"])
	}
	if _, ok := schemaResp.Schema.Attributes["build_info"].(schema.SingleNestedAttribute); !ok {
		t.Fatalf("expected build_info to be a single nested attribute, got %T", schemaResp.Schema.Attributes["build_info"])
	}
	if _, ok := schemaResp.Schema.Attributes["last_activity_at"].(schema.StringAttribute); !ok {
		t.Fatalf("expected last_activity_at to be a string attribute, got %T", schemaResp.Schema.Attributes["last_activity_at"])
	}
}

func TestSandboxResourceUpdateClearsIntervals(t *testing.T) {
	t.Parallel()

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		requests[r.Method+" "+path]++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && path == "/sandbox/sandbox-1/autostop/0":
			_, _ = w.Write([]byte(sandboxIntervalJSON(60, 120)))
		case r.Method == http.MethodPost && path == "/sandbox/sandbox-1/autoarchive/0":
			_, _ = w.Write([]byte(sandboxIntervalJSON(0, 120)))
		case r.Method == http.MethodPost && path == "/sandbox/sandbox-1/autodelete/-1":
			_, _ = w.Write([]byte(sandboxIntervalJSON(0, -1)))
		case r.Method == http.MethodGet && path == "/sandbox/sandbox-1":
			_, _ = w.Write([]byte(sandboxIntervalJSON(0, -1)))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	sandboxResource := &SandboxResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	sandboxSchema := resourceTestSchema(t, sandboxResource)
	plan := resourceTestPlan(t, sandboxSchema, map[string]tftypes.Value{
		"id":                    tftypes.NewValue(tftypes.String, "sandbox-1"),
		"auto_stop_interval":    tftypes.NewValue(tftypes.Number, nil),
		"auto_archive_interval": tftypes.NewValue(tftypes.Number, nil),
		"auto_delete_interval":  tftypes.NewValue(tftypes.Number, nil),
	})
	state := resourceTestState(t, sandboxSchema, map[string]tftypes.Value{
		"id":                    tftypes.NewValue(tftypes.String, "sandbox-1"),
		"auto_stop_interval":    tftypes.NewValue(tftypes.Number, 15),
		"auto_archive_interval": tftypes.NewValue(tftypes.Number, 60),
		"auto_delete_interval":  tftypes.NewValue(tftypes.Number, 120),
	})

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: sandboxSchema}}
	sandboxResource.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %s", updateResp.Diagnostics)
	}

	for _, key := range []string{
		"POST /sandbox/sandbox-1/autostop/0",
		"POST /sandbox/sandbox-1/autoarchive/0",
		"POST /sandbox/sandbox-1/autodelete/-1",
		"GET /sandbox/sandbox-1",
	} {
		if requests[key] != 1 {
			t.Fatalf("expected one %s request, got %d; all requests: %#v", key, requests[key], requests)
		}
	}
}

func sandboxIntervalJSON(autoArchive, autoDelete int) string {
	return fmt.Sprintf(`{
		"id": "sandbox-1",
		"organizationId": "org-1",
		"name": "sandbox",
		"user": "daytona",
		"env": {},
		"labels": {},
		"public": false,
		"networkBlockAll": false,
		"target": "us",
		"cpu": 2,
		"gpu": 0,
		"memory": 4,
		"disk": 10,
		"state": "started",
		"autoStopInterval": 0,
		"autoArchiveInterval": %d,
		"autoDeleteInterval": %d,
		"createdAt": "2026-06-01T12:00:00Z",
		"updatedAt": "2026-06-01T12:05:00Z",
		"toolboxProxyUrl": "https://toolbox.example"
	}`, autoArchive, autoDelete)
}

func TestSandboxResourceCreateRequestAdvancedFields(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var createPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(organizationHeader) != "org-1" {
			t.Fatalf("expected organization header %q, got %q", "org-1", r.Header.Get(organizationHeader))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %s", err)
		}
		if err := json.Unmarshal(body, &createPayload); err != nil {
			t.Fatalf("failed unmarshalling create payload %q: %s", string(body), err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "sandbox-1",
			"organizationId": "org-1",
			"name": "sandbox",
			"snapshot": "daytona-small",
			"user": "daytona",
			"env": {},
			"labels": {"team": "platform"},
			"public": false,
			"networkBlockAll": false,
			"target": "us",
			"cpu": 2,
			"gpu": 1,
			"gpuType": "H100",
			"memory": 4,
			"disk": 10,
			"state": "started",
			"autoStopInterval": 15,
			"autoArchiveInterval": 0,
			"autoDeleteInterval": -1,
			"volumes": [
				{"volumeId": "vol-1", "mountPath": "/workspace/data", "subpath": "datasets"}
			],
			"buildInfo": {
				"dockerfileContent": "FROM ubuntu:24.04\nRUN echo hi\n",
				"contextHashes": ["hash-1"],
				"createdAt": "2026-06-01T12:00:00Z",
				"updatedAt": "2026-06-01T12:00:00Z",
				"snapshotRef": "snapshot-ref-1"
			},
			"createdAt": "2026-06-01T12:00:00Z",
			"updatedAt": "2026-06-01T12:05:00Z",
			"lastActivityAt": "2026-06-01T12:06:00Z",
			"toolboxProxyUrl": "https://toolbox.example"
		}`))
	}))
	defer server.Close()

	volumeType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"volume_id":  tftypes.String,
		"mount_path": tftypes.String,
		"subpath":    tftypes.String,
	}}
	buildInfoType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"dockerfile_content": tftypes.String,
		"context_hashes":     tftypes.List{ElementType: tftypes.String},
		"created_at":         tftypes.String,
		"updated_at":         tftypes.String,
		"snapshot_ref":       tftypes.String,
	}}

	sandboxResource := &SandboxResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, sandboxResource, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "sandbox"),
		"snapshot": tftypes.NewValue(tftypes.String, "daytona-small"),
		"gpu":      tftypes.NewValue(tftypes.Number, 1),
		"gpu_types": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "H100"),
			tftypes.NewValue(tftypes.String, "RTX-PRO-6000"),
		}),
		"volumes": tftypes.NewValue(tftypes.List{ElementType: volumeType}, []tftypes.Value{
			tftypes.NewValue(volumeType, map[string]tftypes.Value{
				"volume_id":  tftypes.NewValue(tftypes.String, "vol-1"),
				"mount_path": tftypes.NewValue(tftypes.String, "/workspace/data"),
				"subpath":    tftypes.NewValue(tftypes.String, "datasets"),
			}),
		}),
		"build_info": tftypes.NewValue(buildInfoType, map[string]tftypes.Value{
			"dockerfile_content": tftypes.NewValue(tftypes.String, "FROM ubuntu:24.04\nRUN echo hi\n"),
			"context_hashes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "hash-1"),
			}),
			"created_at":   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			"updated_at":   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			"snapshot_ref": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		}),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	sandboxResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/sandbox" {
		t.Fatalf("expected path %q, got %q", "/sandbox", gotPath)
	}
	if createPayload["name"] != "sandbox" {
		t.Fatalf("expected payload name sandbox, got %#v", createPayload["name"])
	}

	gpuTypes, ok := createPayload["gpuType"].([]any)
	if !ok || len(gpuTypes) != 2 || gpuTypes[0] != "H100" || gpuTypes[1] != "RTX-PRO-6000" {
		t.Fatalf("expected payload gpuType [H100 RTX-PRO-6000], got %#v", createPayload["gpuType"])
	}

	volumes, ok := createPayload["volumes"].([]any)
	if !ok || len(volumes) != 1 {
		t.Fatalf("expected one payload volume, got %#v", createPayload["volumes"])
	}
	volume, ok := volumes[0].(map[string]any)
	if !ok || volume["volumeId"] != "vol-1" || volume["mountPath"] != "/workspace/data" || volume["subpath"] != "datasets" {
		t.Fatalf("expected payload volume with id/path/subpath, got %#v", volumes[0])
	}

	buildInfo, ok := createPayload["buildInfo"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload buildInfo object, got %#v", createPayload["buildInfo"])
	}
	if buildInfo["dockerfileContent"] != "FROM ubuntu:24.04\nRUN echo hi\n" {
		t.Fatalf("expected payload Dockerfile content, got %#v", buildInfo["dockerfileContent"])
	}
	contextHashes, ok := buildInfo["contextHashes"].([]any)
	if !ok || len(contextHashes) != 1 || contextHashes[0] != "hash-1" {
		t.Fatalf("expected payload contextHashes [hash-1], got %#v", buildInfo["contextHashes"])
	}

	var data sandboxResourceModel
	createResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "sandbox-1" {
		t.Fatalf("expected state ID sandbox-1, got %q", data.ID.ValueString())
	}
	if data.GPUType.ValueString() != "H100" {
		t.Fatalf("expected state gpu_type H100, got %q", data.GPUType.ValueString())
	}
	if data.LastActivityAt.ValueString() != "2026-06-01T12:06:00Z" {
		t.Fatalf("expected state last_activity_at, got %q", data.LastActivityAt.ValueString())
	}

	var stateVolumes []sandboxVolumeModel
	diags := data.Volumes.ElementsAs(context.Background(), &stateVolumes, false)
	if diags.HasError() {
		t.Fatalf("unexpected volumes diagnostics: %s", diags)
	}
	if len(stateVolumes) != 1 || stateVolumes[0].VolumeID.ValueString() != "vol-1" || stateVolumes[0].Subpath.ValueString() != "datasets" {
		t.Fatalf("expected state volume vol-1/datasets, got %#v", stateVolumes)
	}

	var stateBuildInfo buildInfoModel
	diags = data.BuildInfo.As(context.Background(), &stateBuildInfo, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		t.Fatalf("unexpected build_info diagnostics: %s", diags)
	}
	if stateBuildInfo.SnapshotRef.ValueString() != "snapshot-ref-1" {
		t.Fatalf("expected state build_info snapshot_ref snapshot-ref-1, got %q", stateBuildInfo.SnapshotRef.ValueString())
	}
	if stateBuildInfo.CreatedAt.ValueString() != time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339) {
		t.Fatalf("expected state build_info created_at, got %q", stateBuildInfo.CreatedAt.ValueString())
	}
}

func hasStringPlanModifierDescription(attr schema.StringAttribute, description string) bool {
	for _, modifier := range attr.PlanModifiers {
		if modifier.Description(context.Background()) == description {
			return true
		}
	}

	return false
}

func hasInt64PlanModifierDescription(attr schema.Int64Attribute, description string) bool {
	for _, modifier := range attr.PlanModifiers {
		if modifier.Description(context.Background()) == description {
			return true
		}
	}

	return false
}
