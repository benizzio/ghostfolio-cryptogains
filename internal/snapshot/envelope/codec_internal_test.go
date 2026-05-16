package envelope

import (
	"bytes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"testing"

	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
)

// TestJSONCodecEncodeCoversSuccessAndFailurePaths verifies JSON envelope
// encoding across supported success and injected failure branches.
// Authored by: OpenCode
func TestJSONCodecEncodeCoversSuccessAndFailurePaths(t *testing.T) {
	var codec = JSONCodec{}
	var envelope = snapshotEnvelopeFixture()

	encoded, err := codec.Encode(envelope)
	if err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("expected encoded envelope bytes")
	}

	invalid := envelope
	invalid.Header.Magic = "bad"
	if _, err := codec.Encode(invalid); err == nil {
		t.Fatalf("expected invalid header to fail encoding")
	}

	missingCiphertext := envelope
	missingCiphertext.Ciphertext = nil
	if _, err := codec.Encode(missingCiphertext); err == nil {
		t.Fatalf("expected missing ciphertext to fail encoding")
	}

	originalMarshal := marshalEnvelopeJSON
	marshalEnvelopeJSON = func(any) ([]byte, error) {
		return nil, errors.New("marshal boom")
	}
	t.Cleanup(func() {
		marshalEnvelopeJSON = originalMarshal
	})

	if _, err := codec.Encode(envelope); err == nil {
		t.Fatalf("expected injected marshal failure")
	}
}

// TestJSONCodecDecodeCoversSuccessAndFailurePaths verifies JSON envelope
// decoding across supported success and validation failure branches.
// Authored by: OpenCode
func TestJSONCodecDecodeCoversSuccessAndFailurePaths(t *testing.T) {
	var codec = JSONCodec{}
	var envelope = snapshotEnvelopeFixture()

	encoded, err := codec.Encode(envelope)
	if err != nil {
		t.Fatalf("encode envelope fixture: %v", err)
	}

	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if decoded.Header.Magic != snapshotmodel.EnvelopeMagic {
		t.Fatalf("unexpected decoded header: %#v", decoded.Header)
	}

	if _, err := codec.Decode([]byte("{")); err == nil {
		t.Fatalf("expected invalid JSON to fail decoding")
	}

	invalid := envelope
	invalid.Header.Magic = "bad"
	encoded, err = originalMarshalEnvelopeJSON(invalid)
	if err != nil {
		t.Fatalf("marshal invalid envelope: %v", err)
	}
	if _, err := codec.Decode(encoded); err == nil {
		t.Fatalf("expected invalid header to fail decoding")
	}

	missingCiphertext := envelope
	missingCiphertext.Ciphertext = nil
	encoded, err = originalMarshalEnvelopeJSON(missingCiphertext)
	if err != nil {
		t.Fatalf("marshal missing-ciphertext envelope: %v", err)
	}
	if _, err := codec.Decode(encoded); err == nil {
		t.Fatalf("expected missing ciphertext to fail decoding")
	}

	originalUnmarshal := unmarshalEnvelopeJSON
	unmarshalEnvelopeJSON = func([]byte, any) error {
		return errors.New("unmarshal boom")
	}
	t.Cleanup(func() {
		unmarshalEnvelopeJSON = originalUnmarshal
	})
	if _, err := codec.Decode(encoded); err == nil {
		t.Fatalf("expected injected unmarshal failure")
	}
}

// TestAuthenticatedHeaderBytesCoversSuccessAndFailures verifies cleartext
// header serialization across supported success and injected failure paths.
// Authored by: OpenCode
func TestAuthenticatedHeaderBytesCoversSuccessAndFailures(t *testing.T) {
	var header = snapshotEnvelopeHeaderFixture()

	encoded, err := AuthenticatedHeaderBytes(header)
	if err != nil {
		t.Fatalf("encode authenticated header: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("expected authenticated header bytes")
	}

	invalid := header
	invalid.Magic = "bad"
	if _, err := AuthenticatedHeaderBytes(invalid); err == nil {
		t.Fatalf("expected invalid header to fail")
	}

	originalMarshal := marshalEnvelopeJSON
	marshalEnvelopeJSON = func(any) ([]byte, error) {
		return nil, errors.New("marshal boom")
	}
	t.Cleanup(func() {
		marshalEnvelopeJSON = originalMarshal
	})
	if _, err := AuthenticatedHeaderBytes(header); err == nil {
		t.Fatalf("expected injected marshal failure")
	}
}

// TestDeriveEncryptionKeyAndCiphertextHelpersCoverBranches verifies key
// derivation and payload encryption helpers across supported branches.
// Authored by: OpenCode
func TestDeriveEncryptionKeyAndCiphertextHelpersCoverBranches(t *testing.T) {
	var header = snapshotEnvelopeHeaderFixture()

	key, err := DeriveEncryptionKey(header, "token")
	if err != nil {
		t.Fatalf("derive encryption key: %v", err)
	}
	if len(key) != int(header.KDFParameters.KeyLength) {
		t.Fatalf("unexpected key length: %d", len(key))
	}

	if _, err := DeriveEncryptionKey(header, "   "); err == nil {
		t.Fatalf("expected blank token to fail key derivation")
	}

	invalid := header
	invalid.Magic = "bad"
	if _, err := DeriveEncryptionKey(invalid, "token"); err == nil {
		t.Fatalf("expected invalid header to fail key derivation")
	}

	ciphertext, err := SealCiphertext(header, "token", []byte("payload"))
	if err != nil {
		t.Fatalf("seal ciphertext: %v", err)
	}
	plaintext, err := OpenCiphertext(header, "token", ciphertext)
	if err != nil {
		t.Fatalf("open ciphertext: %v", err)
	}
	if string(plaintext) != "payload" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}

	if _, err := SealCiphertext(invalid, "token", []byte("payload")); err == nil {
		t.Fatalf("expected invalid header to fail sealing")
	}
	if _, err := OpenCiphertext(header, "token", nil); err == nil {
		t.Fatalf("expected missing ciphertext to fail open")
	}
	if _, err := OpenCiphertext(header, "", []byte("ciphertext")); err == nil {
		t.Fatalf("expected blank token to fail open")
	}
	if _, err := OpenCiphertext(header, "wrong-token", ciphertext); err == nil {
		t.Fatalf("expected wrong token to fail decrypt")
	}

	nonceMismatch := header
	nonceMismatch.Nonce = []byte{1}
	if _, _, err := prepareAEAD(nonceMismatch, "token"); err == nil {
		t.Fatalf("expected nonce-size mismatch to fail")
	}

	invalidKey := header
	invalidKey.KDFParameters.KeyLength = 1
	if _, _, err := prepareAEAD(invalidKey, "token"); err == nil {
		t.Fatalf("expected invalid key length to fail")
	}

	originalMarshal := marshalEnvelopeJSON
	marshalEnvelopeJSON = func(any) ([]byte, error) {
		return nil, errors.New("marshal boom")
	}
	if _, _, err := prepareAEAD(header, "token"); err == nil {
		t.Fatalf("expected authenticated-header encoding failure")
	}
	marshalEnvelopeJSON = originalMarshal

	originalNewGCM := newGCM
	newGCM = func(cipher.Block) (cipher.AEAD, error) {
		return nil, errors.New("gcm boom")
	}
	t.Cleanup(func() {
		newGCM = originalNewGCM
	})
	if _, _, err := prepareAEAD(header, "token"); err == nil {
		t.Fatalf("expected injected GCM failure")
	}
}

// TestValidateEnvelopeHeaderCoversValidationBranches verifies every supported
// cleartext-header validation rule.
// Authored by: OpenCode
func TestValidateEnvelopeHeaderCoversValidationBranches(t *testing.T) {
	testCases := []struct {
		name   string
		header snapshotmodel.EnvelopeHeader
		wantOK bool
	}{
		{name: "valid", header: snapshotEnvelopeHeaderFixture(), wantOK: true},
		{name: "invalid magic", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.Magic = "bad"
			return header
		}()},
		{name: "invalid format version", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.FormatVersion = 0
			return header
		}()},
		{name: "invalid discovery key length", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.ServerDiscoveryKey = []byte{1}
			return header
		}()},
		{name: "invalid algorithm", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.KDFParameters.Algorithm = "bad"
			return header
		}()},
		{name: "invalid kdf version", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.KDFParameters.Version = 0
			return header
		}()},
		{name: "incomplete kdf parameters", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.KDFParameters.MemoryKiB = 0
			return header
		}()},
		{name: "missing salt", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.Salt = nil
			return header
		}()},
		{name: "missing nonce", header: func() snapshotmodel.EnvelopeHeader {
			header := snapshotEnvelopeHeaderFixture()
			header.Nonce = nil
			return header
		}()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateEnvelopeHeader(testCase.header)
			if testCase.wantOK && err != nil {
				t.Fatalf("expected validation success, got %v", err)
			}
			if !testCase.wantOK && err == nil {
				t.Fatalf("expected validation failure")
			}
		})
	}
}

// snapshotEnvelopeFixture returns one valid envelope fixture for internal codec
// tests.
// Authored by: OpenCode
func snapshotEnvelopeFixture() snapshotmodel.Envelope {
	return snapshotmodel.Envelope{
		Header:     snapshotEnvelopeHeaderFixture(),
		Ciphertext: []byte("ciphertext"),
	}
}

// snapshotEnvelopeHeaderFixture returns one valid cleartext snapshot header for
// internal codec tests.
// Authored by: OpenCode
func snapshotEnvelopeHeaderFixture() snapshotmodel.EnvelopeHeader {
	return snapshotmodel.EnvelopeHeader{
		Magic:              snapshotmodel.EnvelopeMagic,
		FormatVersion:      snapshotmodel.EnvelopeFormatVersion,
		ServerDiscoveryKey: bytes.Repeat([]byte{1}, snapshotmodel.ServerDiscoveryKeyLength),
		KDFParameters:      snapshotmodel.DefaultKDFParameters(),
		Salt:               bytes.Repeat([]byte{2}, snapshotmodel.DefaultSaltLength),
		Nonce:              bytes.Repeat([]byte{3}, snapshotmodel.DefaultNonceLength),
	}
}

// originalMarshalEnvelopeJSON preserves direct access to the default JSON
// encoder while tests swap package seams.
// Authored by: OpenCode
func originalMarshalEnvelopeJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}
