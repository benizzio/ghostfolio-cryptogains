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

const invalidRememberedSetupMessage = "The saved server selection is no longer valid. Complete setup again before sync validation can run."

// StartupState captures the initial bootstrap outcome used to choose the first
// TUI workflow screen.
//
// Example:
//
//	state, err := bootstrap.LoadStartupState(context.Background(), bootstrapStore, false)
//	if err != nil {
//		panic(err)
//	}
//	_ = state.NeedsSetup
//
// Authored by: OpenCode
type StartupState struct {
	ActiveConfig        *configmodel.AppSetupConfig
	NeedsSetup          bool
	InvalidSetupMessage string
}

// LoadStartupState loads and validates the remembered bootstrap setup for a
// new application run.
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
			return StartupState{NeedsSetup: true}, nil
		}
		return StartupState{}, fmt.Errorf("load bootstrap setup: %w", err)
	}

	if err := config.ValidateStartupReady(allowDevHTTP); err != nil {
		return StartupState{NeedsSetup: true, InvalidSetupMessage: invalidRememberedSetupMessage}, nil
	}

	return StartupState{ActiveConfig: &config}, nil
}
