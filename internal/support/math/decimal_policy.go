package math

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const reportDecimalPolicyScalePrefix = "scale="
const reportDecimalPolicyRoundingPrefix = "rounding="
const reportDecimalPolicyHalfUp = "half_up"
const maximumDecimalPolicyScale int32 = 64

var activeDecimalPolicyState = decimalPolicyState{policy: DefaultDecimalPolicy()}

// decimalPolicyState stores the process-wide active decimal policy used by
// default arithmetic helpers.
// Authored by: OpenCode
type decimalPolicyState struct {
	mutex  sync.RWMutex
	policy DecimalPolicy
}

// DecimalPolicy captures one fixed-scale round-half-up calculation policy.
// Callers that need non-default behavior must parse or pass this value
// explicitly before calling the policy-aware decimal helpers.
//
// Example:
//
//	policy, err := math.ParseDecimalPolicy("scale=16,rounding=half_up")
//	if err != nil {
//		panic(err)
//	}
//	_ = policy.CanonicalString()
//
// Authored by: OpenCode
type DecimalPolicy struct {
	scale int32
}

// DefaultDecimalPolicy returns the production internal calculation policy.
// Production callers should use this policy unless an explicit test or adapter
// boundary has already selected a supported override.
//
// Example:
//
//	policy := math.DefaultDecimalPolicy()
//	_ = policy.CanonicalString()
//
// Authored by: OpenCode
func DefaultDecimalPolicy() DecimalPolicy {
	return DecimalPolicy{scale: InternalCalculationScale}
}

// ActiveDecimalPolicy returns the process-wide decimal policy currently used by
// default arithmetic helpers.
//
// Example:
//
//	policy := math.ActiveDecimalPolicy()
//	_ = policy.CanonicalString()
//
// Authored by: OpenCode
func ActiveDecimalPolicy() DecimalPolicy {
	activeDecimalPolicyState.mutex.RLock()
	defer activeDecimalPolicyState.mutex.RUnlock()

	return activeDecimalPolicyState.policy
}

// SetActiveDecimalPolicy validates and selects the process-wide decimal policy
// used by default arithmetic helpers.
//
// Example:
//
//	policy, err := math.ParseDecimalPolicy("scale=32,rounding=half_up")
//	if err != nil {
//		panic(err)
//	}
//	err = math.SetActiveDecimalPolicy(policy)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func SetActiveDecimalPolicy(policy DecimalPolicy) error {
	if err := validateDecimalPolicy(policy); err != nil {
		return fmt.Errorf("select active decimal policy %q: %w", policy.CanonicalString(), err)
	}

	activeDecimalPolicyState.mutex.Lock()
	defer activeDecimalPolicyState.mutex.Unlock()

	activeDecimalPolicyState.policy = policy
	return nil
}

// CanonicalString returns the canonical configuration-string form for one
// decimal policy.
// Authored by: OpenCode
func (policy DecimalPolicy) CanonicalString() string {
	return fmt.Sprintf(
		"%s%d,%s%s",
		reportDecimalPolicyScalePrefix,
		policy.scale,
		reportDecimalPolicyRoundingPrefix,
		reportDecimalPolicyHalfUp,
	)
}

// ParseDecimalPolicy validates one configured decimal policy string against the
// documented accepted values.
//
// Example:
//
//	policy, err := math.ParseDecimalPolicy("scale=16,rounding=half_up")
//	if err != nil {
//		panic(err)
//	}
//	_ = policy
//
// Authored by: OpenCode
func ParseDecimalPolicy(configuredValue string) (DecimalPolicy, error) {
	var scaleField, roundingField, found = strings.Cut(configuredValue, ",")
	if !found || !strings.HasPrefix(scaleField, reportDecimalPolicyScalePrefix) || roundingField != reportDecimalPolicyRoundingPrefix+reportDecimalPolicyHalfUp {
		return DecimalPolicy{}, fmt.Errorf("decimal policy %q must use the form scale=<digits>,rounding=%s", configuredValue, reportDecimalPolicyHalfUp)
	}

	var scaleText = strings.TrimPrefix(scaleField, reportDecimalPolicyScalePrefix)
	if scaleText == "" {
		return DecimalPolicy{}, fmt.Errorf("decimal policy %q must use the form scale=<digits>,rounding=%s", configuredValue, reportDecimalPolicyHalfUp)
	}
	for _, character := range scaleText {
		if character < '0' || character > '9' {
			return DecimalPolicy{}, fmt.Errorf("decimal policy %q must use the form scale=<digits>,rounding=%s", configuredValue, reportDecimalPolicyHalfUp)
		}
	}

	var scaleValue, err = strconv.ParseInt(scaleText, 10, 32)
	if err != nil {
		return DecimalPolicy{}, fmt.Errorf("parse decimal policy %q: %w", configuredValue, err)
	}

	var policy = DecimalPolicy{scale: int32(scaleValue)}
	if err = validateDecimalPolicy(policy); err != nil {
		return DecimalPolicy{}, err
	}

	return policy, nil
}

// validateDecimalPolicy rejects unsupported scale values for round-half-up
// decimal policies.
// Authored by: OpenCode
func validateDecimalPolicy(policy DecimalPolicy) error {
	if policy.scale < 0 {
		return fmt.Errorf("decimal policy %q scale must not be negative", policy.CanonicalString())
	}
	if policy.scale > maximumDecimalPolicyScale {
		return fmt.Errorf("decimal policy %q scale %d exceeds maximum supported scale %d", policy.CanonicalString(), policy.scale, maximumDecimalPolicyScale)
	}

	return nil
}
