// Package store defines the protected snapshot persistence boundary.
// Authored by: OpenCode
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

// Test seams wrap JSON encoding so encrypted-store tests can inject payload
// encoding failures safely.
// Authored by: OpenCode
var marshalPayload = json.Marshal

// Test seams wrap secure random reads so encrypted-store tests can exercise
// random-identifier and nonce generation failures safely.
// Authored by: OpenCode
var readRandom = rand.Read

// Test seams wrap payload sealing so encrypted-store tests can inject envelope
// encryption failures safely.
// Authored by: OpenCode
var sealEnvelopeCiphertext = snapshotenvelope.SealCiphertext

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
// Use this constructor for the runtime sync workflow when discovery, decrypt,
// compatibility validation, encryption, and atomic replacement should stay
// behind one snapshot-store boundary.
// Authored by: OpenCode
func NewEncryptedStore(baseConfigDir string, codec snapshotenvelope.Codec) Store {
	var filesystem = NewFilesystemStore(baseConfigDir, codec)
	return &EncryptedStore{filesystem: filesystem, codec: filesystem.codec}
}

// Candidates enumerates protected snapshot headers before decrypt.
//
// Example:
//
//	candidates, err := store.Candidates(context.Background())
//	if err != nil {
//		panic(err)
//	}
//	_ = len(candidates)
//
// This method is intended for discovery flows that need to inspect cleartext
// headers and filter candidate snapshots before spending effort on unlock
// attempts.
// Authored by: OpenCode
func (s *EncryptedStore) Candidates(ctx context.Context) ([]Candidate, error) {
	return s.filesystem.Candidates(ctx)
}

// Read decrypts and decodes one protected snapshot payload.
//
// Example:
//
//	payload, err := store.Read(context.Background(), store.ReadRequest{Candidate: candidate, SecurityToken: "token"})
//	if err != nil {
//		panic(err)
//	}
//	_ = payload.StoredDataVersion.PayloadSchemaVersion
//
// Read expects a candidate discovered from the same store and the runtime token
// supplied for the current session. It validates the envelope header before
// decrypt and validates stored-data compatibility immediately after decode.
// Authored by: OpenCode
func (s *EncryptedStore) Read(ctx context.Context, request ReadRequest) (snapshotmodel.Payload, error) {
	if err := ctx.Err(); err != nil {
		return snapshotmodel.Payload{}, err
	}
	if strings.TrimSpace(request.SecurityToken) == "" {
		return snapshotmodel.Payload{}, fmt.Errorf("security token is required")
	}

	rawEnvelope, err := readFile(request.Candidate.Path)
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
//
// Example:
//
//	candidate, err := store.Write(context.Background(), store.WriteRequest{SnapshotID: "snapshot-1", SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: payload})
//	if err != nil {
//		panic(err)
//	}
//	_ = candidate.Path
//
// Write is intended for successful sync results only. It creates a fresh
// authenticated header, encrypts the serialized payload with the runtime token,
// and replaces the target snapshot file atomically.
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

	payloadBytes, err := marshalPayload(request.Payload)
	if err != nil {
		return Candidate{}, fmt.Errorf("encode protected snapshot payload: %w", err)
	}

	header, err := newEnvelopeHeader(request.ServerOrigin)
	if err != nil {
		return Candidate{}, err
	}

	ciphertext, err := sealEnvelopeCiphertext(header, request.SecurityToken, payloadBytes)
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
	if _, err := readRandom(buffer); err != nil {
		return nil, fmt.Errorf("read secure random bytes: %w", err)
	}
	return buffer, nil
}
