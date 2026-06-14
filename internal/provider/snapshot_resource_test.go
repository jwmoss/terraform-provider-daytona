package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFlattenSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	createdAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 2, 13, 30, 0, 0, time.UTC)
	size := float32(2.5)
	snapshot := &apiclient.SnapshotDto{
		Id:             "snap-1",
		OrganizationId: apiclient.PtrString("org-1"),
		General:        true,
		Name:           "snapshot",
		ImageName:      apiclient.PtrString("ubuntu:22.04"),
		State:          apiclient.SNAPSHOTSTATE_ACTIVE,
		Size:           *apiclient.NewNullableFloat32(&size),
		Entrypoint:     []string{"sleep", "infinity"},
		Cpu:            2,
		Gpu:            1,
		Mem:            4,
		Disk:           10,
		ErrorReason:    *apiclient.NewNullableString(apiclient.PtrString("boom")),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		RegionIds:      []string{"us"},
		Ref:            apiclient.PtrString("ref-1"),
		SandboxClass:   apiclient.PtrString("container"),
	}

	flattened := flattenSnapshot(ctx, snapshot, snapshotResourceModel{ID: types.StringUnknown()})

	if flattened.ID.ValueString() != "snap-1" {
		t.Fatalf("expected ID snap-1, got %q", flattened.ID.ValueString())
	}
	if flattened.Name.ValueString() != "snapshot" {
		t.Fatalf("expected name snapshot, got %q", flattened.Name.ValueString())
	}
	if flattened.OrganizationID.ValueString() != "org-1" {
		t.Fatalf("expected organization_id org-1, got %q", flattened.OrganizationID.ValueString())
	}
	if !flattened.General.ValueBool() {
		t.Fatal("expected general true")
	}
	if flattened.State.ValueString() != "active" {
		t.Fatalf("expected state active, got %q", flattened.State.ValueString())
	}
	if flattened.ImageName.ValueString() != "ubuntu:22.04" {
		t.Fatalf("expected image_name ubuntu:22.04, got %q", flattened.ImageName.ValueString())
	}
	if flattened.CPU.ValueInt64() != 2 || flattened.GPU.ValueInt64() != 1 || flattened.Memory.ValueInt64() != 4 || flattened.Disk.ValueInt64() != 10 {
		t.Fatalf("expected cpu/gpu/memory/disk 2/1/4/10, got %d/%d/%d/%d",
			flattened.CPU.ValueInt64(), flattened.GPU.ValueInt64(), flattened.Memory.ValueInt64(), flattened.Disk.ValueInt64())
	}
	if flattened.Size.ValueString() != "2.5" {
		t.Fatalf("expected size 2.5, got %q", flattened.Size.ValueString())
	}
	if flattened.ErrorReason.ValueString() != "boom" {
		t.Fatalf("expected error_reason boom, got %q", flattened.ErrorReason.ValueString())
	}
	if flattened.Ref.ValueString() != "ref-1" {
		t.Fatalf("expected ref ref-1, got %q", flattened.Ref.ValueString())
	}
	if flattened.SandboxClass.ValueString() != "container" {
		t.Fatalf("expected sandbox_class container, got %q", flattened.SandboxClass.ValueString())
	}
	if flattened.CreatedAt.ValueString() != "2026-06-01T12:00:00Z" {
		t.Fatalf("expected created_at %q, got %q", "2026-06-01T12:00:00Z", flattened.CreatedAt.ValueString())
	}
	if flattened.UpdatedAt.ValueString() != "2026-06-02T13:30:00Z" {
		t.Fatalf("expected updated_at %q, got %q", "2026-06-02T13:30:00Z", flattened.UpdatedAt.ValueString())
	}

	entrypoint := []string{}
	diags := flattened.Entrypoint.ElementsAs(ctx, &entrypoint, false)
	if diags.HasError() {
		t.Fatalf("unexpected entrypoint diagnostics: %s", diags)
	}
	if len(entrypoint) != 2 || entrypoint[0] != "sleep" || entrypoint[1] != "infinity" {
		t.Fatalf("expected entrypoint [sleep infinity], got %#v", entrypoint)
	}

	regionIDs := []string{}
	diags = flattened.RegionIDs.ElementsAs(ctx, &regionIDs, false)
	if diags.HasError() {
		t.Fatalf("unexpected region_ids diagnostics: %s", diags)
	}
	if len(regionIDs) != 1 || regionIDs[0] != "us" {
		t.Fatalf("expected region_ids [us], got %#v", regionIDs)
	}

	priorName := types.StringValue("untouched")
	unchanged := flattenSnapshot(ctx, nil, snapshotResourceModel{Name: priorName})
	if unchanged.Name != priorName {
		t.Fatalf("expected nil snapshot to leave prior model unchanged, got %#v", unchanged.Name)
	}
}

func TestFlattenSnapshotNilOptionalFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	snapshot := &apiclient.SnapshotDto{
		Id:        "snap-1",
		Name:      "snapshot",
		State:     apiclient.SNAPSHOTSTATE_PENDING,
		CreatedAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	prior := snapshotResourceModel{
		ImageName:    types.StringUnknown(),
		SandboxClass: types.StringUnknown(),
	}
	flattened := flattenSnapshot(ctx, snapshot, prior)

	if !flattened.OrganizationID.IsNull() {
		t.Fatalf("expected nil organization ID to flatten to null, got %#v", flattened.OrganizationID)
	}
	if !flattened.ImageName.IsNull() {
		t.Fatalf("expected unknown image_name to flatten to null, got %#v", flattened.ImageName)
	}
	if !flattened.SandboxClass.IsNull() {
		t.Fatalf("expected unknown sandbox_class to flatten to null, got %#v", flattened.SandboxClass)
	}
	if !flattened.Ref.IsNull() {
		t.Fatalf("expected nil ref to flatten to null, got %#v", flattened.Ref)
	}
	if !flattened.Size.IsNull() {
		t.Fatalf("expected unset size to flatten to null, got %#v", flattened.Size)
	}
	if !flattened.ErrorReason.IsNull() {
		t.Fatalf("expected unset error reason to flatten to null, got %#v", flattened.ErrorReason)
	}

	// A configured image name must survive a response that omits it.
	configured := flattenSnapshot(ctx, snapshot, snapshotResourceModel{ImageName: types.StringValue("ubuntu:22.04")})
	if configured.ImageName.ValueString() != "ubuntu:22.04" {
		t.Fatalf("expected configured image_name to be kept, got %q", configured.ImageName.ValueString())
	}
}

func TestSnapshotResourceCreateRequest(t *testing.T) {
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
		if r.Header.Get("User-Agent") != "terraform-provider-daytona/test" {
			t.Fatalf("expected provider user agent, got %q", r.Header.Get("User-Agent"))
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
			"id": "snap-1",
			"organizationId": "org-1",
			"general": false,
			"name": "snapshot",
			"imageName": "ubuntu:22.04",
			"state": "pending",
			"size": null,
			"entrypoint": ["sleep", "infinity"],
			"cpu": 2,
			"gpu": 0,
			"mem": 4,
			"disk": 10,
			"errorReason": null,
			"createdAt": "2026-06-01T12:00:00Z",
			"updatedAt": "2026-06-01T12:00:00Z",
			"lastUsedAt": null,
			"regionIds": ["us"],
			"sandboxClass": "container"
		}`))
	}))
	defer server.Close()

	snapshotResource := &SnapshotResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, snapshotResource, map[string]tftypes.Value{
		"name":       tftypes.NewValue(tftypes.String, "snapshot"),
		"image_name": tftypes.NewValue(tftypes.String, "ubuntu:22.04"),
		"entrypoint": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "sleep"),
			tftypes.NewValue(tftypes.String, "infinity"),
		}),
		"cpu":           tftypes.NewValue(tftypes.Number, 2),
		"gpu":           tftypes.NewValue(tftypes.Number, nil),
		"memory":        tftypes.NewValue(tftypes.Number, 4),
		"disk":          tftypes.NewValue(tftypes.Number, 10),
		"region_id":     tftypes.NewValue(tftypes.String, "us"),
		"sandbox_class": tftypes.NewValue(tftypes.String, "container"),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	snapshotResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/snapshots" {
		t.Fatalf("expected path %q, got %q", "/snapshots", gotPath)
	}
	if createPayload["name"] != "snapshot" {
		t.Fatalf("expected payload name snapshot, got %#v", createPayload["name"])
	}
	if createPayload["imageName"] != "ubuntu:22.04" {
		t.Fatalf("expected payload imageName ubuntu:22.04, got %#v", createPayload["imageName"])
	}
	if createPayload["regionId"] != "us" {
		t.Fatalf("expected payload regionId us, got %#v", createPayload["regionId"])
	}
	if createPayload["sandboxClass"] != "container" {
		t.Fatalf("expected payload sandboxClass container, got %#v", createPayload["sandboxClass"])
	}
	if createPayload["cpu"] != float64(2) || createPayload["memory"] != float64(4) || createPayload["disk"] != float64(10) {
		t.Fatalf("expected payload cpu/memory/disk 2/4/10, got %#v/%#v/%#v",
			createPayload["cpu"], createPayload["memory"], createPayload["disk"])
	}
	if _, present := createPayload["gpu"]; present {
		t.Fatalf("expected null gpu to be omitted from payload, got %#v", createPayload["gpu"])
	}
	entrypoint, ok := createPayload["entrypoint"].([]any)
	if !ok || len(entrypoint) != 2 || entrypoint[0] != "sleep" || entrypoint[1] != "infinity" {
		t.Fatalf("expected payload entrypoint [sleep infinity], got %#v", createPayload["entrypoint"])
	}

	var data snapshotResourceModel
	createResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "snap-1" {
		t.Fatalf("expected state ID snap-1, got %q", data.ID.ValueString())
	}
	if data.State.ValueString() != "pending" {
		t.Fatalf("expected state pending, got %q", data.State.ValueString())
	}
	if data.OrganizationID.ValueString() != "org-1" {
		t.Fatalf("expected state organization_id org-1, got %q", data.OrganizationID.ValueString())
	}
}
