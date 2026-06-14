package provider

import (
	"context"
	"fmt"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SandboxLogsDataSource{}
var _ datasource.DataSource = &SandboxTracesDataSource{}
var _ datasource.DataSource = &SandboxTraceSpansDataSource{}
var _ datasource.DataSource = &SandboxMetricsDataSource{}

func NewSandboxLogsDataSource() datasource.DataSource {
	return &SandboxLogsDataSource{}
}

func NewSandboxTracesDataSource() datasource.DataSource {
	return &SandboxTracesDataSource{}
}

func NewSandboxTraceSpansDataSource() datasource.DataSource {
	return &SandboxTraceSpansDataSource{}
}

func NewSandboxMetricsDataSource() datasource.DataSource {
	return &SandboxMetricsDataSource{}
}

type SandboxLogsDataSource struct {
	client *daytonaClient
}

type sandboxLogsDataSourceModel struct {
	ID                    types.String           `tfsdk:"id"`
	SandboxID             types.String           `tfsdk:"sandbox_id"`
	RequestOrganizationID types.String           `tfsdk:"request_organization_id"`
	From                  types.String           `tfsdk:"from"`
	To                    types.String           `tfsdk:"to"`
	Page                  types.Int64            `tfsdk:"page"`
	Limit                 types.Int64            `tfsdk:"limit"`
	Severities            types.List             `tfsdk:"severities"`
	Search                types.String           `tfsdk:"search"`
	Total                 types.Int64            `tfsdk:"total"`
	TotalPages            types.Int64            `tfsdk:"total_pages"`
	Items                 []sandboxLogEntryModel `tfsdk:"items"`
}

type sandboxLogEntryModel struct {
	Timestamp          types.String  `tfsdk:"timestamp"`
	Body               types.String  `tfsdk:"body"`
	SeverityText       types.String  `tfsdk:"severity_text"`
	SeverityNumber     types.Float64 `tfsdk:"severity_number"`
	ServiceName        types.String  `tfsdk:"service_name"`
	ResourceAttributes types.Map     `tfsdk:"resource_attributes"`
	LogAttributes      types.Map     `tfsdk:"log_attributes"`
	TraceID            types.String  `tfsdk:"trace_id"`
	SpanID             types.String  `tfsdk:"span_id"`
}

func (d *SandboxLogsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_logs"
}

func (d *SandboxLogsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads bounded OpenTelemetry logs for a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id":              requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"from":                    requiredDataSourceStringAttribute("Start of the time range as an RFC3339 timestamp."),
			"to":                      requiredDataSourceStringAttribute("End of the time range as an RFC3339 timestamp."),
			"page":                    optionalComputedInt64DataSourceAttribute("Page number to request. Defaults to 1."),
			"limit":                   optionalComputedInt64DataSourceAttribute("Maximum number of log entries to return. Defaults to 100."),
			"severities":              optionalDataSourceStringListAttribute("Optional severity filters such as `DEBUG`, `INFO`, `WARN`, or `ERROR`."),
			"search": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional search text for log bodies.",
			},
			"total":       computedDataSourceInt64Attribute("Total matching log entries."),
			"total_pages": computedDataSourceInt64Attribute("Total result pages."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned log entries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sandboxLogEntryAttributes(),
				},
			},
		},
	}
}

func (d *SandboxLogsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxLogsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	from, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "from", data.From.ValueString())
	if !ok {
		return
	}
	to, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "to", data.To.ValueString())
	if !ok {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxLogs(ctx, data.SandboxID.ValueString()).From(from).To(to)
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}
	if dataSourceInt64Configured(data.Page) {
		request = request.Page(float32(data.Page.ValueInt64()))
	} else {
		data.Page = types.Int64Value(1)
	}
	if dataSourceInt64Configured(data.Limit) {
		request = request.Limit(float32(data.Limit.ValueInt64()))
	} else {
		data.Limit = types.Int64Value(100)
	}
	if !data.Severities.IsNull() && !data.Severities.IsUnknown() {
		severities, diags := stringList(ctx, data.Severities)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(severities) > 0 {
			request = request.Severities(severities)
		}
	}
	if search := optionalString(data.Search); search != nil {
		request = request.Search(*search)
	}

	logs, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox logs", "read sandbox logs", httpResp, err)
		return
	}
	if logs == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox logs response", fmt.Sprintf("Daytona returned a successful response without logs for sandbox %q.", data.SandboxID.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:logs:%s:%s:%d:%d", data.SandboxID.ValueString(), data.From.ValueString(), data.To.ValueString(), data.Page.ValueInt64(), data.Limit.ValueInt64()))
	data.Total = types.Int64Value(int64(logs.Total))
	data.TotalPages = types.Int64Value(int64(logs.TotalPages))
	data.Items = flattenSandboxLogEntries(ctx, logs.Items)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxTracesDataSource struct {
	client *daytonaClient
}

type sandboxTracesDataSourceModel struct {
	ID                    types.String               `tfsdk:"id"`
	SandboxID             types.String               `tfsdk:"sandbox_id"`
	RequestOrganizationID types.String               `tfsdk:"request_organization_id"`
	From                  types.String               `tfsdk:"from"`
	To                    types.String               `tfsdk:"to"`
	Page                  types.Int64                `tfsdk:"page"`
	Limit                 types.Int64                `tfsdk:"limit"`
	Total                 types.Int64                `tfsdk:"total"`
	TotalPages            types.Int64                `tfsdk:"total_pages"`
	Items                 []sandboxTraceSummaryModel `tfsdk:"items"`
}

type sandboxTraceSummaryModel struct {
	TraceID      types.String  `tfsdk:"trace_id"`
	RootSpanName types.String  `tfsdk:"root_span_name"`
	StartTime    types.String  `tfsdk:"start_time"`
	EndTime      types.String  `tfsdk:"end_time"`
	DurationMs   types.Float64 `tfsdk:"duration_ms"`
	SpanCount    types.Float64 `tfsdk:"span_count"`
	StatusCode   types.String  `tfsdk:"status_code"`
}

func (d *SandboxTracesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_traces"
}

func (d *SandboxTracesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads bounded OpenTelemetry trace summaries for a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id":              requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"from":                    requiredDataSourceStringAttribute("Start of the time range as an RFC3339 timestamp."),
			"to":                      requiredDataSourceStringAttribute("End of the time range as an RFC3339 timestamp."),
			"page":                    optionalComputedInt64DataSourceAttribute("Page number to request. Defaults to 1."),
			"limit":                   optionalComputedInt64DataSourceAttribute("Maximum number of trace summaries to return. Defaults to 100."),
			"total":                   computedDataSourceInt64Attribute("Total matching traces."),
			"total_pages":             computedDataSourceInt64Attribute("Total result pages."),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned trace summaries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sandboxTraceSummaryAttributes(),
				},
			},
		},
	}
}

func (d *SandboxTracesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxTracesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxTracesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	from, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "from", data.From.ValueString())
	if !ok {
		return
	}
	to, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "to", data.To.ValueString())
	if !ok {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxTraces(ctx, data.SandboxID.ValueString()).From(from).To(to)
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}
	if dataSourceInt64Configured(data.Page) {
		request = request.Page(float32(data.Page.ValueInt64()))
	} else {
		data.Page = types.Int64Value(1)
	}
	if dataSourceInt64Configured(data.Limit) {
		request = request.Limit(float32(data.Limit.ValueInt64()))
	} else {
		data.Limit = types.Int64Value(100)
	}

	traces, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox traces", "read sandbox traces", httpResp, err)
		return
	}
	if traces == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox traces response", fmt.Sprintf("Daytona returned a successful response without traces for sandbox %q.", data.SandboxID.ValueString()))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:traces:%s:%s:%d:%d", data.SandboxID.ValueString(), data.From.ValueString(), data.To.ValueString(), data.Page.ValueInt64(), data.Limit.ValueInt64()))
	data.Total = types.Int64Value(int64(traces.Total))
	data.TotalPages = types.Int64Value(int64(traces.TotalPages))
	data.Items = flattenSandboxTraceSummaries(traces.Items)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxTraceSpansDataSource struct {
	client *daytonaClient
}

type sandboxTraceSpansDataSourceModel struct {
	ID                    types.String            `tfsdk:"id"`
	SandboxID             types.String            `tfsdk:"sandbox_id"`
	TraceID               types.String            `tfsdk:"trace_id"`
	RequestOrganizationID types.String            `tfsdk:"request_organization_id"`
	Items                 []sandboxTraceSpanModel `tfsdk:"items"`
}

type sandboxTraceSpanModel struct {
	TraceID        types.String  `tfsdk:"trace_id"`
	SpanID         types.String  `tfsdk:"span_id"`
	ParentSpanID   types.String  `tfsdk:"parent_span_id"`
	SpanName       types.String  `tfsdk:"span_name"`
	Timestamp      types.String  `tfsdk:"timestamp"`
	DurationNs     types.Float64 `tfsdk:"duration_ns"`
	SpanAttributes types.Map     `tfsdk:"span_attributes"`
	StatusCode     types.String  `tfsdk:"status_code"`
	StatusMessage  types.String  `tfsdk:"status_message"`
}

func (d *SandboxTraceSpansDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_trace_spans"
}

func (d *SandboxTraceSpansDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads OpenTelemetry spans for a Daytona sandbox trace.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id":              requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"trace_id":                requiredDataSourceStringAttribute("Trace ID."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"items": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned trace spans.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sandboxTraceSpanAttributes(),
				},
			},
		},
	}
}

func (d *SandboxTraceSpansDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxTraceSpansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxTraceSpansDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxTraceSpans(ctx, data.SandboxID.ValueString(), data.TraceID.ValueString())
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}

	spans, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox trace spans", "read sandbox trace spans", httpResp, err)
		return
	}

	data.ID = types.StringValue(data.SandboxID.ValueString() + ":trace:" + data.TraceID.ValueString() + ":spans")
	data.Items = flattenSandboxTraceSpans(ctx, spans)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type SandboxMetricsDataSource struct {
	client *daytonaClient
}

type sandboxMetricsDataSourceModel struct {
	ID                    types.String               `tfsdk:"id"`
	SandboxID             types.String               `tfsdk:"sandbox_id"`
	RequestOrganizationID types.String               `tfsdk:"request_organization_id"`
	From                  types.String               `tfsdk:"from"`
	To                    types.String               `tfsdk:"to"`
	MetricNames           types.List                 `tfsdk:"metric_names"`
	Series                []sandboxMetricSeriesModel `tfsdk:"series"`
}

type sandboxMetricSeriesModel struct {
	MetricName types.String                  `tfsdk:"metric_name"`
	DataPoints []sandboxMetricDataPointModel `tfsdk:"data_points"`
}

type sandboxMetricDataPointModel struct {
	Timestamp types.String  `tfsdk:"timestamp"`
	Value     types.Float64 `tfsdk:"value"`
}

func (d *SandboxMetricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sandbox_metrics"
}

func (d *SandboxMetricsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads bounded OpenTelemetry metrics for a Daytona sandbox.",
		Attributes: map[string]schema.Attribute{
			"id":                      computedDataSourceStringAttribute("Data source identifier."),
			"sandbox_id":              requiredDataSourceStringAttribute("Daytona sandbox ID."),
			"request_organization_id": optionalOrganizationIDDataSourceStringAttribute(),
			"from":                    requiredDataSourceStringAttribute("Start of the time range as an RFC3339 timestamp."),
			"to":                      requiredDataSourceStringAttribute("End of the time range as an RFC3339 timestamp."),
			"metric_names":            optionalDataSourceStringListAttribute("Optional metric names to return."),
			"series": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Returned metric series.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sandboxMetricSeriesAttributes(),
				},
			},
		},
	}
}

func (d *SandboxMetricsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client := configureDataSourceClient(req.ProviderData, &resp.Diagnostics)
	if client == nil {
		return
	}
	d.client = client
}

func (d *SandboxMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sandboxMetricsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	from, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "from", data.From.ValueString())
	if !ok {
		return
	}
	to, ok := parseRFC3339DataSourceTime(&resp.Diagnostics, "to", data.To.ValueString())
	if !ok {
		return
	}

	request := d.client.api.SandboxAPI.GetSandboxMetrics(ctx, data.SandboxID.ValueString()).From(from).To(to)
	if organizationID := optionalString(data.RequestOrganizationID); organizationID != nil {
		request = request.XDaytonaOrganizationID(*organizationID)
	}
	if !data.MetricNames.IsNull() && !data.MetricNames.IsUnknown() {
		metricNames, diags := stringList(ctx, data.MetricNames)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(metricNames) > 0 {
			request = request.MetricNames(metricNames)
		}
	}

	metrics, httpResp, err := request.Execute()
	if err != nil {
		addAPIError(&resp.Diagnostics, "Unable to read Daytona sandbox metrics", "read sandbox metrics", httpResp, err)
		return
	}
	if metrics == nil {
		resp.Diagnostics.AddError("Empty Daytona sandbox metrics response", fmt.Sprintf("Daytona returned a successful response without metrics for sandbox %q.", data.SandboxID.ValueString()))
		return
	}

	data.ID = types.StringValue(data.SandboxID.ValueString() + ":metrics:" + data.From.ValueString() + ":" + data.To.ValueString())
	data.Series = flattenSandboxMetricSeries(metrics.Series)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func sandboxLogEntryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"timestamp":           computedDataSourceStringAttribute("Log timestamp."),
		"body":                sensitiveComputedDataSourceStringAttribute("Log message body."),
		"severity_text":       computedDataSourceStringAttribute("Severity level text."),
		"severity_number":     computedDataSourceFloat64Attribute("Severity level number, when available."),
		"service_name":        computedDataSourceStringAttribute("Service name that generated the log."),
		"resource_attributes": sensitiveComputedDataSourceStringMapAttribute("OpenTelemetry resource attributes."),
		"log_attributes":      sensitiveComputedDataSourceStringMapAttribute("Log-specific OpenTelemetry attributes."),
		"trace_id":            computedDataSourceStringAttribute("Associated trace ID, when available."),
		"span_id":             computedDataSourceStringAttribute("Associated span ID, when available."),
	}
}

func sandboxTraceSummaryAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"trace_id":       computedDataSourceStringAttribute("Trace ID."),
		"root_span_name": computedDataSourceStringAttribute("Root span name."),
		"start_time":     computedDataSourceStringAttribute("Trace start timestamp."),
		"end_time":       computedDataSourceStringAttribute("Trace end timestamp."),
		"duration_ms":    computedDataSourceFloat64Attribute("Trace duration in milliseconds."),
		"span_count":     computedDataSourceFloat64Attribute("Number of spans in the trace."),
		"status_code":    computedDataSourceStringAttribute("Trace status code, when available."),
	}
}

func sandboxTraceSpanAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"trace_id":        computedDataSourceStringAttribute("Trace ID."),
		"span_id":         computedDataSourceStringAttribute("Span ID."),
		"parent_span_id":  computedDataSourceStringAttribute("Parent span ID, when available."),
		"span_name":       computedDataSourceStringAttribute("Span name."),
		"timestamp":       computedDataSourceStringAttribute("Span timestamp."),
		"duration_ns":     computedDataSourceFloat64Attribute("Span duration in nanoseconds."),
		"span_attributes": sensitiveComputedDataSourceStringMapAttribute("Span attributes."),
		"status_code":     computedDataSourceStringAttribute("Span status code, when available."),
		"status_message":  computedDataSourceStringAttribute("Span status message, when available."),
	}
}

func sandboxMetricSeriesAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"metric_name": computedDataSourceStringAttribute("Metric name."),
		"data_points": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Metric data points.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"timestamp": computedDataSourceStringAttribute("Metric timestamp."),
					"value":     computedDataSourceFloat64Attribute("Metric value."),
				},
			},
		},
	}
}

func flattenSandboxLogEntries(ctx context.Context, logs []apiclient.LogEntry) []sandboxLogEntryModel {
	items := make([]sandboxLogEntryModel, 0, len(logs))
	for _, log := range logs {
		items = append(items, sandboxLogEntryModel{
			Timestamp:          types.StringValue(log.Timestamp),
			Body:               types.StringValue(log.Body),
			SeverityText:       types.StringValue(log.SeverityText),
			SeverityNumber:     nullableFloat32Pointer(log.SeverityNumber),
			ServiceName:        types.StringValue(log.ServiceName),
			ResourceAttributes: stringMapValue(ctx, log.ResourceAttributes),
			LogAttributes:      stringMapValue(ctx, log.LogAttributes),
			TraceID:            pointerStringValue(log.TraceId),
			SpanID:             pointerStringValue(log.SpanId),
		})
	}
	return items
}

func flattenSandboxTraceSummaries(traces []apiclient.TraceSummary) []sandboxTraceSummaryModel {
	items := make([]sandboxTraceSummaryModel, 0, len(traces))
	for _, trace := range traces {
		items = append(items, sandboxTraceSummaryModel{
			TraceID:      types.StringValue(trace.TraceId),
			RootSpanName: types.StringValue(trace.RootSpanName),
			StartTime:    types.StringValue(trace.StartTime),
			EndTime:      types.StringValue(trace.EndTime),
			DurationMs:   float64Value(trace.DurationMs),
			SpanCount:    float64Value(trace.SpanCount),
			StatusCode:   pointerStringValue(trace.StatusCode),
		})
	}
	return items
}

func flattenSandboxTraceSpans(ctx context.Context, spans []apiclient.TraceSpan) []sandboxTraceSpanModel {
	items := make([]sandboxTraceSpanModel, 0, len(spans))
	for _, span := range spans {
		items = append(items, sandboxTraceSpanModel{
			TraceID:        types.StringValue(span.TraceId),
			SpanID:         types.StringValue(span.SpanId),
			ParentSpanID:   pointerStringValue(span.ParentSpanId),
			SpanName:       types.StringValue(span.SpanName),
			Timestamp:      types.StringValue(span.Timestamp),
			DurationNs:     float64Value(span.DurationNs),
			SpanAttributes: stringMapValue(ctx, span.SpanAttributes),
			StatusCode:     pointerStringValue(span.StatusCode),
			StatusMessage:  pointerStringValue(span.StatusMessage),
		})
	}
	return items
}

func flattenSandboxMetricSeries(series []apiclient.MetricSeries) []sandboxMetricSeriesModel {
	items := make([]sandboxMetricSeriesModel, 0, len(series))
	for _, metricSeries := range series {
		items = append(items, sandboxMetricSeriesModel{
			MetricName: types.StringValue(metricSeries.MetricName),
			DataPoints: flattenSandboxMetricDataPoints(metricSeries.DataPoints),
		})
	}
	return items
}

func flattenSandboxMetricDataPoints(points []apiclient.MetricDataPoint) []sandboxMetricDataPointModel {
	items := make([]sandboxMetricDataPointModel, 0, len(points))
	for _, point := range points {
		items = append(items, sandboxMetricDataPointModel{
			Timestamp: types.StringValue(point.Timestamp),
			Value:     float64Value(point.Value),
		})
	}
	return items
}

func optionalComputedInt64DataSourceAttribute(description string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: description,
	}
}

func optionalDataSourceStringListAttribute(description string) schema.ListAttribute {
	return schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: description,
	}
}

func dataSourceInt64Configured(value types.Int64) bool {
	return !value.IsNull() && !value.IsUnknown()
}
