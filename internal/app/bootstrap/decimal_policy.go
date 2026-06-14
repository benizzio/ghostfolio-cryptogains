package bootstrap

import (
	"fmt"
	"os"

	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
)

const reportDecimalPolicyEnvironmentVariable = "GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY"

var lookupEnv = os.LookupEnv
var setActiveDecimalPolicy = supportmath.SetActiveDecimalPolicy

// ConfigureProcessDecimalPolicy reads the process decimal-policy override once
// from startup configuration and applies it to the shared math helpers.
//
// Example:
//
//	policy, err := bootstrap.ConfigureProcessDecimalPolicy()
//	if err != nil {
//		panic(err)
//	}
//	_ = policy.CanonicalString()
//
// Authored by: OpenCode
func ConfigureProcessDecimalPolicy() (supportmath.DecimalPolicy, error) {
	var policy = supportmath.DefaultDecimalPolicy()
	var configuredValue, configured = lookupEnv(reportDecimalPolicyEnvironmentVariable)
	if configured {
		var err error
		policy, err = supportmath.ParseDecimalPolicy(configuredValue)
		if err != nil {
			return supportmath.DecimalPolicy{}, fmt.Errorf(
				"configure %s=%q: %w",
				reportDecimalPolicyEnvironmentVariable,
				configuredValue,
				err,
			)
		}
	}

	if err := setActiveDecimalPolicy(policy); err != nil {
		return supportmath.DecimalPolicy{}, fmt.Errorf("configure %s: %w", reportDecimalPolicyEnvironmentVariable, err)
	}

	return policy, nil
}
