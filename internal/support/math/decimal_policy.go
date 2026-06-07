package math

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const reportDecimalPolicyEnvironmentVariable = "GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY"
const reportDecimalPolicyScalePrefix = "scale="
const reportDecimalPolicyRoundingPrefix = "rounding="
const reportDecimalPolicyHalfUp = "half_up"

// decimalPolicy captures one fixed-scale round-half-up calculation policy.
// Authored by: OpenCode
type decimalPolicy struct {
	scale int32
}

// defaultDecimalPolicy returns the production internal calculation policy.
// Authored by: OpenCode
func defaultDecimalPolicy() decimalPolicy {
	return decimalPolicy{scale: InternalCalculationScale}
}

// canonicalString returns the canonical environment-variable form for one
// decimal policy.
// Authored by: OpenCode
func (policy decimalPolicy) canonicalString() string {
	return fmt.Sprintf(
		"%s%d,%s%s",
		reportDecimalPolicyScalePrefix,
		policy.scale,
		reportDecimalPolicyRoundingPrefix,
		reportDecimalPolicyHalfUp,
	)
}

// selectedDecimalPolicy resolves the active internal calculation decimal policy
// from GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY or falls back to the
// production default when the variable is unset.
// Authored by: OpenCode
func selectedDecimalPolicy() (decimalPolicy, error) {
	var configuredValue, isSet = os.LookupEnv(reportDecimalPolicyEnvironmentVariable)
	if !isSet {
		return defaultDecimalPolicy(), nil
	}

	return parseDecimalPolicy(configuredValue)
}

// parseDecimalPolicy validates one configured decimal policy string against the
// documented accepted values.
// Authored by: OpenCode
func parseDecimalPolicy(configuredValue string) (decimalPolicy, error) {
	var scaleField, roundingField, found = strings.Cut(configuredValue, ",")
	if !found || !strings.HasPrefix(scaleField, reportDecimalPolicyScalePrefix) || roundingField != reportDecimalPolicyRoundingPrefix+reportDecimalPolicyHalfUp {
		return decimalPolicy{}, fmt.Errorf("decimal policy %q must use the form scale=<digits>,rounding=%s", configuredValue, reportDecimalPolicyHalfUp)
	}

	var scaleText = strings.TrimPrefix(scaleField, reportDecimalPolicyScalePrefix)
	if scaleText == "" {
		return decimalPolicy{}, fmt.Errorf("decimal policy %q must use the form scale=<digits>,rounding=%s", configuredValue, reportDecimalPolicyHalfUp)
	}
	for _, character := range scaleText {
		if character < '0' || character > '9' {
			return decimalPolicy{}, fmt.Errorf("decimal policy %q must use the form scale=<digits>,rounding=%s", configuredValue, reportDecimalPolicyHalfUp)
		}
	}

	var scaleValue, err = strconv.ParseInt(scaleText, 10, 32)
	if err != nil {
		return decimalPolicy{}, fmt.Errorf("parse decimal policy %q: %w", configuredValue, err)
	}

	var policy = decimalPolicy{scale: int32(scaleValue)}
	if policy.scale != InternalCalculationScale || policy.canonicalString() != configuredValue {
		return decimalPolicy{}, fmt.Errorf("decimal policy %q is not supported; supported value: %q", configuredValue, defaultDecimalPolicy().canonicalString())
	}

	return policy, nil
}
