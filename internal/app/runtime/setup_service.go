// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

// SaveSetupRequest contains the setup selection that should be normalized and
// persisted for startup use.
//
// Authored by: OpenCode
type SaveSetupRequest struct {
	ServerMode   string
	ServerOrigin string
	SavedAt      time.Time
}

// SaveSetupResult contains the normalized setup configuration written by the
// application layer.
//
// Authored by: OpenCode
type SaveSetupResult struct {
	Config configmodel.AppSetupConfig
}

// SetupService validates and persists the bootstrap setup chosen in the TUI.
//
// The service centralizes setup normalization and persistence so the TUI layer
// only manages input state and screen transitions.
//
// Authored by: OpenCode
type SetupService interface {
	Save(context.Context, SaveSetupRequest) (SaveSetupResult, error)
}

// setupService implements setup normalization and persistence behind the
// application-facing setup service boundary.
//
// Authored by: OpenCode
type setupService struct {
	store        configstore.Store
	allowDevHTTP bool
}

// NewSetupService creates the runtime service used to normalize and persist
// bootstrap setup selections.
//
// Example:
//
//	service := runtime.NewSetupService(store, true)
//	_ = service
//
// The returned service applies the same startup normalization rules that later
// reads use, so persisted setup stays aligned with bootstrap validation.
// Authored by: OpenCode
func NewSetupService(store configstore.Store, allowDevHTTP bool) SetupService {
	return &setupService{store: store, allowDevHTTP: allowDevHTTP}
}

// Save validates the selected server mode and origin, writes the normalized
// bootstrap setup, and returns the persisted configuration.
//
// Example:
//
//	result, err := service.Save(context.Background(), runtime.SaveSetupRequest{
//		ServerMode:   configmodel.ServerModeCustomOrigin,
//		ServerOrigin: "https://ghostfolio.example",
//		SavedAt:      time.Now(),
//	})
//	if err != nil {
//		panic(err)
//	}
//	_ = result.Config.ServerOrigin
//
// Save returns validation errors from `configmodel.NewSetupConfig` unchanged so
// callers can surface the exact input problem, and it returns persistence
// errors from the configured `store` when disk writes fail.
// Authored by: OpenCode
func (s *setupService) Save(ctx context.Context, request SaveSetupRequest) (SaveSetupResult, error) {
	var config, err = configmodel.NewSetupConfig(
		request.ServerMode,
		request.ServerOrigin,
		s.allowDevHTTP,
		request.SavedAt,
	)
	if err != nil {
		return SaveSetupResult{}, err
	}

	err = s.store.Save(ctx, config)
	if err != nil {
		return SaveSetupResult{}, err
	}

	return SaveSetupResult{Config: config}, nil
}
