package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestOrganizationRoleResourceReadContract(t *testing.T) {
	t.Parallel()

	runResourceReadContractTest(t, "/organizations/org-1/roles",
		func(client *daytonaClient) resource.Resource {
			return &OrganizationRoleResource{client: client}
		},
		map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "role-1"),
			"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		},
		[]resourceReadContractCase{
			{name: "api error keeps state", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantError: true},
			{name: "not found removes state", statusCode: http.StatusNotFound, body: `{"message":"missing"}`, wantRemoved: true},
			// An empty role list means the role was deleted out-of-band, so
			// Terraform must plan a re-create instead of erroring.
			{name: "missing role removes state", statusCode: http.StatusOK, body: `[]`, wantRemoved: true},
		})
}

func TestOrganizationRoleResourceCRUDRequests(t *testing.T) {
	t.Parallel()

	var createPayload, updatePayload map[string]any
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get(organizationHeader) != "org-1" {
			t.Errorf("expected organization header %q, got %q", "org-1", r.Header.Get(organizationHeader))
		}

		path := r.URL.EscapedPath()
		requests[r.Method+" "+path]++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && path == "/organizations/org-1/roles":
			decodeTestPayload(t, r.Body, &createPayload)
			_, _ = w.Write([]byte(organizationRoleJSON("Can deploy")))
		case r.Method == http.MethodGet && path == "/organizations/org-1/roles":
			_, _ = w.Write([]byte(`[` + organizationRoleJSON("Can deploy") + `]`))
		case r.Method == http.MethodPut && path == "/organizations/org-1/roles/role-1":
			decodeTestPayload(t, r.Body, &updatePayload)
			_, _ = w.Write([]byte(organizationRoleJSON("Can deploy and read")))
		case r.Method == http.MethodDelete && path == "/organizations/org-1/roles/role-1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	roleResource := &OrganizationRoleResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	roleSchema := resourceTestSchema(t, roleResource)

	createPlan := resourceTestPlan(t, roleSchema, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"name":            tftypes.NewValue(tftypes.String, "deployer"),
		"description":     tftypes.NewValue(tftypes.String, "Can deploy"),
		"permissions":     tftypesStringSet("write:sandboxes"),
	})
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: roleSchema}}
	roleResource.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}
	if createPayload["name"] != "deployer" || createPayload["description"] != "Can deploy" {
		t.Fatalf("expected create payload name/description, got %#v", createPayload)
	}
	createPermissions, ok := createPayload["permissions"].([]any)
	if !ok || len(createPermissions) != 1 || createPermissions[0] != "write:sandboxes" {
		t.Fatalf("expected create permissions [write:sandboxes], got %#v", createPayload["permissions"])
	}

	var created organizationRoleResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("unexpected created state diagnostics: %s", diags)
	}
	if created.ID.ValueString() != "role-1" {
		t.Fatalf("expected created role ID role-1, got %q", created.ID.ValueString())
	}
	if created.IsGlobal.ValueBool() {
		t.Fatal("expected created role is_global false")
	}

	// Read must refresh drifted attributes from the matching list entry.
	state := resourceTestState(t, roleSchema, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "role-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"name":            tftypes.NewValue(tftypes.String, "stale-name"),
	})
	readResp := resource.ReadResponse{State: state}
	roleResource.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}
	var read organizationRoleResourceModel
	if diags := readResp.State.Get(ctx, &read); diags.HasError() {
		t.Fatalf("unexpected read state diagnostics: %s", diags)
	}
	if read.Name.ValueString() != "deployer" {
		t.Fatalf("expected refreshed role name deployer, got %q", read.Name.ValueString())
	}

	updatePlan := resourceTestPlan(t, roleSchema, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "role-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"name":            tftypes.NewValue(tftypes.String, "deployer"),
		"description":     tftypes.NewValue(tftypes.String, "Can deploy and read"),
		"permissions":     tftypesStringSet("write:sandboxes"),
	})
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: roleSchema}}
	roleResource.Update(ctx, resource.UpdateRequest{Plan: updatePlan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %s", updateResp.Diagnostics)
	}
	if updatePayload["description"] != "Can deploy and read" {
		t.Fatalf("expected update description, got %#v", updatePayload["description"])
	}
	var updated organizationRoleResourceModel
	if diags := updateResp.State.Get(ctx, &updated); diags.HasError() {
		t.Fatalf("unexpected updated state diagnostics: %s", diags)
	}
	if updated.Description.ValueString() != "Can deploy and read" {
		t.Fatalf("expected updated description from response, got %q", updated.Description.ValueString())
	}

	deleteState := resourceTestState(t, roleSchema, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "role-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
	})
	deleteResp := resource.DeleteResponse{}
	roleResource.Delete(ctx, resource.DeleteRequest{State: deleteState}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected delete diagnostics: %s", deleteResp.Diagnostics)
	}

	for _, key := range []string{
		"POST /organizations/org-1/roles",
		"GET /organizations/org-1/roles",
		"PUT /organizations/org-1/roles/role-1",
		"DELETE /organizations/org-1/roles/role-1",
	} {
		if requests[key] != 1 {
			t.Fatalf("expected one %s request, got %d", key, requests[key])
		}
	}
}

func organizationRoleJSON(description string) string {
	return fmt.Sprintf(`{
		"id": "role-1",
		"name": "deployer",
		"description": %q,
		"permissions": ["write:sandboxes"],
		"isGlobal": false,
		"createdAt": "2026-06-10T00:00:00Z",
		"updatedAt": "2026-06-11T00:00:00Z"
	}`, description)
}
