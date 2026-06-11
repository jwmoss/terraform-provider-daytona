// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultAPIURL = "https://app.daytona.io/api"

// Ensure DaytonaProvider satisfies provider.Provider.
var _ provider.Provider = &DaytonaProvider{}

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
				MarkdownDescription: "Daytona API key. May also be set with the `DAYTONA_API_KEY` environment variable.",
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

	if strings.TrimSpace(apiKey) == "" {
		resp.Diagnostics.AddError(
			"Missing Daytona API key",
			"Set the api_key provider attribute or the DAYTONA_API_KEY environment variable.",
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

	client := newDaytonaClient(apiURL, apiKey, organizationID, p.version)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DaytonaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAPIKeyResource,
		NewDockerRegistryResource,
		NewOrganizationResource,
		NewOrganizationInvitationResource,
		NewOrganizationMemberAccessResource,
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
		NewAPIKeysDataSource,
		NewCurrentAPIKeyDataSource,
		NewDockerRegistriesDataSource,
		NewOrganizationInvitationsDataSource,
		NewOrganizationMembersDataSource,
		NewOrganizationRolesDataSource,
		NewOrganizationsDataSource,
		NewRegionsDataSource,
		NewRunnersDataSource,
		NewSandboxesDataSource,
		NewSnapshotsDataSource,
		NewVolumesDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DaytonaProvider{
			version: version,
		}
	}
}
