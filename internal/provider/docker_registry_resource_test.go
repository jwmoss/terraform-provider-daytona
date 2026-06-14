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

func TestFlattenDockerRegistry(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 2, 13, 30, 0, 0, time.UTC)
	registry := &apiclient.DockerRegistry{
		Id:           "reg-1",
		Name:         "registry",
		Url:          "https://registry.example.com",
		Username:     "robot",
		Project:      "team-a",
		RegistryType: "organization",
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	prior := dockerRegistryResourceModel{
		ID:       types.StringUnknown(),
		Password: types.StringValue("p4ss"),
	}
	flattened := flattenDockerRegistry(registry, prior)

	if flattened.ID.ValueString() != "reg-1" {
		t.Fatalf("expected ID reg-1, got %q", flattened.ID.ValueString())
	}
	if flattened.Name.ValueString() != "registry" {
		t.Fatalf("expected name registry, got %q", flattened.Name.ValueString())
	}
	if flattened.URL.ValueString() != "https://registry.example.com" {
		t.Fatalf("expected URL %q, got %q", "https://registry.example.com", flattened.URL.ValueString())
	}
	if flattened.Username.ValueString() != "robot" {
		t.Fatalf("expected username robot, got %q", flattened.Username.ValueString())
	}
	if flattened.Project.ValueString() != "team-a" {
		t.Fatalf("expected project team-a, got %q", flattened.Project.ValueString())
	}
	if flattened.RegistryType.ValueString() != "organization" {
		t.Fatalf("expected registry_type organization, got %q", flattened.RegistryType.ValueString())
	}
	if flattened.CreatedAt.ValueString() != "2026-06-01T12:00:00Z" {
		t.Fatalf("expected created_at %q, got %q", "2026-06-01T12:00:00Z", flattened.CreatedAt.ValueString())
	}
	if flattened.UpdatedAt.ValueString() != "2026-06-02T13:30:00Z" {
		t.Fatalf("expected updated_at %q, got %q", "2026-06-02T13:30:00Z", flattened.UpdatedAt.ValueString())
	}
	// The API never returns the password, so flatten must keep the configured one.
	if flattened.Password.ValueString() != "p4ss" {
		t.Fatalf("expected configured password to be kept, got %q", flattened.Password.ValueString())
	}

	priorName := types.StringValue("untouched")
	unchanged := flattenDockerRegistry(nil, dockerRegistryResourceModel{Name: priorName})
	if unchanged.Name != priorName {
		t.Fatalf("expected nil registry to leave prior model unchanged, got %#v", unchanged.Name)
	}
}

func dockerRegistryResourceJSON(name string) string {
	payload := map[string]any{
		"id":           "reg-1",
		"name":         name,
		"url":          "https://registry.example.com",
		"username":     "robot",
		"project":      "team-a",
		"registryType": "organization",
		"createdAt":    "2026-06-01T12:00:00Z",
		"updatedAt":    "2026-06-02T13:30:00Z",
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func TestDockerRegistryResourceCreateRequest(t *testing.T) {
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
		_, _ = w.Write([]byte(dockerRegistryResourceJSON("registry")))
	}))
	defer server.Close()

	registryResource := &DockerRegistryResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, registryResource, map[string]tftypes.Value{
		"name":     tftypes.NewValue(tftypes.String, "registry"),
		"url":      tftypes.NewValue(tftypes.String, "https://registry.example.com"),
		"username": tftypes.NewValue(tftypes.String, "robot"),
		"password": tftypes.NewValue(tftypes.String, "p4ss"),
		"project":  tftypes.NewValue(tftypes.String, "team-a"),
	})

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	registryResource.Create(context.Background(), resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("unexpected create diagnostics: %s", createResp.Diagnostics)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, gotMethod)
	}
	if gotPath != "/docker-registry" {
		t.Fatalf("expected path %q, got %q", "/docker-registry", gotPath)
	}
	for key, expected := range map[string]string{
		"name":     "registry",
		"url":      "https://registry.example.com",
		"username": "robot",
		"password": "p4ss",
		"project":  "team-a",
	} {
		if createPayload[key] != expected {
			t.Fatalf("expected payload %s %q, got %#v", key, expected, createPayload[key])
		}
	}

	var data dockerRegistryResourceModel
	createResp.State.Get(context.Background(), &data)
	if data.ID.ValueString() != "reg-1" {
		t.Fatalf("expected state ID reg-1, got %q", data.ID.ValueString())
	}
	if data.RegistryType.ValueString() != "organization" {
		t.Fatalf("expected state registry_type organization, got %q", data.RegistryType.ValueString())
	}
	if data.Password.ValueString() != "p4ss" {
		t.Fatalf("expected state password to keep configured value, got %q", data.Password.ValueString())
	}
}

func TestDockerRegistryResourceUpdateRequest(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	var updatePayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading body: %s", err)
		}
		if err := json.Unmarshal(body, &updatePayload); err != nil {
			t.Fatalf("failed unmarshalling update payload %q: %s", string(body), err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(dockerRegistryResourceJSON("renamed")))
	}))
	defer server.Close()

	registryResource := &DockerRegistryResource{client: newDaytonaClient(server.URL, "test-token", "org-1", "test")}
	plan := resourcePlan(t, registryResource, map[string]tftypes.Value{
		"id":       tftypes.NewValue(tftypes.String, "reg-1"),
		"name":     tftypes.NewValue(tftypes.String, "renamed"),
		"url":      tftypes.NewValue(tftypes.String, "https://registry.example.com"),
		"username": tftypes.NewValue(tftypes.String, "robot"),
		"password": tftypes.NewValue(tftypes.String, "rotated"),
		"project":  tftypes.NewValue(tftypes.String, "team-a"),
	})

	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: plan.Schema}}
	registryResource.Update(context.Background(), resource.UpdateRequest{Plan: plan}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("unexpected update diagnostics: %s", updateResp.Diagnostics)
	}

	if gotMethod != http.MethodPatch {
		t.Fatalf("expected method %s, got %s", http.MethodPatch, gotMethod)
	}
	if gotPath != "/docker-registry/reg-1" {
		t.Fatalf("expected path %q, got %q", "/docker-registry/reg-1", gotPath)
	}
	for key, expected := range map[string]string{
		"name":     "renamed",
		"url":      "https://registry.example.com",
		"username": "robot",
		"password": "rotated",
		"project":  "team-a",
	} {
		if updatePayload[key] != expected {
			t.Fatalf("expected payload %s %q, got %#v", key, expected, updatePayload[key])
		}
	}

	var data dockerRegistryResourceModel
	updateResp.State.Get(context.Background(), &data)
	if data.Name.ValueString() != "renamed" {
		t.Fatalf("expected state name renamed, got %q", data.Name.ValueString())
	}
}
