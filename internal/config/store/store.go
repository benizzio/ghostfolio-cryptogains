// Package store persists and reloads the bootstrap setup file.
// Authored by: OpenCode
package store

import (
	"context"
	"errors"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

// ErrNotFound indicates that no bootstrap setup file exists yet.
var ErrNotFound = errors.New("bootstrap setup file not found")

// Store defines the persistence contract for the bootstrap setup file.
//
// Example:
//
//	var bootstrapStore store.Store
//	_ = bootstrapStore
//
// Store implementations load and persist only startup-readable machine-local
// setup. `Load` returns `ErrNotFound` when no setup exists yet. `Save` is
// expected to replace the full document atomically, and `Delete` removes the
// remembered setup without treating an already-missing file as an error.
// `Path` returns the concrete location used for the saved setup file so tests
// and documentation can refer to it deterministically.
// Authored by: OpenCode
type Store interface {
	Load(context.Context) (configmodel.AppSetupConfig, error)
	Save(context.Context, configmodel.AppSetupConfig) error
	Delete(context.Context) error
	Path() string
}
