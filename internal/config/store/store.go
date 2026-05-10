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
// Authored by: OpenCode
type Store interface {
	Load(context.Context) (configmodel.AppSetupConfig, error)
	Save(context.Context, configmodel.AppSetupConfig) error
	Delete(context.Context) error
	Path() string
}
