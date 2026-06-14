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

func TestOrganizationInvitationResourceReadContract(t *testing.T) {
	t.Parallel()

	runResourceReadContractTest(t, "/organizations/org-1/invitations",
		func(client *daytonaClient) resource.Resource {
			return &OrganizationInvitationResource{client: client}
		},
		map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "inv-1"),
			"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		},
		[]resourceReadContractCase{
			{name: "api error keeps state", statusCode: http.StatusInternalServerError, body: `{"message":"boom"}`, wantError: true},
			{name: "not found removes state", statusCode: http.StatusNotFound, body: `{"message":"missing"}`, wantRemoved: true},
			// An empty invitation list means it was accepted, declined, or
			// cancelled out-of-band, so the resource must leave state.
			{name: "missing invitation removes state", statusCode: http.StatusOK, body: `[]`, wantRemoved: true},
		})
}

func TestOrganizationInvitationResourceCRUDRequests(t *testing.T) {
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
		case r.Method == http.MethodPost && path == "/organizations/org-1/invitations":
			decodeTestPayload(t, r.Body, &createPayload)
			_, _ = w.Write([]byte(organizationInvitationJSON("member")))
		case r.Method == http.MethodGet && path == "/organizations/org-1/invitations":
			_, _ = w.Write([]byte(`[` + organizationInvitationJSON("member") + `]`))
		case r.Method == http.MethodPut && path == "/organizations/org-1/invitations/inv-1":
			decodeTestPayload(t, r.Body, &updatePayload)
			_, _ = w.Write([]byte(organizationInvitationJSON("owner")))
		case r.Method == http.MethodPost && path == "/organizations/org-1/invitations/inv-1/cancel":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	invitationResource := &OrganizationInvitationResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	invitationSchema := resourceTestSchema(t, invitationResource)

	createPlan := resourceTestPlan(t, invitationSchema, map[string]tftypes.Value{
		"organization_id":   tftypes.NewValue(tftypes.String, "org-1"),
		"email":             tftypes.NewValue(tftypes.String, "member@example.com"),
		"role":              tftypes.NewValue(tftypes.String, "member"),
		"assigned_role_ids": tftypesStringSet("role-1"),
		"expires_at":        tftypes.NewValue(tftypes.String, "2026-12-31T23:59:59Z"),
	})
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: invitationSchema}}
	invitationResource.Create(ctx, resource.CreateRequest{Plan: createPlan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}
	if createPayload["email"] != "member@example.com" || createPayload["role"] != "member" {
		t.Fatalf("expected create payload email/role, got %#v", createPayload)
	}
	assignedRoles, ok := createPayload["assignedRoleIds"].([]any)
	if !ok || len(assignedRoles) != 1 || assignedRoles[0] != "role-1" {
		t.Fatalf("expected create assignedRoleIds [role-1], got %#v", createPayload["assignedRoleIds"])
	}
	if createPayload["expiresAt"] != "2026-12-31T23:59:59Z" {
		t.Fatalf("expected create expiresAt RFC3339, got %#v", createPayload["expiresAt"])
	}

	var created organizationInvitationResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("unexpected created state diagnostics: %s", diags)
	}
	if created.ID.ValueString() != "inv-1" {
		t.Fatalf("expected created invitation ID inv-1, got %q", created.ID.ValueString())
	}
	if created.Status.ValueString() != "pending" {
		t.Fatalf("expected created status pending, got %q", created.Status.ValueString())
	}
	if created.InvitedBy.ValueString() != "owner@example.com" {
		t.Fatalf("expected invited_by owner@example.com, got %q", created.InvitedBy.ValueString())
	}

	// Read must refresh drifted attributes from the matching list entry.
	state := resourceTestState(t, invitationSchema, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "inv-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
		"email":           tftypes.NewValue(tftypes.String, "stale@example.com"),
	})
	readResp := resource.ReadResponse{State: state}
	invitationResource.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("unexpected read diagnostics: %s", readResp.Diagnostics)
	}
	var read organizationInvitationResourceModel
	if diags := readResp.State.Get(ctx, &read); diags.HasError() {
		t.Fatalf("unexpected read state diagnostics: %s", diags)
	}
	if read.Email.ValueString() != "member@example.com" {
		t.Fatalf("expected refreshed email member@example.com, got %q", read.Email.ValueString())
	}

	updatePlan := resourceTestPlan(t, invitationSchema, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "inv-1"),
		"organization_id":   tftypes.NewValue(tftypes.String, "org-1"),
		"email":             tftypes.NewValue(tftypes.String, "member@example.com"),
		"role":              tftypes.NewValue(tftypes.String, "owner"),
		"assigned_role_ids": tftypesStringSet("role-1"),
	})
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: invitationSchema}}
	invitationResource.Update(ctx, resource.UpdateRequest{Plan: updatePlan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %s", updateResp.Diagnostics)
	}
	if updatePayload["role"] != "owner" {
		t.Fatalf("expected update role owner, got %#v", updatePayload["role"])
	}
	var updated organizationInvitationResourceModel
	if diags := updateResp.State.Get(ctx, &updated); diags.HasError() {
		t.Fatalf("unexpected updated state diagnostics: %s", diags)
	}
	if updated.Role.ValueString() != "owner" {
		t.Fatalf("expected updated role owner from response, got %q", updated.Role.ValueString())
	}

	deleteState := resourceTestState(t, invitationSchema, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "inv-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
	})
	deleteResp := resource.DeleteResponse{}
	invitationResource.Delete(ctx, resource.DeleteRequest{State: deleteState}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("unexpected delete diagnostics: %s", deleteResp.Diagnostics)
	}

	for _, key := range []string{
		"POST /organizations/org-1/invitations",
		"GET /organizations/org-1/invitations",
		"PUT /organizations/org-1/invitations/inv-1",
		"POST /organizations/org-1/invitations/inv-1/cancel",
	} {
		if requests[key] != 1 {
			t.Fatalf("expected one %s request, got %d", key, requests[key])
		}
	}
}

func TestOrganizationInvitationResourceCreateRejectsInvalidExpiresAt(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer server.Close()

	invitationResource := &OrganizationInvitationResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	invitationSchema := resourceTestSchema(t, invitationResource)
	plan := resourceTestPlan(t, invitationSchema, map[string]tftypes.Value{
		"organization_id":   tftypes.NewValue(tftypes.String, "org-1"),
		"email":             tftypes.NewValue(tftypes.String, "member@example.com"),
		"role":              tftypes.NewValue(tftypes.String, "member"),
		"assigned_role_ids": tftypesStringSet(),
		"expires_at":        tftypes.NewValue(tftypes.String, "tomorrow"),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: invitationSchema}}
	invitationResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)

	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected invalid expires_at diagnostics")
	}
	// Validation must run before the API call so a bad timestamp never
	// creates an invitation that Terraform then loses track of.
	if requestCount != 0 {
		t.Fatalf("expected no API requests for invalid expires_at, got %d", requestCount)
	}
}

func organizationInvitationJSON(role string) string {
	return fmt.Sprintf(`{
		"id": "inv-1",
		"email": "member@example.com",
		"invitedBy": "owner@example.com",
		"organizationId": "org-1",
		"organizationName": "engineering",
		"expiresAt": "2026-12-31T23:59:59Z",
		"status": "pending",
		"role": %q,
		"assignedRoles": [%s],
		"createdAt": "2026-06-10T00:00:00Z",
		"updatedAt": "2026-06-11T00:00:00Z"
	}`, role, organizationRoleJSON("Can deploy"))
}
