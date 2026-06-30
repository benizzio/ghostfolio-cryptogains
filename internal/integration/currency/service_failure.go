// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
// Authored by: OpenCode
package currency

import (
	"errors"
	"strings"
)

// classifyLookupFailure converts provider and mapper failures into the public
// conversion-failure contract.
// Authored by: OpenCode
func classifyLookupFailure(request RateLookupRequest, providerID ProviderID, err error) error {
	var failure *ConversionFailure
	if errors.As(err, &failure) {
		return err
	}

	return NewConversionFailure(request, providerID, conversionFailureReasonForProviderError(err), err.Error())
}

// classifyEvidenceFailure converts malformed or mismatched canonical evidence
// into the public conversion-failure contract.
// Authored by: OpenCode
func classifyEvidenceFailure(request RateLookupRequest, providerID ProviderID, err error) error {
	var failure *ConversionFailure
	if errors.As(err, &failure) {
		return err
	}
	var reason = ConversionFailureReasonMalformedRate
	var message = strings.ToLower(err.Error())
	if strings.Contains(message, "quote direction") {
		reason = ConversionFailureReasonAmbiguousQuote
	} else if isAuthorityMismatchMessage(message) {
		reason = ConversionFailureReasonAuthorityMismatch
	}

	return NewConversionFailure(request, providerID, reason, err.Error())
}

// conversionFailureReasonForRequestError classifies malformed public lookup
// requests before provider IO is attempted.
// Authored by: OpenCode
func conversionFailureReasonForRequestError(err error) ConversionFailureReason {
	var message = strings.ToLower(err.Error())
	if strings.Contains(message, "unsupported source currency") ||
		strings.Contains(message, "unsupported base currency") {
		return ConversionFailureReasonUnsupportedCurrency
	}

	return ConversionFailureReasonInvalidActivityCurrency
}

// conversionFailureReasonForProviderError classifies provider lookup and mapper
// errors without exposing raw provider details to callers.
// Authored by: OpenCode
func conversionFailureReasonForProviderError(err error) ConversionFailureReason {
	var message = strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unsupported source currency") || strings.Contains(message, "unsupported base currency"):
		return ConversionFailureReasonUnsupportedCurrency
	case strings.Contains(message, "no current or prior available observation"):
		return ConversionFailureReasonMissingRate
	case strings.Contains(message, "ambiguous quote direction"):
		return ConversionFailureReasonAmbiguousQuote
	case isProviderUnavailableMessage(message):
		return ConversionFailureReasonProviderUnavailable
	default:
		return ConversionFailureReasonMalformedRate
	}
}

// isAuthorityMismatchMessage reports whether validation text indicates provider
// identity or authority evidence does not match the lookup request.
// Authored by: OpenCode
func isAuthorityMismatchMessage(message string) bool {
	return strings.Contains(message, "provider selection") ||
		strings.Contains(message, "requires provider") ||
		strings.Contains(message, "requires authority") ||
		strings.Contains(message, "lookup identity") ||
		strings.Contains(message, "does not match")
}

// isProviderUnavailableMessage reports whether provider IO failed before a
// defensible rate observation could be mapped.
// Authored by: OpenCode
func isProviderUnavailableMessage(message string) bool {
	return strings.Contains(message, "provider returned http status") ||
		strings.Contains(message, "request provider evidence") ||
		strings.Contains(message, "read provider evidence") ||
		strings.Contains(message, "build provider request") ||
		strings.Contains(message, "provider") && strings.Contains(message, "context")
}
