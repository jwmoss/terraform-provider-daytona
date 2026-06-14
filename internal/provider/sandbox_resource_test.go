package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
