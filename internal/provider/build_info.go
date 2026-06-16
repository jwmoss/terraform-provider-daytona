package provider

import (
	"context"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type buildInfoModel struct {
	DockerfileContent types.String `tfsdk:"dockerfile_content"`
	ContextHashes     types.List   `tfsdk:"context_hashes"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	SnapshotRef       types.String `tfsdk:"snapshot_ref"`
}

func buildInfoAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: description,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
			objectplanmodifier.RequiresReplace(),
		},
		Attributes: map[string]schema.Attribute{
			"dockerfile_content": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Dockerfile content used to build the sandbox or snapshot.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"context_hashes": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Context hashes used for the build.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Build metadata creation timestamp.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Build metadata update timestamp.",
			},
			"snapshot_ref": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Snapshot reference produced by the build.",
			},
		},
	}
}

func buildInfoAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"dockerfile_content": types.StringType,
		"context_hashes":     types.ListType{ElemType: types.StringType},
		"created_at":         types.StringType,
		"updated_at":         types.StringType,
		"snapshot_ref":       types.StringType,
	}
}

func expandCreateBuildInfo(ctx context.Context, value types.Object) (*apiclient.CreateBuildInfo, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	var data buildInfoModel
	diags.Append(value.As(ctx, &data, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	dockerfileContent := optionalString(data.DockerfileContent)
	if dockerfileContent == nil {
		diags.AddAttributeError(
			path.Root("build_info").AtName("dockerfile_content"),
			"Missing Dockerfile content",
			"build_info.dockerfile_content must be set when build_info is configured.",
		)
		return nil, diags
	}

	buildInfo := apiclient.NewCreateBuildInfo(*dockerfileContent)
	contextHashes, listDiags := stringList(ctx, data.ContextHashes)
	diags.Append(listDiags...)
	if diags.HasError() {
		return nil, diags
	}
	if len(contextHashes) > 0 {
		buildInfo.SetContextHashes(contextHashes)
	}

	return buildInfo, diags
}

func flattenBuildInfo(ctx context.Context, buildInfo *apiclient.BuildInfo, prior types.Object) types.Object {
	if buildInfo == nil {
		if prior.IsUnknown() || prior.IsNull() || len(prior.AttributeTypes(ctx)) == 0 {
			return types.ObjectNull(buildInfoAttributeTypes())
		}
		return prior
	}

	priorData := buildInfoModel{}
	if !prior.IsNull() && !prior.IsUnknown() {
		_ = prior.As(ctx, &priorData, basetypes.ObjectAsOptions{})
	}

	dockerfileContent := pointerStringValue(buildInfo.DockerfileContent)
	if dockerfileContent.IsNull() && !priorData.DockerfileContent.IsNull() && !priorData.DockerfileContent.IsUnknown() {
		dockerfileContent = priorData.DockerfileContent
	}

	contextHashes := listStringValue(ctx, buildInfo.ContextHashes)
	if len(buildInfo.ContextHashes) == 0 {
		if !priorData.ContextHashes.IsNull() && !priorData.ContextHashes.IsUnknown() {
			contextHashes = priorData.ContextHashes
		} else {
			contextHashes = types.ListNull(types.StringType)
		}
	}

	createdAt := types.StringNull()
	if !buildInfo.CreatedAt.IsZero() {
		createdAt = types.StringValue(buildInfo.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !buildInfo.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(buildInfo.UpdatedAt.Format(time.RFC3339))
	}

	snapshotRef := types.StringNull()
	if buildInfo.SnapshotRef != "" {
		snapshotRef = types.StringValue(buildInfo.SnapshotRef)
	}

	objectValue, diags := types.ObjectValue(buildInfoAttributeTypes(), map[string]attr.Value{
		"dockerfile_content": dockerfileContent,
		"context_hashes":     contextHashes,
		"created_at":         createdAt,
		"updated_at":         updatedAt,
		"snapshot_ref":       snapshotRef,
	})
	if diags.HasError() {
		return types.ObjectNull(buildInfoAttributeTypes())
	}

	return objectValue
}
