package runtimeflow

import (
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

// MustCloudSetupConfig creates a valid Cloud setup fixture.
// Authored by: OpenCode
func MustCloudSetupConfig(t *testing.T) configmodel.AppSetupConfig {
	t.Helper()
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	return config
}
