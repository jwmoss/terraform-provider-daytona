package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestOrganizationMemberAccessResourceReadContract(t *testing.T) {
	t.Parallel()

	runResourceReadContractTest(t, "/organizations/org-1/users",
		func(client *daytonaClient) resource.Resource {
			return &OrganizationMemberAccessResource{client: client}
		},
		map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "user-1"),
			"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
			"user_id":         tftypes.NewValue(tftypes.String, "user-1"),
		},
		[]resourceReadContractCase{
			{name: "api error keeps state", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantError: true},
			{name: "not found removes state", statusCode: http.StatusNotFound, body: `{"message":"missing"}`, wantRemoved: true},
			// A member list without the user means they left or were removed
			// out-of-band, so the resource must leave state.
			{name: "missing member removes state", statusCode: http.StatusOK, body: `[]`, wantRemoved: true},
		})
}

func TestOrganizationMemberAccessResourceCreateAndRead(t *testing.T) {
	t.Parallel()

	var accessPayload map[string]any
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
		case r.Method == http.MethodPost && path == "/organizations/org-1/users/user-1/access":
			decodeTestPayload(t, r.Body, &accessPayload)
			_, _ = w.Write([]byte(organizationMemberJSON()))
		case r.Method == http.MethodGet && path == "/organizations/org-1/users":
			_, _ = w.Write([]byte(`[` + organizationMemberJSON() + `]`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	memberResource := &OrganizationMemberAccessResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	memberSchema := resourceTestSchema(t, memberResource)

	createPlan := resourceTestPlan(t, memberSchema, map[string]tftypes.Value{
		"organization_id":   tftypes.NewValue(tftypes.String, "org-1"),
		"user_id":           tftypes.NewValue(tftypes.String, "user-1"),
		"role":              tftypes.NewValue(tftypes.String, "member"),
		"assigned_role_ids": tftypesStringSet("role-1"),
	})
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: memberSchema}}
	memberResource.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}
	if accessPayload["role"] != "member" {
		t.Fatalf("expected access payload role member, got %#v", accessPayload["role"])
	}
	assignedRoles, ok := accessPayload["assignedRoleIds"].([]any)
	if !ok || len(assignedRoles) != 1 || assignedRoles[0] != "role-1" {
		t.Fatalf("expected access assignedRoleIds [role-1], got %#v", accessPayload["assignedRoleIds"])
	}

	var created organizationMemberAccessResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("unexpected created state diagnostics: %s", diags)
	}
	// The member user ID doubles as the resource ID so imports and reads agree.
	if created.ID.ValueString() != "user-1" {
		t.Fatalf("expected created member ID user-1, got %q", created.ID.ValueString())
	}
	if created.Email.ValueString() != "member@example.com" {
		t.Fatalf("expected created member email member@example.com, got %q", created.Email.ValueString())
	}

	// Read must refresh drifted attributes from the matching member entry.
	state := resourceTestState(t, memberSchema, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "user-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"user_id":         tftypes.NewValue(tftypes.String, "user-1"),
		"name":            tftypes.NewValue(tftypes.String, "stale-name"),
	})
	readResp := resource.ReadResponse{State: state}
	memberResource.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}
	var read organizationMemberAccessResourceModel
	if diags := readResp.State.Get(ctx, &read); diags.HasError() {
		t.Fatalf("unexpected read state diagnostics: %s", diags)
	}
	if read.Name.ValueString() != "Member One" {
		t.Fatalf("expected refreshed member name, got %q", read.Name.ValueString())
	}

	if requests["POST /organizations/org-1/users/user-1/access"] != 1 {
		t.Fatalf("expected one access update request, got %d", requests["POST /organizations/org-1/users/user-1/access"])
	}
	if requests["GET /organizations/org-1/users"] != 1 {
		t.Fatalf("expected one member list request, got %d", requests["GET /organizations/org-1/users"])
	}
}

func TestOrganizationMemberAccessResourceDeleteContract(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{name: "deleted", statusCode: http.StatusNoContent},
		// A member already removed out-of-band must not fail destroy.
		{name: "already removed", statusCode: http.StatusNotFound},
		{name: "api error fails destroy", statusCode: http.StatusInternalServerError, wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.EscapedPath() != "/organizations/org-1/users/user-1" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(testCase.statusCode)
				if testCase.statusCode != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"message":"detail"}`))
				}
			}))
			defer server.Close()

			memberResource := &OrganizationMemberAccessResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
			state := resourceTestState(t, resourceTestSchema(t, memberResource), map[string]tftypes.Value{
				"id":              tftypes.NewValue(tftypes.String, "user-1"),
				"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
				"user_id":         tftypes.NewValue(tftypes.String, "user-1"),
			})

			deleteResp := resource.DeleteResponse{}
			memberResource.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleteResp)

			if deleteResp.Diagnostics.HasError() != testCase.wantError {
				t.Fatalf("expected error=%t, got diagnostics: %s", testCase.wantError, deleteResp.Diagnostics)
			}
		})
	}
}

func organizationMemberJSON() string {
	return `{
		"userId": "user-1",
		"organizationId": "org-1",
		"name": "Member One",
		"email": "member@example.com",
		"role": "member",
		"assignedRoles": [` + organizationRoleJSON("Can deploy") + `],
		"createdAt": "2026-06-10T00:00:00Z",
		"updatedAt": "2026-06-11T00:00:00Z"
	}`
}
