// Package client implements the minimal Ghostfolio HTTP boundary used by this
// validation-only slice.
// Authored by: OpenCode
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
)

const apiBasePath = "/api/v1"

// FailureCategory identifies the single user-visible failure category produced
// by the Ghostfolio boundary in this slice.
//
// Example:
//
//	var category client.FailureCategory = client.FailureRejectedToken
//	_ = category
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
// Example:
//
//	err := &client.RequestFailure{Category: client.FailureTimeout, Message: "request timed out"}
//	_ = err.Error()
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

// Client executes the minimal Ghostfolio auth and activities-probe requests for
// this slice.
//
// Example:
//
//	transportClient := client.New(http.DefaultClient)
//	_ = transportClient
//
// Authored by: OpenCode
type Client struct {
	httpClient *http.Client
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
		httpClient = http.DefaultClient
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
	var err error
	var request *http.Request
	request, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, requestBody)
	if err != nil {
		return dto.AuthResponse{}, fmt.Errorf("build auth request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	var response *http.Response
	response, err = c.httpClient.Do(request)
	if err != nil {
		return dto.AuthResponse{}, classifyTransportFailure(err)
	}
	defer closeBody(response)

	if response.StatusCode == http.StatusForbidden {
		return dto.AuthResponse{}, &RequestFailure{Category: FailureRejectedToken, Message: "the supplied Ghostfolio token was rejected"}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return dto.AuthResponse{}, &RequestFailure{Category: FailureUnsuccessfulServerResponse, Message: fmt.Sprintf("auth request returned HTTP %d", response.StatusCode)}
	}
	if err := requireJSONContentType(response.Header.Get("Content-Type")); err != nil {
		return dto.AuthResponse{}, &RequestFailure{Category: FailureIncompatibleServerContract, Message: err.Error(), Err: err}
	}

	var payload dto.AuthResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return dto.AuthResponse{}, &RequestFailure{Category: FailureIncompatibleServerContract, Message: "auth response was not valid JSON", Err: err}
	}

	return payload, nil
}

// FetchActivitiesProbe executes the one-page activities validation boundary for
// this slice.
//
// Example:
//
//	response, err := transportClient.FetchActivitiesProbe(ctx, "https://ghostfol.io", "jwt")
//	if err != nil {
//		panic(err)
//	}
//	_ = response.Count
//
// Authored by: OpenCode
func (c *Client) FetchActivitiesProbe(ctx context.Context, origin string, authToken string) (dto.ActivitiesProbeResponse, error) {
	var endpoint = strings.TrimRight(origin, "/") + apiBasePath + "/activities?skip=0&take=1&sortColumn=date&sortDirection=asc"
	var err error
	var request *http.Request
	request, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return dto.ActivitiesProbeResponse{}, fmt.Errorf("build activities request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+authToken)

	var response *http.Response
	response, err = c.httpClient.Do(request)
	if err != nil {
		return dto.ActivitiesProbeResponse{}, classifyTransportFailure(err)
	}
	defer closeBody(response)

	switch response.StatusCode {
	case http.StatusBadRequest:
		return dto.ActivitiesProbeResponse{}, &RequestFailure{Category: FailureIncompatibleServerContract, Message: "activities request did not match the supported server contract"}
	case http.StatusUnauthorized, http.StatusForbidden:
		return dto.ActivitiesProbeResponse{}, &RequestFailure{Category: FailureUnsuccessfulServerResponse, Message: fmt.Sprintf("activities request returned HTTP %d", response.StatusCode)}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return dto.ActivitiesProbeResponse{}, &RequestFailure{Category: FailureUnsuccessfulServerResponse, Message: fmt.Sprintf("activities request returned HTTP %d", response.StatusCode)}
	}
	if err := requireJSONContentType(response.Header.Get("Content-Type")); err != nil {
		return dto.ActivitiesProbeResponse{}, &RequestFailure{Category: FailureIncompatibleServerContract, Message: err.Error(), Err: err}
	}

	var payload dto.ActivitiesProbeResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return dto.ActivitiesProbeResponse{}, &RequestFailure{Category: FailureIncompatibleServerContract, Message: "activities response was not valid JSON", Err: err}
	}

	return payload, nil
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

	return &RequestFailure{Category: FailureConnectivityProblem, Message: "could not reach the selected Ghostfolio server", Err: err}
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
