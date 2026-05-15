// Package store defines the protected snapshot persistence boundary.
// Authored by: OpenCode
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

// EncryptedStore implements token-derived protected snapshot read and write operations.
// Authored by: OpenCode
type EncryptedStore struct {
	filesystem *FilesystemStore
	codec      snapshotenvelope.Codec
}

// NewEncryptedStore creates the protected snapshot store used by the sync workflow.
//
// Example:
//
//	store := store.NewEncryptedStore("/tmp/config", envelope.NewJSONCodec())
//	_, _ = store.Candidates(context.Background())
//
// Authored by: OpenCode
func NewEncryptedStore(baseConfigDir string, codec snapshotenvelope.Codec) Store {
	var filesystem = NewFilesystemStore(baseConfigDir, codec)
	return &EncryptedStore{filesystem: filesystem, codec: filesystem.codec}
}

// Candidates enumerates protected snapshot headers before decrypt.
// Authored by: OpenCode
func (s *EncryptedStore) Candidates(ctx context.Context) ([]Candidate, error) {
	return s.filesystem.Candidates(ctx)
}

// Read decrypts and decodes one protected snapshot payload.
// Authored by: OpenCode
func (s *EncryptedStore) Read(ctx context.Context, request ReadRequest) (snapshotmodel.Payload, error) {
	if err := ctx.Err(); err != nil {
		return snapshotmodel.Payload{}, err
	}
	if strings.TrimSpace(request.SecurityToken) == "" {
		return snapshotmodel.Payload{}, fmt.Errorf("security token is required")
	}

	rawEnvelope, err := os.ReadFile(request.Candidate.Path)
	if err != nil {
		return snapshotmodel.Payload{}, fmt.Errorf("read snapshot file: %w", err)
	}

	envelopeDocument, err := s.codec.Decode(rawEnvelope)
	if err != nil {
		return snapshotmodel.Payload{}, err
	}
	if err := ValidateEnvelopeCompatibility(envelopeDocument.Header); err != nil {
		return snapshotmodel.Payload{}, err
	}

	plaintext, err := snapshotenvelope.OpenCiphertext(envelopeDocument.Header, request.SecurityToken, envelopeDocument.Ciphertext)
	if err != nil {
		return snapshotmodel.Payload{}, err
	}

	var payload snapshotmodel.Payload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return snapshotmodel.Payload{}, fmt.Errorf("decode protected snapshot payload: %w", err)
	}
	if err := ValidatePayloadCompatibility(payload); err != nil {
		return snapshotmodel.Payload{}, err
	}

	return payload, nil
}

// Write encrypts and atomically persists one protected snapshot payload.
// Authored by: OpenCode
func (s *EncryptedStore) Write(ctx context.Context, request WriteRequest) (Candidate, error) {
	if err := ctx.Err(); err != nil {
		return Candidate{}, err
	}
	if strings.TrimSpace(request.SecurityToken) == "" {
		return Candidate{}, fmt.Errorf("security token is required")
	}
	if strings.TrimSpace(request.ServerOrigin) == "" {
		return Candidate{}, fmt.Errorf("server origin is required")
	}

	payloadBytes, err := json.Marshal(request.Payload)
	if err != nil {
		return Candidate{}, fmt.Errorf("encode protected snapshot payload: %w", err)
	}

	header, err := newEnvelopeHeader(request.ServerOrigin)
	if err != nil {
		return Candidate{}, err
	}

	ciphertext, err := snapshotenvelope.SealCiphertext(header, request.SecurityToken, payloadBytes)
	if err != nil {
		return Candidate{}, err
	}

	rawEnvelope, err := s.codec.Encode(snapshotmodel.Envelope{Header: header, Ciphertext: ciphertext})
	if err != nil {
		return Candidate{}, err
	}

	snapshotID := strings.TrimSpace(request.SnapshotID)
	if snapshotID == "" {
		snapshotID, err = randomIdentifier(16)
		if err != nil {
			return Candidate{}, err
		}
	}

	path := s.filesystem.SnapshotPath(snapshotID)
	if err := ReplaceFileAtomically(path, rawEnvelope); err != nil {
		return Candidate{}, err
	}

	return Candidate{SnapshotID: snapshotID, Path: path, Header: header}, nil
}

// newEnvelopeHeader creates the authenticated cleartext header for one snapshot write.
// Authored by: OpenCode
func newEnvelopeHeader(serverOrigin string) (snapshotmodel.EnvelopeHeader, error) {
	salt, err := randomBytes(snapshotmodel.DefaultSaltLength)
	if err != nil {
		return snapshotmodel.EnvelopeHeader{}, err
	}
	nonce, err := randomBytes(snapshotmodel.DefaultNonceLength)
	if err != nil {
		return snapshotmodel.EnvelopeHeader{}, err
	}

	return snapshotmodel.EnvelopeHeader{
		Magic:              snapshotmodel.EnvelopeMagic,
		FormatVersion:      snapshotmodel.EnvelopeFormatVersion,
		ServerDiscoveryKey: snapshotenvelope.DeriveServerDiscoveryKey(serverOrigin),
		KDFParameters:      snapshotmodel.DefaultKDFParameters(),
		Salt:               salt,
		Nonce:              nonce,
	}, nil
}

// randomIdentifier creates one opaque hexadecimal identifier.
// Authored by: OpenCode
func randomIdentifier(byteLength int) (string, error) {
	rawValue, err := randomBytes(byteLength)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(rawValue), nil
}

// randomBytes returns securely random bytes for salts, nonces, and identifiers.
// Authored by: OpenCode
func randomBytes(length int) ([]byte, error) {
	var buffer = make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return nil, fmt.Errorf("read secure random bytes: %w", err)
	}
	return buffer, nil
}
