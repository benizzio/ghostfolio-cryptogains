// Package model defines the protected snapshot data structures shared across
// snapshot packages.
// Authored by: OpenCode
package model

import (
	"crypto/sha256"

	"golang.org/x/crypto/argon2"
)

const (
	// EnvelopeMagic identifies files written by the protected snapshot feature.
	EnvelopeMagic = "GFCSNP"

	// EnvelopeFormatVersion is the supported on-disk envelope version.
	EnvelopeFormatVersion = 1

	// KDFAlgorithmArgon2id names the supported token-derived key derivation
	// algorithm.
	KDFAlgorithmArgon2id = "argon2id"

	// Argon2Version records the upstream package version used by the envelope
	// metadata.
	Argon2Version = argon2.Version

	// ServerDiscoveryKeyLength is the SHA-256 length used to scope snapshot discovery.
	ServerDiscoveryKeyLength = sha256.Size

	// DefaultSaltLength is the default random salt length used by the envelope.
	DefaultSaltLength = 16

	// DefaultNonceLength is the default AES-GCM nonce length used by the envelope.
	DefaultNonceLength = 12

	// DefaultKeyLength is the default AES-256 key length derived from the token.
	DefaultKeyLength = 32

	// DefaultArgon2MemoryKiB is the default Argon2id memory cost persisted in KiB.
	DefaultArgon2MemoryKiB = 64 * 1024

	// DefaultArgon2Iterations is the default Argon2id iteration cost.
	DefaultArgon2Iterations = 3

	// DefaultArgon2Parallelism is the default Argon2id parallelism cost.
	DefaultArgon2Parallelism = 1
)

// Envelope stores the authenticated cleartext header plus the encrypted payload
// bytes for one protected snapshot.
// Authored by: OpenCode
type Envelope struct {
	Header     EnvelopeHeader `json:"header"`
	Ciphertext []byte         `json:"ciphertext"`
}

// EnvelopeHeader contains only the metadata needed before decrypt.
// Authored by: OpenCode
type EnvelopeHeader struct {
	Magic              string        `json:"magic"`
	FormatVersion      int           `json:"format_version"`
	ServerDiscoveryKey []byte        `json:"server_discovery_key"`
	KDFParameters      KDFParameters `json:"kdf_parameters"`
	Salt               []byte        `json:"salt"`
	Nonce              []byte        `json:"nonce"`
}

// KDFParameters captures the persisted Argon2id cost settings for one
// protected snapshot.
// Authored by: OpenCode
type KDFParameters struct {
	Algorithm   string `json:"algorithm"`
	Version     int    `json:"version"`
	MemoryKiB   uint32 `json:"memory_kib"`
	Iterations  uint32 `json:"iterations"`
	Parallelism uint8  `json:"parallelism"`
	KeyLength   uint32 `json:"key_length"`
}

// DefaultKDFParameters returns the supported baseline Argon2id profile for new
// protected snapshots.
//
// Example:
//
//	parameters := model.DefaultKDFParameters()
//	_ = parameters.Algorithm
//
// Authored by: OpenCode
func DefaultKDFParameters() KDFParameters {
	return KDFParameters{
		Algorithm:   KDFAlgorithmArgon2id,
		Version:     Argon2Version,
		MemoryKiB:   DefaultArgon2MemoryKiB,
		Iterations:  DefaultArgon2Iterations,
		Parallelism: DefaultArgon2Parallelism,
		KeyLength:   DefaultKeyLength,
	}
}
