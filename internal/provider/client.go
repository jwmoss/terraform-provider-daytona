// Copyright (c) Jonathan Moss.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"strings"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const organizationHeader = "X-Daytona-Organization-ID"

type daytonaClient struct {
	api            *apiclient.APIClient
	apiURL         string
	organizationID string
}

func newDaytonaClient(apiURL, apiKey, organizationID, version string) *daytonaClient {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")

	cfg := apiclient.NewConfiguration()
	cfg.Servers = apiclient.ServerConfigurations{
		{
			URL:         apiURL,
			Description: "Configured Daytona API endpoint",
		},
	}
	cfg.UserAgent = fmt.Sprintf("terraform-provider-daytona/%s", version)
	cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)

	if strings.TrimSpace(organizationID) != "" {
		cfg.AddDefaultHeader(organizationHeader, strings.TrimSpace(organizationID))
	}

	return &daytonaClient{
		api:            apiclient.NewAPIClient(cfg),
		apiURL:         apiURL,
		organizationID: strings.TrimSpace(organizationID),
	}
}

func isNotFound(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}

func addAPIError(diags *diag.Diagnostics, summary, operation string, resp *http.Response, err error) {
	if err == nil {
		return
	}

	detail := fmt.Sprintf("Daytona %s failed: %s", operation, err)
	if resp != nil {
		detail = fmt.Sprintf("%s (HTTP %d)", detail, resp.StatusCode)
	}

	if bodyProvider, ok := err.(interface{ Body() []byte }); ok {
		body := strings.TrimSpace(string(bodyProvider.Body()))
		if body != "" {
			if len(body) > 1000 {
				body = body[:1000] + "..."
			}
			detail = detail + "\n\nResponse body: " + body
		}
	}

	diags.AddError(summary, detail)
}
