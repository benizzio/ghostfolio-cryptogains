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
// Authored by: OpenCode
func ValidateEnvelopeCompatibility(header snapshotmodel.EnvelopeHeader) error {
	if !snapshotmodel.SupportsEnvelopeFormatVersion(header.FormatVersion) {
		return fmt.Errorf("envelope format version %d: %w", header.FormatVersion, ErrUnsupportedStoredDataVersion)
	}
	return nil
}

// ValidatePayloadCompatibility verifies that one decrypted payload uses a
// supported stored-data version set.
// Authored by: OpenCode
func ValidatePayloadCompatibility(payload snapshotmodel.Payload) error {
	if !snapshotmodel.SupportsStoredDataVersion(payload.StoredDataVersion) {
		return fmt.Errorf("payload stored-data version is unsupported: %w", ErrUnsupportedStoredDataVersion)
	}
	return nil
}
