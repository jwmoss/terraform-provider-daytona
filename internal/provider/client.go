package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const organizationHeader = "X-Daytona-Organization-ID"

// requestTimeout bounds every API request so a stalled connection fails instead
// of hanging terraform apply indefinitely.
const requestTimeout = 5 * time.Minute

// maxRetries bounds how many times a transient Daytona API failure is retried
// before the operation fails the terraform apply.
const maxRetries = 4

// retryWaitMin and retryWaitMax bound the exponential backoff between retries.
// They are package variables so tests can shrink them; production keeps the
// defaults below.
var (
	retryWaitMin = 1 * time.Second
	retryWaitMax = 30 * time.Second
)

// newRetryableHTTPClient returns an *http.Client that retries transient Daytona
// API failures with capped exponential backoff that honors any Retry-After
// header, so a single transient blip does not fail an entire terraform apply.
// Retries are gated by safeRetryPolicy so non-idempotent mutations are never
// replayed. The timeout bounds each individual attempt, not the whole retry
// sequence.
func newRetryableHTTPClient(timeout time.Duration) *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = maxRetries
	retryClient.RetryWaitMin = retryWaitMin
	retryClient.RetryWaitMax = retryWaitMax
	retryClient.Logger = nil
	retryClient.HTTPClient.Timeout = timeout
	retryClient.CheckRetry = safeRetryPolicy

	return retryClient.StandardClient()
}

// safeRetryPolicy retries only failures that are safe to replay, so a lost
// response can never duplicate a create or repeat a non-idempotent mutation:
//   - HTTP 429 is retried for any method, because the server rejects the request
//     before processing it (nothing was mutated).
//   - HTTP 5xx is retried only for idempotent methods, because the server may
//     have already applied the change before the error was returned.
//   - Transport errors carry no response (and so no method here), and the
//     request may have reached the server, so they are not retried.
func safeRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil || resp == nil {
		return false, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}
	if resp.Request != nil && !isIdempotentMethod(resp.Request.Method) {
		return false, nil
	}

	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// isIdempotentMethod reports whether replaying a request with this method cannot
// cause a duplicate or additional side effect.
func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete, http.MethodTrace:
		return true
	default:
		return false
	}
}

// maxErrorBodyBytes bounds how much of a response body is buffered for error
// reporting.
const maxErrorBodyBytes = 1 << 20

type daytonaClient struct {
	api            *apiclient.APIClient
	httpClient     *http.Client
	apiURL         string
	authToken      string
	organizationID string
	userAgent      string
}

func newDaytonaClient(apiURL, authToken, organizationID, version string) *daytonaClient {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")

	cfg := apiclient.NewConfiguration()
	cfg.Servers = apiclient.ServerConfigurations{
		{
			URL:         apiURL,
			Description: "Configured Daytona API endpoint",
		},
	}
	cfg.UserAgent = fmt.Sprintf("terraform-provider-daytona/%s", version)
	cfg.AddDefaultHeader("Authorization", "Bearer "+authToken)

	if strings.TrimSpace(organizationID) != "" {
		cfg.AddDefaultHeader(organizationHeader, strings.TrimSpace(organizationID))
	}

	httpClient := newRetryableHTTPClient(requestTimeout)
	cfg.HTTPClient = httpClient

	return &daytonaClient{
		api:            apiclient.NewAPIClient(cfg),
		httpClient:     httpClient,
		apiURL:         apiURL,
		authToken:      authToken,
		organizationID: strings.TrimSpace(organizationID),
		userAgent:      cfg.UserAgent,
	}
}

func (c *daytonaClient) patchJSON(ctx context.Context, endpoint string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.apiURL+"/"+strings.TrimLeft(endpoint, "/"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.organizationID != "" {
		req.Header.Set(organizationHeader, c.organizationID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil || resp == nil {
		return resp, err
	}

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	if readErr != nil {
		return resp, readErr
	}
	if resp.StatusCode >= 300 {
		return resp, rawAPIError{status: resp.Status, body: respBody}
	}
	return resp, nil
}

type rawAPIError struct {
	status string
	body   []byte
}

func (e rawAPIError) Error() string {
	return e.status
}

func (e rawAPIError) Body() []byte {
	return e.body
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

func addEmptyAPIResponseError(diags *diag.Diagnostics, summary, operation string, resp *http.Response) {
	detail := fmt.Sprintf("Daytona %s returned a successful response without a response body.", operation)
	if resp != nil {
		detail = fmt.Sprintf("%s (HTTP %d)", detail, resp.StatusCode)
	}
	diags.AddError(summary, detail)
}
