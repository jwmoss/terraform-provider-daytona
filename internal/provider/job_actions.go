package provider

import (
	"context"
	"fmt"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ action.Action = &JobStatusUpdateAction{}
var _ action.ActionWithConfigure = &JobStatusUpdateAction{}

func NewJobStatusUpdateAction() action.Action {
	return &JobStatusUpdateAction{}
}

type JobStatusUpdateAction struct {
	client *daytonaClient
}

type jobStatusUpdateActionModel struct {
	JobID          types.String `tfsdk:"job_id"`
	Status         types.String `tfsdk:"status"`
	ErrorMessage   types.String `tfsdk:"error_message"`
	ResultMetadata types.String `tfsdk:"result_metadata"`
}

func (a *JobStatusUpdateAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_update_job_status"
}

func (a *JobStatusUpdateAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Updates a Daytona background job status. This action is intended for runner/job-worker integrations.",
		Attributes: map[string]actionschema.Attribute{
			"job_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Daytona job ID.",
			},
			"status": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: fmt.Sprintf("New Daytona job status. Supported values are: %s.", strings.Join(jobStatusValues(), ", ")),
			},
			"error_message": actionschema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "Optional error message when marking a job failed.",
			},
			"result_metadata": actionschema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				MarkdownDescription: "Optional result metadata string to attach to the job status update.",
			},
		},
	}
}

func (a *JobStatusUpdateAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	a.client = configureActionDaytonaClient(req.ProviderData, &resp.Diagnostics)
}

func (a *JobStatusUpdateAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data jobStatusUpdateActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !ensureActionClient(a.client, &resp.Diagnostics) {
		return
	}

	jobID := strings.TrimSpace(data.JobID.ValueString())
	if jobID == "" {
		resp.Diagnostics.AddAttributeError(path.Root("job_id"), "Missing Daytona job ID", "Configure job_id with the Daytona job ID to update.")
		return
	}

	status := apiclient.JobStatus(strings.TrimSpace(data.Status.ValueString()))
	if !status.IsValid() || status == apiclient.JOBSTATUS_UNKNOWN_DEFAULT_OPEN_API {
		resp.Diagnostics.AddAttributeError(
			path.Root("status"),
			"Invalid Daytona job status",
			fmt.Sprintf("Status must be one of %s.", strings.Join(jobStatusValues(), ", ")),
		)
		return
	}

	payload := *apiclient.NewUpdateJobStatus(status)
	if errorMessage := optionalString(data.ErrorMessage); errorMessage != nil {
		payload.SetErrorMessage(*errorMessage)
	}
	if resultMetadata := optionalString(data.ResultMetadata); resultMetadata != nil {
		payload.SetResultMetadata(*resultMetadata)
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: "Updating Daytona job status."})
	}

	_, httpResp, err := a.client.api.JobsAPI.UpdateJobStatus(ctx, jobID).
		UpdateJobStatus(payload).
		Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to update Daytona job status", "update job status", httpResp, err)
	}
}
