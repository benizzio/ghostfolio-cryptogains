// Package bootstrap contains startup configuration and bootstrap helpers for
// the terminal application.
// Authored by: OpenCode
package bootstrap

import (
	"context"
	"errors"
	"fmt"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

// SetupRequirementReason identifies why startup routing requires the setup
// workflow instead of the main menu.
//
// Authored by: OpenCode
type SetupRequirementReason string

const (
	// SetupRequirementNone indicates that no startup fallback explanation is
	// needed for the current setup workflow entry.
	SetupRequirementNone SetupRequirementReason = ""

	// SetupRequirementMissing indicates that no remembered setup exists yet.
	SetupRequirementMissing SetupRequirementReason = "missing"

	// SetupRequirementInvalidRememberedSetup indicates that persisted setup data
	// exists but no longer satisfies the bootstrap validation rules.
	SetupRequirementInvalidRememberedSetup SetupRequirementReason = "invalid_remembered_setup"
)

// StartupState captures the initial bootstrap outcome used to choose the first
// TUI workflow screen.
//
// Authored by: OpenCode
type StartupState struct {
	ActiveConfig           *configmodel.AppSetupConfig
	NeedsSetup             bool
	SetupRequirementReason SetupRequirementReason
}

// LoadStartupState loads and validates the remembered bootstrap setup for a
// new application run.
//
// The returned state uses `ActiveConfig` when startup may proceed directly to
// the main menu. When setup is still required, callers should use
// `SetupRequirementReason` to decide whether the TUI needs to explain a
// missing first-run setup or an invalid remembered setup.
//
// Example:
//
//	state, err := bootstrap.LoadStartupState(context.Background(), bootstrapStore, true)
//	if err != nil {
//		panic(err)
//	}
//	_ = state.ActiveConfig
//
// Authored by: OpenCode
func LoadStartupState(ctx context.Context, bootstrapStore configstore.Store, allowDevHTTP bool) (StartupState, error) {
	var config, err = bootstrapStore.Load(ctx)
	if err != nil {
		if errors.Is(err, configstore.ErrNotFound) {
			return StartupState{NeedsSetup: true, SetupRequirementReason: SetupRequirementMissing}, nil
		}
		return StartupState{}, fmt.Errorf("load bootstrap setup: %w", err)
	}

	if config.ValidateStartupReady(allowDevHTTP) == nil {
		return StartupState{ActiveConfig: &config}, nil
	}

	return StartupState{NeedsSetup: true, SetupRequirementReason: SetupRequirementInvalidRememberedSetup}, nil
}
