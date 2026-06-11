// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOrganizationRegionQuotaResourceSchema(t *testing.T) {
	t.Parallel()

	quotaResource := NewOrganizationRegionQuotaResource()

	var metadataResp resource.MetadataResponse
	quotaResource.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_organization_region_quota" {
		t.Fatalf("expected type name %q, got %q", "daytona_organization_region_quota", metadataResp.TypeName)
	}

	var schemaResp resource.SchemaResponse
	quotaResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	for _, attributeName := range []string{"organization_id", "region_id", "sandbox_class"} {
		attr, ok := schemaResp.Schema.Attributes[attributeName].(schema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute, got %T", attributeName, schemaResp.Schema.Attributes[attributeName])
		}
		if !attr.Required {
			t.Fatalf("expected %s to be required", attributeName)
		}
	}

	for _, attributeName := range []string{"total_cpu_quota", "total_memory_quota", "total_disk_quota", "total_gpu_quota"} {
		attr, ok := schemaResp.Schema.Attributes[attributeName].(schema.Float64Attribute)
		if !ok {
			t.Fatalf("expected %s to be a float64 attribute, got %T", attributeName, schemaResp.Schema.Attributes[attributeName])
		}
		if !attr.Required {
			t.Fatalf("expected %s to be required", attributeName)
		}
	}

	allowedGPUAttr, ok := schemaResp.Schema.Attributes["allowed_gpu_types"].(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected allowed_gpu_types to be a list attribute, got %T", schemaResp.Schema.Attributes["allowed_gpu_types"])
	}
	if !allowedGPUAttr.Optional || !allowedGPUAttr.Computed {
		t.Fatal("expected allowed_gpu_types to be optional and computed")
	}
}

func TestAdminOrganizationRegionQuotaResourceSchema(t *testing.T) {
	t.Parallel()

	quotaResource := NewAdminOrganizationRegionQuotaResource()

	var metadataResp resource.MetadataResponse
	quotaResource.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "daytona"}, &metadataResp)
	if metadataResp.TypeName != "daytona_admin_organization_region_quota" {
		t.Fatalf("expected type name %q, got %q", "daytona_admin_organization_region_quota", metadataResp.TypeName)
	}

	var schemaResp resource.SchemaResponse
	quotaResource.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}

	for _, attributeName := range []string{"organization_id", "region_id", "sandbox_class"} {
		attr, ok := schemaResp.Schema.Attributes[attributeName].(schema.StringAttribute)
		if !ok {
			t.Fatalf("expected %s to be a string attribute, got %T", attributeName, schemaResp.Schema.Attributes[attributeName])
		}
		if !attr.Required {
			t.Fatalf("expected %s to be required", attributeName)
		}
	}

	for _, attributeName := range []string{"total_cpu_quota", "total_memory_quota", "total_disk_quota", "total_gpu_quota"} {
		attr, ok := schemaResp.Schema.Attributes[attributeName].(schema.Float64Attribute)
		if !ok {
			t.Fatalf("expected %s to be a float64 attribute, got %T", attributeName, schemaResp.Schema.Attributes[attributeName])
		}
		if !attr.Required {
			t.Fatalf("expected %s to be required", attributeName)
		}
	}
}

func TestOrganizationRegionQuotaResourceUpdateAndRead(t *testing.T) {
	t.Parallel()

	var patchPayload map[string]any
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "terraform-provider-daytona/test" {
			t.Fatalf("expected provider user agent, got %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get(organizationHeader) != "org-header" {
			t.Fatalf("expected organization header %q, got %q", "org-header", r.Header.Get(organizationHeader))
		}

		path := r.URL.EscapedPath()
		requests[path]++
		switch {
		case r.Method == http.MethodPatch && path == "/organizations/org-1/quota/region-1":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed reading body: %s", err)
			}
			if err := json.Unmarshal(body, &patchPayload); err != nil {
				t.Fatalf("failed unmarshalling patch payload %q: %s", string(body), err)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && path == "/organizations/org-1/usage":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"regionUsage": [
					{
						"regionId": "region-1",
						"sandboxClass": "container",
						"totalCpuQuota": 8,
						"currentCpuUsage": 1,
						"totalMemoryQuota": 16,
						"currentMemoryUsage": 2,
						"totalDiskQuota": 100,
						"currentDiskUsage": 3,
						"totalGpuQuota": 2,
						"currentGpuUsage": 0,
						"allowedGpuTypes": ["H100"],
						"maxCpuPerSandbox": 4,
						"maxMemoryPerSandbox": 8,
						"maxDiskPerSandbox": 50,
						"maxDiskPerNonEphemeralSandbox": 75,
						"maxCpuPerGpuSandbox": 6,
						"maxMemoryPerGpuSandbox": 12,
						"maxDiskPerGpuSandbox": 80
					}
				],
				"totalSnapshotQuota": 100,
				"currentSnapshotUsage": 0,
				"totalVolumeQuota": 100,
				"currentVolumeUsage": 0
			}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	quotaResource := &OrganizationRegionQuotaResource{client: newDaytonaClient(server.URL, "test-token", "org-header", "test")}
	data := organizationRegionQuotaResourceModel{
		OrganizationID:                types.StringValue("org-1"),
		RegionID:                      types.StringValue("region-1"),
		SandboxClass:                  types.StringValue("container"),
		TotalCPUQuota:                 types.Float64Value(8),
		TotalMemoryQuota:              types.Float64Value(16),
		TotalDiskQuota:                types.Float64Value(100),
		TotalGPUQuota:                 types.Float64Value(2),
		AllowedGPUTypes:               listStringValue(ctx, []string{"H100"}),
		MaxCPUPerSandbox:              types.Float64Value(4),
		MaxMemoryPerSandbox:           types.Float64Value(8),
		MaxDiskPerSandbox:             types.Float64Value(50),
		MaxDiskPerNonEphemeralSandbox: types.Float64Value(75),
		MaxCPUPerGPUSandbox:           types.Float64Value(6),
		MaxMemoryPerGPUSandbox:        types.Float64Value(12),
		MaxDiskPerGPUSandbox:          types.Float64Value(80),
	}

	var diags diag.Diagnostics
	if !quotaResource.applyOrganizationRegionQuota(ctx, data, &diags) {
		t.Fatalf("unexpected apply diagnostics: %s", diags)
	}

	read, found, _, err := quotaResource.readOrganizationRegionQuota(ctx, data)
	if err != nil {
		t.Fatalf("unexpected read error: %s", err)
	}
	if !found {
		t.Fatal("expected quota to be found")
	}

	if requests["/organizations/org-1/quota/region-1"] != 1 {
		t.Fatalf("expected one quota update request, got %d", requests["/organizations/org-1/quota/region-1"])
	}
	if requests["/organizations/org-1/usage"] != 1 {
		t.Fatalf("expected one usage read request, got %d", requests["/organizations/org-1/usage"])
	}

	if patchPayload["sandboxClass"] != "container" {
		t.Fatalf("expected sandboxClass container, got %#v", patchPayload["sandboxClass"])
	}
	if patchPayload["totalCpuQuota"] != float64(8) {
		t.Fatalf("expected totalCpuQuota 8, got %#v", patchPayload["totalCpuQuota"])
	}
	allowedGPU, ok := patchPayload["allowedGpuTypes"].([]any)
	if !ok || len(allowedGPU) != 1 || allowedGPU[0] != "H100" {
		t.Fatalf("expected allowedGpuTypes [H100], got %#v", patchPayload["allowedGpuTypes"])
	}
	if patchPayload["maxDiskPerGpuSandbox"] != float64(80) {
		t.Fatalf("expected maxDiskPerGpuSandbox 80, got %#v", patchPayload["maxDiskPerGpuSandbox"])
	}

	if read.ID.ValueString() != "org-1:region-1:container" {
		t.Fatalf("expected ID org-1:region-1:container, got %q", read.ID.ValueString())
	}
	if read.MaxDiskPerNonEphemeralSandbox.ValueFloat64() != 75 {
		t.Fatalf("expected max disk per non-ephemeral sandbox 75, got %f", read.MaxDiskPerNonEphemeralSandbox.ValueFloat64())
	}
}

func TestAdminOrganizationRegionQuotaResourceCRUDRequests(t *testing.T) {
	t.Parallel()

	var createPayload map[string]any
	var updatePayload map[string]any
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "terraform-provider-daytona/test" {
			t.Fatalf("expected provider user agent, got %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get(organizationHeader) != "org-header" {
			t.Fatalf("expected organization header %q, got %q", "org-header", r.Header.Get(organizationHeader))
		}

		path := r.URL.EscapedPath()
		requests[r.Method+" "+path]++
		switch {
		case r.Method == http.MethodPost && path == "/admin/organizations/org-1/quota/region-1":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed reading create body: %s", err)
			}
			if err := json.Unmarshal(body, &createPayload); err != nil {
				t.Fatalf("failed unmarshalling create payload %q: %s", string(body), err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(adminOrganizationRegionQuotaJSON(8, 16, 100, 2)))
		case r.Method == http.MethodGet && path == "/admin/organizations/org-1/quota/region-1/container":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(adminOrganizationRegionQuotaJSON(10, 20, 120, 3)))
		case r.Method == http.MethodPatch && path == "/admin/organizations/org-1/quota/region-1":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed reading update body: %s", err)
			}
			if err := json.Unmarshal(body, &updatePayload); err != nil {
				t.Fatalf("failed unmarshalling update payload %q: %s", string(body), err)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && path == "/admin/organizations/org-1/quota/region-1/container":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	quotaResource := &AdminOrganizationRegionQuotaResource{client: newDaytonaClient(server.URL, "test-token", "org-header", "test")}
	data := organizationRegionQuotaResourceModel{
		OrganizationID:                types.StringValue("org-1"),
		RegionID:                      types.StringValue("region-1"),
		SandboxClass:                  types.StringValue("container"),
		TotalCPUQuota:                 types.Float64Value(8),
		TotalMemoryQuota:              types.Float64Value(16),
		TotalDiskQuota:                types.Float64Value(100),
		TotalGPUQuota:                 types.Float64Value(2),
		AllowedGPUTypes:               listStringValue(ctx, []string{"H100"}),
		MaxCPUPerSandbox:              types.Float64Value(4),
		MaxMemoryPerSandbox:           types.Float64Value(8),
		MaxDiskPerSandbox:             types.Float64Value(50),
		MaxDiskPerNonEphemeralSandbox: types.Float64Value(75),
		MaxCPUPerGPUSandbox:           types.Float64Value(6),
		MaxMemoryPerGPUSandbox:        types.Float64Value(12),
		MaxDiskPerGPUSandbox:          types.Float64Value(80),
	}

	var diags diag.Diagnostics
	created, ok := quotaResource.createAdminOrganizationRegionQuota(ctx, data, &diags)
	if !ok || diags.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", diags)
	}
	read, found, _, err := quotaResource.readAdminOrganizationRegionQuota(ctx, data)
	if err != nil {
		t.Fatalf("unexpected read error: %s", err)
	}
	if !found {
		t.Fatal("expected admin quota to be found")
	}
	if !quotaResource.updateAdminOrganizationRegionQuota(ctx, data, &diags) || diags.HasError() {
		t.Fatalf("unexpected update diagnostics: %s", diags)
	}
	if !quotaResource.deleteAdminOrganizationRegionQuota(ctx, data, &diags) || diags.HasError() {
		t.Fatalf("unexpected delete diagnostics: %s", diags)
	}

	for _, key := range []string{
		"POST /admin/organizations/org-1/quota/region-1",
		"GET /admin/organizations/org-1/quota/region-1/container",
		"PATCH /admin/organizations/org-1/quota/region-1",
		"DELETE /admin/organizations/org-1/quota/region-1/container",
	} {
		if requests[key] != 1 {
			t.Fatalf("expected one %s request, got %d", key, requests[key])
		}
	}

	if createPayload["sandboxClass"] != "container" {
		t.Fatalf("expected create sandboxClass container, got %#v", createPayload["sandboxClass"])
	}
	if createPayload["totalCpuQuota"] != float64(8) {
		t.Fatalf("expected create totalCpuQuota 8, got %#v", createPayload["totalCpuQuota"])
	}
	allowedGPU, ok := createPayload["allowedGpuTypes"].([]any)
	if !ok || len(allowedGPU) != 1 || allowedGPU[0] != "H100" {
		t.Fatalf("expected create allowedGpuTypes [H100], got %#v", createPayload["allowedGpuTypes"])
	}
	if updatePayload["maxDiskPerGpuSandbox"] != float64(80) {
		t.Fatalf("expected update maxDiskPerGpuSandbox 80, got %#v", updatePayload["maxDiskPerGpuSandbox"])
	}
	if created.ID.ValueString() != "org-1:region-1:container" {
		t.Fatalf("expected created ID org-1:region-1:container, got %q", created.ID.ValueString())
	}
	if read.TotalCPUQuota.ValueFloat64() != 10 {
		t.Fatalf("expected read total CPU quota 10, got %f", read.TotalCPUQuota.ValueFloat64())
	}
}

func TestOrganizationRegionQuotaResourcePayloadValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	base := organizationRegionQuotaResourceModel{
		OrganizationID:   types.StringValue("org-1"),
		RegionID:         types.StringValue("region-1"),
		SandboxClass:     types.StringValue("container"),
		TotalCPUQuota:    types.Float64Value(8),
		TotalMemoryQuota: types.Float64Value(16),
		TotalDiskQuota:   types.Float64Value(100),
		TotalGPUQuota:    types.Float64Value(2),
		AllowedGPUTypes:  listStringValue(ctx, []string{"H100"}),
	}

	invalidSandboxClass := base
	invalidSandboxClass.SandboxClass = types.StringValue("bad-class")
	var sandboxClassDiags diag.Diagnostics
	if _, ok := organizationRegionQuotaPayload(ctx, invalidSandboxClass, &sandboxClassDiags); ok {
		t.Fatal("expected invalid sandbox class validation to fail")
	}
	if !sandboxClassDiags.HasError() {
		t.Fatal("expected invalid sandbox class diagnostics")
	}

	invalidGPUType := base
	invalidGPUType.AllowedGPUTypes = listStringValue(ctx, []string{"bad-gpu"})
	var gpuDiags diag.Diagnostics
	if _, ok := organizationRegionQuotaPayload(ctx, invalidGPUType, &gpuDiags); ok {
		t.Fatal("expected invalid GPU type validation to fail")
	}
	if !gpuDiags.HasError() {
		t.Fatal("expected invalid GPU type diagnostics")
	}
}

func adminOrganizationRegionQuotaJSON(cpu, memory, disk, gpu int) string {
	return fmt.Sprintf(`{
		"organizationId": "org-1",
		"regionId": "region-1",
		"sandboxClass": "container",
		"totalCpuQuota": %d,
		"totalMemoryQuota": %d,
		"totalDiskQuota": %d,
		"totalGpuQuota": %d,
		"allowedGpuTypes": ["H100"],
		"maxCpuPerSandbox": 4,
		"maxMemoryPerSandbox": 8,
		"maxDiskPerSandbox": 50,
		"maxDiskPerNonEphemeralSandbox": 75,
		"maxCpuPerGpuSandbox": 6,
		"maxMemoryPerGpuSandbox": 12,
		"maxDiskPerGpuSandbox": 80
	}`, cpu, memory, disk, gpu)
}
