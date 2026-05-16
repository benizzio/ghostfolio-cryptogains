// Package client implements the minimal Ghostfolio HTTP boundary used by this
// sync-and-storage slice.
// Authored by: OpenCode
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
)

const apiBasePath = "/api/v1"

const defaultHTTPClientTimeout = 30 * time.Second

const defaultActivitiesPageSize = 250

// FailureCategory identifies the single user-visible failure category produced
// by the Ghostfolio boundary in this slice.
//
// Authored by: OpenCode
type FailureCategory string

const (
	// FailureRejectedToken indicates that Ghostfolio rejected the supplied token.
	FailureRejectedToken FailureCategory = "rejected token"

	// FailureTimeout indicates that the request exceeded the allowed wait time.
	FailureTimeout FailureCategory = "timeout"

	// FailureConnectivityProblem indicates a transport-level reachability failure.
	FailureConnectivityProblem FailureCategory = "connectivity problem"

	// FailureUnsuccessfulServerResponse indicates a non-success HTTP response that
	// does not prove contract incompatibility.
	FailureUnsuccessfulServerResponse FailureCategory = "unsuccessful server response"

	// FailureIncompatibleServerContract indicates a reachable server whose
	// behavior does not match this slice's supported contract.
	FailureIncompatibleServerContract FailureCategory = "incompatible server contract"
)

// RequestFailure captures a categorized boundary failure without exposing
// secrets or raw payload details.
//
// Authored by: OpenCode
type RequestFailure struct {
	Category FailureCategory
	Message  string
	Err      error
}

// Error returns the safe error string for the categorized request failure.
//
// Example:
//
//	err := &client.RequestFailure{Category: client.FailureTimeout, Message: "request timed out"}
//	_ = err.Error()
//
// Authored by: OpenCode
func (e *RequestFailure) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Category)
}

// Unwrap returns the underlying cause for the categorized request failure.
//
// Example:
//
//	err := &client.RequestFailure{Err: context.DeadlineExceeded}
//	_ = err.Unwrap()
//
// Authored by: OpenCode
func (e *RequestFailure) Unwrap() error {
	return e.Err
}

// Client executes the Ghostfolio auth and activities-history requests for this
// slice.
//
// Authored by: OpenCode
type Client struct {
	httpClient *http.Client
}

// requestSpec describes one JSON-based Ghostfolio boundary request.
// Authored by: OpenCode
type requestSpec struct {
	Method             string
	Endpoint           string
	Body               io.Reader
	Headers            map[string]string
	BuildErrorMessage  string
	DecodeErrorMessage string
	StatusClassifier   func(*http.Response) error
}

// New creates a Ghostfolio API client backed by the provided HTTP client.
//
// Example:
//
//	transportClient := client.New(http.DefaultClient)
//	_ = transportClient
//
// Authored by: OpenCode
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPClientTimeout}
	}
	return &Client{httpClient: httpClient}
}

// Authenticate executes the anonymous-auth boundary for this slice.
//
// Example:
//
//	response, err := transportClient.Authenticate(ctx, "https://ghostfol.io", "token")
//	if err != nil {
//		panic(err)
//	}
//	_ = response.AuthToken
//
// Authored by: OpenCode
func (c *Client) Authenticate(ctx context.Context, origin string, accessToken string) (dto.AuthResponse, error) {
	var endpoint = strings.TrimRight(origin, "/") + apiBasePath + "/auth/anonymous"
	var requestBody = strings.NewReader(fmt.Sprintf("{\"accessToken\":%q}", accessToken))

	return executeJSONRequest[dto.AuthResponse](
		c.httpClient, ctx, requestSpec{
			Method:             http.MethodPost,
			Endpoint:           endpoint,
			Body:               requestBody,
			Headers:            map[string]string{"Content-Type": "application/json"},
			BuildErrorMessage:  "build auth request",
			DecodeErrorMessage: "auth response was not valid JSON",
			StatusClassifier:   classifyAuthStatus,
		},
	)
}

// FetchActivitiesHistory executes the paginated activities retrieval boundary
// for this slice.
//
// Example:
//
//	response, err := transportClient.FetchActivitiesHistory(ctx, "https://ghostfol.io", "jwt")
//	if err != nil {
//		panic(err)
//	}
//	_ = len(response.Activities)
//
// Authored by: OpenCode
func (c *Client) FetchActivitiesHistory(
	ctx context.Context,
	origin string,
	authToken string,
) (dto.ActivityPageResponse, error) {
	var allActivities = []dto.ActivityPageEntry{}
	var expectedCount = -1

	for skip := 0; ; skip = len(allActivities) {
		page, err := c.fetchActivitiesPage(ctx, origin, authToken, skip, defaultActivitiesPageSize)
		if err != nil {
			return dto.ActivityPageResponse{}, err
		}

		if expectedCount == -1 {
			expectedCount = page.Count
		} else if page.Count != expectedCount {
			return dto.ActivityPageResponse{}, incompatibleServerFailure("activities pagination count changed during retrieval", nil)
		}

		if expectedCount == 0 {
			if len(page.Activities) != 0 {
				return dto.ActivityPageResponse{}, incompatibleServerFailure("activities must be empty when count is zero", nil)
			}
			return dto.ActivityPageResponse{Activities: []dto.ActivityPageEntry{}, Count: 0}, nil
		}

		if len(page.Activities) == 0 {
			return dto.ActivityPageResponse{}, incompatibleServerFailure("activities pagination ended before the reported count was retrieved", nil)
		}

		allActivities = append(allActivities, page.Activities...)
		if len(allActivities) > expectedCount {
			return dto.ActivityPageResponse{}, incompatibleServerFailure("activities pagination exceeded the reported count", nil)
		}
		if len(allActivities) >= expectedCount {
			return dto.ActivityPageResponse{Activities: allActivities, Count: expectedCount}, nil
		}
	}
}

// fetchActivitiesPage executes one paginated activities request.
// Authored by: OpenCode
func (c *Client) fetchActivitiesPage(
	ctx context.Context,
	origin string,
	authToken string,
	skip int,
	take int,
) (dto.ActivityPageResponse, error) {
	var endpoint, err = activitiesEndpoint(origin, skip, take)
	if err != nil {
		return dto.ActivityPageResponse{}, err
	}

	return executeJSONRequest[dto.ActivityPageResponse](
		c.httpClient, ctx, requestSpec{
			Method:             http.MethodGet,
			Endpoint:           endpoint,
			Headers:            map[string]string{"Authorization": "Bearer " + authToken},
			BuildErrorMessage:  "build activities request",
			DecodeErrorMessage: "activities response was not valid JSON",
			StatusClassifier:   classifyActivitiesStatus,
		},
	)
}

// activitiesEndpoint builds the Ghostfolio paginated activities URL.
// Authored by: OpenCode
func activitiesEndpoint(origin string, skip int, take int) (string, error) {
	var endpoint, err = url.Parse(strings.TrimRight(origin, "/") + apiBasePath + "/activities")
	if err != nil {
		return "", fmt.Errorf("build activities request: %w", err)
	}

	var query = endpoint.Query()
	query.Set("skip", fmt.Sprintf("%d", skip))
	query.Set("take", fmt.Sprintf("%d", take))
	query.Set("sortColumn", "date")
	query.Set("sortDirection", "asc")
	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}

// executeJSONRequest runs one JSON-based Ghostfolio boundary request through the
// shared request pipeline.
// Authored by: OpenCode
func executeJSONRequest[T any](httpClient *http.Client, ctx context.Context, spec requestSpec) (T, error) {
	var zero T
	var request, err = http.NewRequestWithContext(ctx, spec.Method, spec.Endpoint, spec.Body)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", spec.BuildErrorMessage, err)
	}

	applyHeaders(request, spec.Headers)

	var response *http.Response
	response, err = httpClient.Do(request)
	if err != nil {
		return zero, classifyTransportFailure(err)
	}
	defer closeBody(response)

	if spec.StatusClassifier != nil {
		err = spec.StatusClassifier(response)
		if err != nil {
			return zero, err
		}
	}

	err = requireJSONContentType(response.Header.Get("Content-Type"))
	if err != nil {
		return zero, incompatibleServerFailure(err.Error(), err)
	}

	var payload T
	var decoder = json.NewDecoder(response.Body)
	decoder.UseNumber()
	err = decoder.Decode(&payload)
	if err != nil {
		return zero, incompatibleServerFailure(spec.DecodeErrorMessage, err)
	}

	return payload, nil
}

// applyHeaders attaches the configured request headers to the outgoing HTTP request.
// Authored by: OpenCode
func applyHeaders(request *http.Request, headers map[string]string) {
	for headerName, headerValue := range headers {
		request.Header.Set(headerName, headerValue)
	}
}

// classifyAuthStatus maps auth response codes to the supported failure taxonomy.
// Authored by: OpenCode
func classifyAuthStatus(response *http.Response) error {
	if response.StatusCode == http.StatusForbidden {
		return &RequestFailure{Category: FailureRejectedToken, Message: "the supplied Ghostfolio token was rejected"}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return unsuccessfulResponseFailure("auth request", response.StatusCode)
	}
	return nil
}

// classifyActivitiesStatus maps activities response codes to the supported failure taxonomy.
// Authored by: OpenCode
func classifyActivitiesStatus(response *http.Response) error {
	switch response.StatusCode {
	case http.StatusBadRequest:
		return &RequestFailure{
			Category: FailureIncompatibleServerContract,
			Message:  "activities request did not match the supported server contract",
		}
	case http.StatusUnauthorized, http.StatusForbidden:
		return unsuccessfulResponseFailure("activities request", response.StatusCode)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return unsuccessfulResponseFailure("activities request", response.StatusCode)
	}
	return nil
}

// unsuccessfulResponseFailure builds a categorized non-success HTTP response error.
// Authored by: OpenCode
func unsuccessfulResponseFailure(requestName string, statusCode int) error {
	return &RequestFailure{
		Category: FailureUnsuccessfulServerResponse,
		Message:  fmt.Sprintf("%s returned HTTP %d", requestName, statusCode),
	}
}

// incompatibleServerFailure builds a categorized incompatible-contract error.
// Authored by: OpenCode
func incompatibleServerFailure(message string, err error) error {
	return &RequestFailure{Category: FailureIncompatibleServerContract, Message: message, Err: err}
}

// classifyTransportFailure maps a transport-level error to a supported user-visible category.
// Authored by: OpenCode
func classifyTransportFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &RequestFailure{Category: FailureTimeout, Message: "request timed out", Err: err}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &RequestFailure{Category: FailureTimeout, Message: "request timed out", Err: err}
	}

	return &RequestFailure{
		Category: FailureConnectivityProblem,
		Message:  "could not reach the selected Ghostfolio server",
		Err:      err,
	}
}

// requireJSONContentType validates that a response declares JSON content.
// Authored by: OpenCode
func requireJSONContentType(contentType string) error {
	var mediaType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("response content type was not usable JSON")
	}
	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		return nil
	}
	return fmt.Errorf("response content type was not usable JSON")
}

// closeBody closes a response body.
// Authored by: OpenCode
func closeBody(response *http.Response) {
	_ = response.Body.Close()
}
