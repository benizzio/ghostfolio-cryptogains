// Package store defines the protected snapshot persistence boundary.
// Authored by: OpenCode
package store

import (
	"errors"
	"fmt"

	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

var (
	// ErrUnsupportedStoredDataVersion indicates that a local protected snapshot
	// uses an unsupported envelope or payload version.
	ErrUnsupportedStoredDataVersion = errors.New("unsupported stored-data version")

	// ErrIncompatibleStoredData indicates that new protected data could not be
	// persisted safely in the current stored-data model.
	ErrIncompatibleStoredData = errors.New("incompatible stored data")
)

// ValidateEnvelopeCompatibility verifies that one cleartext header uses a
// supported envelope format version.
//
// Example:
//
//	err := store.ValidateEnvelopeCompatibility(candidate.Header)
//	if err != nil {
//		panic(err)
//	}
//
// Run this check after discovery and before decrypt so unsupported envelope
// versions fail without attempting token-derived unlock.
// Authored by: OpenCode
func ValidateEnvelopeCompatibility(header snapshotmodel.EnvelopeHeader) error {
	if !snapshotmodel.SupportsEnvelopeFormatVersion(header.FormatVersion) {
		return fmt.Errorf("envelope format version %d: %w", header.FormatVersion, ErrUnsupportedStoredDataVersion)
	}
	return nil
}

// ValidatePayloadCompatibility verifies that one decrypted payload uses a
// supported stored-data version set.
//
// Example:
//
//	err := store.ValidatePayloadCompatibility(payload)
//	if err != nil {
//		panic(err)
//	}
//
// Run this check immediately after decrypt and decode so unsupported stored-data
// versions fail before the runtime loads protected activity data.
// Authored by: OpenCode
func ValidatePayloadCompatibility(payload snapshotmodel.Payload) error {
	if !snapshotmodel.SupportsStoredDataVersion(payload.StoredDataVersion) {
		return fmt.Errorf("payload stored-data version is unsupported: %w", ErrUnsupportedStoredDataVersion)
	}
	return nil
}
