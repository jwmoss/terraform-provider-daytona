package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultAPIURL = "https://app.daytona.io/api"

// Ensure DaytonaProvider satisfies provider.Provider.
var _ provider.Provider = &DaytonaProvider{}
var _ provider.ProviderWithActions = &DaytonaProvider{}

// DaytonaProvider defines the provider implementation.
type DaytonaProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// DaytonaProviderModel describes the provider configuration.
type DaytonaProviderModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	AccessToken    types.String `tfsdk:"access_token"`
	APIURL         types.String `tfsdk:"api_url"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func (p *DaytonaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "daytona"
	resp.Version = p.version
}

func (p *DaytonaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing Daytona sandboxes and supporting infrastructure.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Daytona API key. May also be set with the `DAYTONA_API_KEY` environment variable. Daytona API keys only work with API-key-enabled routes; use `access_token` for JWT-only Daytona provisioning routes.",
				Optional:            true,
				Sensitive:           true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Daytona OAuth access token for JWT-only Daytona API routes. May also be set with the `DAYTONA_ACCESS_TOKEN` environment variable. When configured, this token takes precedence over `api_key` for the provider's bearer token.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Daytona API base URL. May also be set with `DAYTONA_API_URL`. Defaults to `%s`.", defaultAPIURL),
				Optional:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "Optional Daytona organization ID to send as `X-Daytona-Organization-ID`. May also be set with `DAYTONA_ORGANIZATION_ID`.",
				Optional:            true,
			},
		},
	}
}

func (p *DaytonaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data DaytonaProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("DAYTONA_API_KEY")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	accessToken := os.Getenv("DAYTONA_ACCESS_TOKEN")
	if !data.AccessToken.IsNull() {
		accessToken = data.AccessToken.ValueString()
	}

	authToken := strings.TrimSpace(apiKey)
	if strings.TrimSpace(accessToken) != "" {
		authToken = strings.TrimSpace(accessToken)
		if data.AccessToken.IsNull() && !data.APIKey.IsNull() && strings.TrimSpace(apiKey) != "" {
			resp.Diagnostics.AddWarning(
				"DAYTONA_ACCESS_TOKEN overrides configured api_key",
				"The provider is authenticating with the access token from the DAYTONA_ACCESS_TOKEN environment variable even though api_key is set in the provider configuration, because access tokens take precedence. Unset DAYTONA_ACCESS_TOKEN to authenticate with the configured api_key.",
			)
		}
	}

	if authToken == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona authentication token",
			"Set access_token or DAYTONA_ACCESS_TOKEN for JWT-only Daytona API routes, or set api_key or DAYTONA_API_KEY for API-key-enabled routes.",
		)
		return
	}

	apiURL := os.Getenv("DAYTONA_API_URL")
	if !data.APIURL.IsNull() {
		apiURL = data.APIURL.ValueString()
	}
	if strings.TrimSpace(apiURL) == "" {
		apiURL = defaultAPIURL
	}

	organizationID := os.Getenv("DAYTONA_ORGANIZATION_ID")
	if !data.OrganizationID.IsNull() {
		organizationID = data.OrganizationID.ValueString()
	}

	client := newDaytonaClient(apiURL, authToken, organizationID, p.version)
	resp.ActionData = client
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DaytonaProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{
		NewAdminCreateUserAction,
		NewAdminInitializeWebhooksAction,
		NewAdminRegenerateUserKeyPairAction,
		NewAdminRecoverSandboxAction,
		NewAdminSendWebhookAction,
		NewAdminSetDefaultDockerRegistryAction,
		NewAdminSetSnapshotGeneralStatusAction,
		NewAPIKeyForUserRevokeAction,
		NewOrganizationInvitationAcceptAction,
		NewOrganizationInvitationDeclineAction,
		NewOrganizationLeaveAction,
		NewOrganizationSuspendAction,
		NewOrganizationUnsuspendAction,
		NewUserLinkAccountAction,
		NewUserUnlinkAccountAction,
		NewUserSmsMFAEnrollmentAction,
		NewSandboxCreateBackupAction,
		NewSandboxCreateSnapshotAction,
		NewSandboxArchiveAction,
		NewSandboxExpireSignedPortPreviewURLAction,
		NewSandboxForkAction,
		NewSandboxRecoverAction,
		NewSandboxRevokeSSHAccessAction,
		NewSandboxStartAction,
		NewSandboxStopAction,
		NewSandboxUpdateLastActivityAction,
		NewSnapshotActivateAction,
		NewSnapshotDeactivateAction,
		NewWebhookInitializeAction,
		NewWebhookRefreshEndpointsAction,
	}
}

func (p *DaytonaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAdminOrganizationRegionQuotaResource,
		NewAdminRunnerResource,
		NewAPIKeyResource,
		NewDockerRegistryResource,
		NewOrganizationResource,
		NewOrganizationInvitationResource,
		NewOrganizationMemberAccessResource,
		NewOrganizationOtelConfigResource,
		NewOrganizationRegionQuotaResource,
		NewOrganizationRoleResource,
		NewRegionResource,
		NewRunnerResource,
		NewSandboxResource,
		NewSnapshotResource,
		NewVolumeResource,
	}
}

func (p *DaytonaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccountProvidersDataSource,
		NewAdminAuditLogsDataSource,
		NewAdminRunnerDataSource,
		NewAdminRunnersDataSource,
		NewAdminSnapshotImageCleanupStatusDataSource,
		NewAdminUserDataSource,
		NewAdminUsersDataSource,
		NewAdminWebhookMessageAttemptsDataSource,
		NewAdminWebhookStatusDataSource,
		NewAPIKeyDataSource,
		NewAPIKeysDataSource,
		NewAuthenticatedRunnerSandboxesDataSource,
		NewConfigDataSource,
		NewCurrentAPIKeyDataSource,
		NewCurrentUserOrganizationInvitationsDataSource,
		NewCurrentUserDataSource,
		NewDockerRegistriesDataSource,
		NewDockerRegistryDataSource,
		NewDockerRegistryPushAccessDataSource,
		NewHealthDataSource,
		NewJobDataSource,
		NewJobsDataSource,
		NewObjectStoragePushAccessDataSource,
		NewOrganizationInvitationDataSource,
		NewOrganizationAuditLogsDataSource,
		NewOrganizationInvitationsDataSource,
		NewOrganizationMemberDataSource,
		NewOrganizationMembersDataSource,
		NewOrganizationOtelConfigDataSource,
		NewOrganizationOtelConfigBySandboxAuthTokenDataSource,
		NewOrganizationRoleDataSource,
		NewOrganizationRolesDataSource,
		NewOrganizationUsageDataSource,
		NewOrganizationDataSource,
		NewOrganizationsDataSource,
		NewRegionDataSource,
		NewRegionsDataSource,
		NewRunnerDataSource,
		NewRunnerForSandboxDataSource,
		NewRunnerFullDataSource,
		NewRunnersDataSource,
		NewRunnersBySnapshotRefDataSource,
		NewSandboxAccessDataSource,
		NewSandboxAncestorsDataSource,
		NewSandboxAuthTokenValidationDataSource,
		NewSandboxBuildLogsURLDataSource,
		NewSandboxForksDataSource,
		NewSandboxIDFromSignedPreviewTokenDataSource,
		NewSandboxLogsDataSource,
		NewSandboxMetricsDataSource,
		NewSandboxOrganizationDataSource,
		NewSandboxParentDataSource,
		NewSandboxPortPreviewURLDataSource,
		NewSandboxPublicStatusDataSource,
		NewSandboxQueryDataSource,
		NewSandboxRegionQuotaDataSource,
		NewSandboxSignedPortPreviewURLDataSource,
		NewSandboxSSHAccessDataSource,
		NewSandboxSSHAccessValidationDataSource,
		NewSandboxDataSource,
		NewSandboxesDataSource,
		NewSharedRegionsDataSource,
		NewSandboxTraceSpansDataSource,
		NewSandboxTracesDataSource,
		NewSandboxToolboxProxyURLDataSource,
		NewSnapshotBuildLogsURLDataSource,
		NewSnapshotDataSource,
		NewSnapshotsDataSource,
		NewVolumeByNameDataSource,
		NewVolumeDataSource,
		NewVolumesDataSource,
		NewWebhookAppPortalAccessDataSource,
		NewWebhookInitializationStatusDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DaytonaProvider{
			version: version,
		}
	}
}
