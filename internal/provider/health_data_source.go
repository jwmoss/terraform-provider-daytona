package provider

import (
	"context"
	"errors"
	"net/http"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &HealthDataSource{}

func NewHealthDataSource() datasource.DataSource {
	return &HealthDataSource{}
}

type HealthDataSource struct {
	client *daytonaClient
}

type healthDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Live            types.Bool   `tfsdk:"live"`
	Ready           types.Bool   `tfsdk:"ready"`
	Status          types.String `tfsdk:"status"`
	LiveHTTPStatus  types.Int64  `tfsdk:"live_http_status"`
	ReadyHTTPStatus types.Int64  `tfsdk:"ready_http_status"`
	ResponseJSON    types.String `tfsdk:"response_json"`
	InfoJSON        types.String `tfsdk:"info_json"`
	ErrorJSON       types.String `tfsdk:"error_json"`
	DetailsJSON     types.String `tfsdk:"details_json"`
}

func (d *HealthDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *HealthDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Daytona service liveness and readiness status.",
		Attributes: map[string]schema.Attribute{
			"id":                computedDataSourceStringAttribute("Data source identifier."),
			"live":              computedDataSourceBoolAttribute("Whether the Daytona liveness endpoint returned a successful HTTP status."),
			"ready":             computedDataSourceBoolAttribute("Whether the Daytona readiness endpoint returned a successful HTTP status."),
			"status":            computedDataSourceStringAttribute("Readiness status reported by Daytona, when available."),
			"live_http_status":  computedDataSourceInt64Attribute("HTTP status code returned by the liveness endpoint."),
			"ready_http_status": computedDataSourceInt64Attribute("HTTP status code returned by the readiness endpoint."),
			"response_json":     computedDataSourceStringAttribute("Full readiness response as a JSON object string."),
			"info_json":         computedDataSourceStringAttribute("Readiness info checks as a JSON object string."),
			"error_json":        computedDataSourceStringAttribute("Readiness error checks as a JSON object string."),
			"details_json":      computedDataSourceStringAttribute("Readiness detail checks as a JSON object string."),
		},
	}
}

func (d *HealthDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *HealthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data healthDataSourceModel

	liveResp, err := d.client.api.HealthAPI.HealthControllerLive(ctx).Execute()
	data.LiveHTTPStatus = httpStatusValue(liveResp)
	if err != nil {
		if liveResp == nil {
			addAPIError(&resp.Diagnostics, "Unable to read Daytona liveness", "read health liveness", liveResp, err)
			return
		}
		data.Live = types.BoolValue(false)
	} else {
		data.Live = types.BoolValue(successHTTPStatus(liveResp.StatusCode))
	}

	ready, readyResp, err := d.client.api.HealthAPI.HealthControllerCheck(ctx).Execute()
	data.ReadyHTTPStatus = httpStatusValue(readyResp)
	if err != nil {
		if !setUnhealthyReadiness(&data, readyResp, err) {
			addAPIError(&resp.Diagnostics, "Unable to read Daytona readiness", "read health readiness", readyResp, err)
			return
		}
	} else {
		if ready == nil {
			resp.Diagnostics.AddError("Empty Daytona readiness response", "Daytona returned a successful readiness response without a body.")
			return
		}
		setHealthyReadiness(&data, ready, readyResp)
	}

	data.ID = types.StringValue("health")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setHealthyReadiness(data *healthDataSourceModel, ready *apiclient.HealthControllerCheck200Response, httpResp *http.Response) {
	data.Ready = types.BoolValue(httpResp != nil && successHTTPStatus(httpResp.StatusCode))
	data.Status = pointerStringValue(ready.Status)
	data.ResponseJSON = jsonStringValue(ready)
	data.InfoJSON = jsonStringValue(ready.Info)
	data.ErrorJSON = jsonStringValue(ready.Error)
	data.DetailsJSON = healthDetailsJSON(ready.Details)
}

func setUnhealthyReadiness(data *healthDataSourceModel, httpResp *http.Response, err error) bool {
	if httpResp == nil || httpResp.StatusCode != http.StatusServiceUnavailable {
		return false
	}

	data.Ready = types.BoolValue(false)

	var apiErr *apiclient.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return true
	}

	switch model := apiErr.Model().(type) {
	case apiclient.HealthControllerCheck503Response:
		data.Status = pointerStringValue(model.Status)
		data.ResponseJSON = jsonStringValue(model)
		data.InfoJSON = jsonStringValue(model.Info)
		data.ErrorJSON = jsonStringValue(model.Error)
		data.DetailsJSON = healthDetailsJSON(model.Details)
	case *apiclient.HealthControllerCheck503Response:
		data.Status = pointerStringValue(model.Status)
		data.ResponseJSON = jsonStringValue(model)
		data.InfoJSON = jsonStringValue(model.Info)
		data.ErrorJSON = jsonStringValue(model.Error)
		data.DetailsJSON = healthDetailsJSON(model.Details)
	default:
		if body := apiErr.Body(); len(body) > 0 {
			data.ResponseJSON = types.StringValue(string(body))
		}
	}

	return true
}

func healthDetailsJSON(details *map[string]apiclient.HealthControllerCheck200ResponseInfoValue) types.String {
	if details == nil {
		return types.StringNull()
	}
	return jsonStringValue(*details)
}

func httpStatusValue(resp *http.Response) types.Int64 {
	if resp == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(resp.StatusCode))
}

func successHTTPStatus(status int) bool {
	return status >= 200 && status < 300
}
