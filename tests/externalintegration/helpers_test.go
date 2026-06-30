// Package externalintegration contains opt-in live checks for official external
// provider clients.
// Authored by: OpenCode
package externalintegration

import (
	"os"
	"testing"
)

const externalIntegrationEnvironmentVariable = "GHOSTFOLIO_CRYPTOGAINS_RUN_EXTERNAL_INTEGRATION"

// requireExternalIntegration skips live external integration checks unless the
// dedicated environment variable is explicitly set to 1.
// Authored by: OpenCode
func requireExternalIntegration(t *testing.T) {
	t.Helper()

	if os.Getenv(externalIntegrationEnvironmentVariable) != "1" {
		t.Skipf("set %s=1 to run opt-in external currency provider integration tests", externalIntegrationEnvironmentVariable)
	}
}
