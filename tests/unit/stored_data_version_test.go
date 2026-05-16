package unit

import (
	"errors"
	"testing"

	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
)

func TestValidateEnvelopeCompatibilityRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	err := snapshotstore.ValidateEnvelopeCompatibility(snapshotmodel.EnvelopeHeader{
		Magic:              snapshotmodel.EnvelopeMagic,
		FormatVersion:      snapshotmodel.EnvelopeFormatVersion + 1,
		ServerDiscoveryKey: make([]byte, snapshotmodel.ServerDiscoveryKeyLength),
		KDFParameters:      snapshotmodel.DefaultKDFParameters(),
		Salt:               make([]byte, snapshotmodel.DefaultSaltLength),
		Nonce:              make([]byte, snapshotmodel.DefaultNonceLength),
	})
	if !errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
		t.Fatalf("expected unsupported stored-data version error, got %v", err)
	}
}

func TestValidatePayloadCompatibilityRejectsUnsupportedPayloadAndActivityVersions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		mutate func(snapshotmodel.StoredDataVersion) snapshotmodel.StoredDataVersion
	}{
		{
			name: "payload schema",
			mutate: func(version snapshotmodel.StoredDataVersion) snapshotmodel.StoredDataVersion {
				version.PayloadSchemaVersion++
				return version
			},
		},
		{
			name: "activity model",
			mutate: func(version snapshotmodel.StoredDataVersion) snapshotmodel.StoredDataVersion {
				version.ActivityModelVersion++
				return version
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			payload := defaultRawSnapshotPayload("https://ghostfol.io")
			payload.StoredDataVersion = testCase.mutate(payload.StoredDataVersion)

			err := snapshotstore.ValidatePayloadCompatibility(payload)
			if !errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
				t.Fatalf("expected unsupported stored-data version error, got %v", err)
			}
		})
	}
}

func TestSupportsStoredDataVersionAcceptsCurrentVersions(t *testing.T) {
	t.Parallel()

	if !snapshotmodel.SupportsStoredDataVersion(snapshotmodel.DefaultStoredDataVersion("")) {
		t.Fatalf("expected current stored-data version to be supported")
	}
}
