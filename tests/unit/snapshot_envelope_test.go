// Package unit verifies focused exact-decimal and snapshot helpers for the
// sync-and-storage slice.
// Authored by: OpenCode
package unit

import (
	"bytes"
	"testing"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

// TestSnapshotEnvelopeAuthenticatesHeaderBytes verifies that header tampering
// breaks protected payload decryption.
// Authored by: OpenCode
func TestSnapshotEnvelopeAuthenticatesHeaderBytes(t *testing.T) {
	t.Parallel()

	header := snapshotmodel.EnvelopeHeader{
		Magic:              snapshotmodel.EnvelopeMagic,
		FormatVersion:      snapshotmodel.EnvelopeFormatVersion,
		ServerDiscoveryKey: snapshotenvelope.DeriveServerDiscoveryKey("https://ghostfol.io"),
		KDFParameters:      snapshotmodel.DefaultKDFParameters(),
		Salt:               bytes.Repeat([]byte{1}, snapshotmodel.DefaultSaltLength),
		Nonce:              bytes.Repeat([]byte{2}, snapshotmodel.DefaultNonceLength),
	}

	ciphertext, err := snapshotenvelope.SealCiphertext(header, "token", []byte("payload"))
	if err != nil {
		t.Fatalf("seal ciphertext: %v", err)
	}

	header.ServerDiscoveryKey[0] ^= 0xff
	if _, err := snapshotenvelope.OpenCiphertext(header, "token", ciphertext); err == nil {
		t.Fatalf("expected header tampering to fail decryption")
	}
}
