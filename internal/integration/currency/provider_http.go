// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const providerRequestTimeout = 15 * time.Second

// fetchProviderPayload performs one fixed-provider HTTP GET and returns the body.
// Authored by: OpenCode
func fetchProviderPayload(ctx context.Context, httpClient *http.Client, endpoint string) ([]byte, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, providerRequestTimeout)
		defer cancel()
	}
	var request, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build provider request: %w", err)
	}
	var response, doErr = httpClient.Do(request)
	if doErr != nil {
		return nil, fmt.Errorf("request provider evidence: %w", doErr)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil, fmt.Errorf("provider returned HTTP status %d", response.StatusCode)
	}
	var payload, readErr = io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read provider evidence: %w", readErr)
	}

	return payload, nil
}
