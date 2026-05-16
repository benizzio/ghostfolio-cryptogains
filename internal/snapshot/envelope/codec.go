// Package envelope defines the protected snapshot envelope boundary.
// Authored by: OpenCode
package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	"golang.org/x/crypto/argon2"
)

// Test seams wrap JSON encoding so envelope tests can inject codec failures
// safely.
// Authored by: OpenCode
var marshalEnvelopeJSON = json.Marshal

// Test seams wrap JSON decoding so envelope tests can inject codec failures
// safely.
// Authored by: OpenCode
var unmarshalEnvelopeJSON = json.Unmarshal

// Test seams wrap GCM construction so envelope tests can inject AEAD creation
// failures safely.
// Authored by: OpenCode
var newGCM = cipher.NewGCM

// Codec defines the serialization boundary for protected snapshot envelopes.
//
// Example:
//
//	var codec Codec
//	_, _ = codec.Encode(snapshotmodel.Envelope{})
//
// Implementations are expected to serialize and validate the cleartext header
// and ciphertext container without owning filesystem persistence concerns.
// Authored by: OpenCode
type Codec interface {
	Encode(snapshotmodel.Envelope) ([]byte, error)
	Decode([]byte) (snapshotmodel.Envelope, error)
}

// JSONCodec serializes protected snapshot envelopes as a JSON document with a
// cleartext authenticated header and opaque ciphertext bytes.
// Authored by: OpenCode
type JSONCodec struct{}

// NewJSONCodec creates the default envelope codec used by snapshot discovery
// and persistence helpers.
//
// Example:
//
//	codec := envelope.NewJSONCodec()
//	_, _ = codec.Encode(snapshotmodel.Envelope{})
//
// Authored by: OpenCode
func NewJSONCodec() Codec {
	return JSONCodec{}
}

// Encode serializes one protected snapshot envelope.
// Authored by: OpenCode
func (JSONCodec) Encode(envelope snapshotmodel.Envelope) ([]byte, error) {
	if err := validateEnvelopeHeader(envelope.Header); err != nil {
		return nil, err
	}
	if len(envelope.Ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is required")
	}

	encoded, err := marshalEnvelopeJSON(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot envelope: %w", err)
	}

	return encoded, nil
}

// Decode deserializes one protected snapshot envelope.
// Authored by: OpenCode
func (JSONCodec) Decode(raw []byte) (snapshotmodel.Envelope, error) {
	var envelope snapshotmodel.Envelope
	if err := unmarshalEnvelopeJSON(raw, &envelope); err != nil {
		return snapshotmodel.Envelope{}, fmt.Errorf("decode snapshot envelope: %w", err)
	}
	if err := validateEnvelopeHeader(envelope.Header); err != nil {
		return snapshotmodel.Envelope{}, err
	}
	if len(envelope.Ciphertext) == 0 {
		return snapshotmodel.Envelope{}, fmt.Errorf("ciphertext is required")
	}

	return envelope, nil
}

// DeriveServerDiscoveryKey derives the server-scoped snapshot discovery key
// from one canonical Ghostfolio origin.
//
// Example:
//
//	key := envelope.DeriveServerDiscoveryKey("https://ghostfol.io")
//	_ = key
//
// Authored by: OpenCode
func DeriveServerDiscoveryKey(serverOrigin string) []byte {
	var sum = sha256.Sum256([]byte(strings.TrimSpace(serverOrigin)))
	var derived = make([]byte, len(sum))
	copy(derived, sum[:])
	return derived
}

// AuthenticatedHeaderBytes serializes the cleartext header bytes authenticated
// as AEAD additional authenticated data.
//
// Example:
//
//	header := snapshotmodel.EnvelopeHeader{Magic: snapshotmodel.EnvelopeMagic, FormatVersion: snapshotmodel.EnvelopeFormatVersion, ServerDiscoveryKey: make([]byte, snapshotmodel.ServerDiscoveryKeyLength), KDFParameters: snapshotmodel.DefaultKDFParameters(), Salt: make([]byte, snapshotmodel.DefaultSaltLength), Nonce: make([]byte, snapshotmodel.DefaultNonceLength)}
//	encoded, err := envelope.AuthenticatedHeaderBytes(header)
//	if err != nil {
//		panic(err)
//	}
//	_ = encoded
//
// Authored by: OpenCode
func AuthenticatedHeaderBytes(header snapshotmodel.EnvelopeHeader) ([]byte, error) {
	if err := validateEnvelopeHeader(header); err != nil {
		return nil, err
	}

	encoded, err := marshalEnvelopeJSON(header)
	if err != nil {
		return nil, fmt.Errorf("encode authenticated header: %w", err)
	}

	return encoded, nil
}

// DeriveEncryptionKey derives the AES-256 encryption key for one protected
// snapshot header and supplied runtime token.
//
// Authored by: OpenCode
func DeriveEncryptionKey(header snapshotmodel.EnvelopeHeader, securityToken string) ([]byte, error) {
	if strings.TrimSpace(securityToken) == "" {
		return nil, fmt.Errorf("security token is required")
	}
	if err := validateEnvelopeHeader(header); err != nil {
		return nil, err
	}

	var key = argon2.IDKey(
		[]byte(securityToken),
		header.Salt,
		header.KDFParameters.Iterations,
		header.KDFParameters.MemoryKiB,
		header.KDFParameters.Parallelism,
		header.KDFParameters.KeyLength,
	)

	return key, nil
}

// SealCiphertext encrypts one payload and authenticates the cleartext header as
// AEAD additional authenticated data.
//
// Authored by: OpenCode
func SealCiphertext(header snapshotmodel.EnvelopeHeader, securityToken string, plaintext []byte) ([]byte, error) {
	var aead, authenticatedHeader, err = prepareAEAD(header, securityToken)
	if err != nil {
		return nil, err
	}

	return aead.Seal(nil, header.Nonce, plaintext, authenticatedHeader), nil
}

// OpenCiphertext decrypts one payload and authenticates the cleartext header as
// AEAD additional authenticated data.
//
// Authored by: OpenCode
func OpenCiphertext(header snapshotmodel.EnvelopeHeader, securityToken string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is required")
	}

	var aead, authenticatedHeader, err = prepareAEAD(header, securityToken)
	if err != nil {
		return nil, err
	}

	plaintext, err := aead.Open(nil, header.Nonce, ciphertext, authenticatedHeader)
	if err != nil {
		return nil, fmt.Errorf("decrypt protected snapshot payload: %w", err)
	}

	return plaintext, nil
}

// prepareAEAD builds the AES-GCM instance and authenticated header bytes for
// one protected snapshot operation.
// Authored by: OpenCode
func prepareAEAD(header snapshotmodel.EnvelopeHeader, securityToken string) (cipher.AEAD, []byte, error) {
	var key, err = DeriveEncryptionKey(header, securityToken)
	if err != nil {
		return nil, nil, err
	}

	var block cipher.Block
	block, err = aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create AES cipher: %w", err)
	}

	var aead cipher.AEAD
	aead, err = newGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create AES-GCM instance: %w", err)
	}
	if len(header.Nonce) != aead.NonceSize() {
		return nil, nil, fmt.Errorf("nonce length must be %d bytes", aead.NonceSize())
	}

	authenticatedHeader, err := AuthenticatedHeaderBytes(header)
	if err != nil {
		return nil, nil, err
	}

	return aead, authenticatedHeader, nil
}

// validateEnvelopeHeader verifies the cleartext snapshot header shape accepted
// by the foundational codec and crypto helpers.
// Authored by: OpenCode
func validateEnvelopeHeader(header snapshotmodel.EnvelopeHeader) error {
	if header.Magic != snapshotmodel.EnvelopeMagic {
		return fmt.Errorf("snapshot magic is invalid")
	}
	if header.FormatVersion <= 0 {
		return fmt.Errorf("snapshot format version must be positive")
	}
	if len(header.ServerDiscoveryKey) != snapshotmodel.ServerDiscoveryKeyLength {
		return fmt.Errorf("server discovery key length is invalid")
	}
	if header.KDFParameters.Algorithm != snapshotmodel.KDFAlgorithmArgon2id {
		return fmt.Errorf("snapshot kdf algorithm is invalid")
	}
	if header.KDFParameters.Version != snapshotmodel.Argon2Version {
		return fmt.Errorf("snapshot kdf version is invalid")
	}
	if header.KDFParameters.MemoryKiB == 0 || header.KDFParameters.Iterations == 0 || header.KDFParameters.Parallelism == 0 || header.KDFParameters.KeyLength == 0 {
		return fmt.Errorf("snapshot kdf parameters are incomplete")
	}
	if len(header.Salt) == 0 {
		return fmt.Errorf("snapshot salt is required")
	}
	if len(header.Nonce) == 0 {
		return fmt.Errorf("snapshot nonce is required")
	}

	return nil
}
