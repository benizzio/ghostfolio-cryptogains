package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

// TestEncryptedStoreReadCoversBranches verifies protected payload reads across
// success and failure branches.
// Authored by: OpenCode
func TestEncryptedStoreReadCoversBranches(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := NewEncryptedStore(t.TempDir(), nil).Read(ctx, ReadRequest{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context, got %v", err)
		}
	})

	t.Run("blank token", func(t *testing.T) {
		_, err := NewEncryptedStore(t.TempDir(), nil).Read(context.Background(), ReadRequest{SecurityToken: "   "})
		if err == nil {
			t.Fatalf("expected blank token to fail")
		}
	})

	t.Run("read file error", func(t *testing.T) {
		_, err := NewEncryptedStore(t.TempDir(), nil).Read(context.Background(), ReadRequest{
			Candidate:     Candidate{Path: filepath.Join(t.TempDir(), "missing.snapshot")},
			SecurityToken: "token",
		})
		if err == nil {
			t.Fatalf("expected read-file failure")
		}
	})

	t.Run("decode error", func(t *testing.T) {
		var baseDir = t.TempDir()
		var path = filepath.Join(baseDir, "broken.snapshot")
		if err := os.WriteFile(path, []byte("raw"), snapshotFileMode); err != nil {
			t.Fatalf("write broken snapshot: %v", err)
		}
		var codec = stubCodec{decode: func([]byte) (snapshotmodel.Envelope, error) {
			return snapshotmodel.Envelope{}, errors.New("decode boom")
		}}
		var filesystem = NewFilesystemStore(baseDir, codec)
		var store = &EncryptedStore{filesystem: filesystem, codec: filesystem.codec}

		_, err := store.Read(context.Background(), ReadRequest{Candidate: Candidate{Path: path}, SecurityToken: "token"})
		if err == nil {
			t.Fatalf("expected decode failure")
		}
	})

	t.Run("unsupported envelope version", func(t *testing.T) {
		var candidate = writeEncryptedSnapshotFixture(t, t.TempDir(), encryptedSnapshotFixture{
			SnapshotID:    "unsupported-envelope",
			ServerOrigin:  "https://ghostfol.io",
			SecurityToken: "token",
			Header: func() snapshotmodel.EnvelopeHeader {
				header := storeHeaderFixture("https://ghostfol.io")
				header.FormatVersion++
				return header
			}(),
			Payload: encryptedStorePayloadFixture("https://ghostfol.io"),
		})

		_, err := NewEncryptedStore(t.TempDir(), nil).Read(context.Background(), ReadRequest{Candidate: candidate, SecurityToken: "token"})
		if !errors.Is(err, ErrUnsupportedStoredDataVersion) {
			t.Fatalf("expected unsupported envelope version, got %v", err)
		}
	})

	t.Run("decrypt failure", func(t *testing.T) {
		var candidate = writeEncryptedSnapshotFixture(t, t.TempDir(), encryptedSnapshotFixture{
			SnapshotID:    "wrong-token",
			ServerOrigin:  "https://ghostfol.io",
			SecurityToken: "correct-token",
			Header:        storeHeaderFixture("https://ghostfol.io"),
			Payload:       encryptedStorePayloadFixture("https://ghostfol.io"),
		})

		_, err := NewEncryptedStore(t.TempDir(), nil).Read(context.Background(), ReadRequest{Candidate: candidate, SecurityToken: "wrong-token"})
		if err == nil {
			t.Fatalf("expected wrong token to fail decryption")
		}
	})

	t.Run("invalid payload json", func(t *testing.T) {
		var candidate = writeEncryptedRawPayloadFixture(t, t.TempDir(), encryptedRawPayloadFixture{
			SnapshotID:    "invalid-json",
			ServerOrigin:  "https://ghostfol.io",
			SecurityToken: "token",
			Header:        storeHeaderFixture("https://ghostfol.io"),
			Plaintext:     []byte("{"),
		})

		_, err := NewEncryptedStore(t.TempDir(), nil).Read(context.Background(), ReadRequest{Candidate: candidate, SecurityToken: "token"})
		if err == nil {
			t.Fatalf("expected invalid payload JSON to fail")
		}
	})

	t.Run("unsupported payload version", func(t *testing.T) {
		payload := encryptedStorePayloadFixture("https://ghostfol.io")
		payload.StoredDataVersion.PayloadSchemaVersion++
		var candidate = writeEncryptedSnapshotFixture(t, t.TempDir(), encryptedSnapshotFixture{
			SnapshotID:    "unsupported-payload",
			ServerOrigin:  "https://ghostfol.io",
			SecurityToken: "token",
			Header:        storeHeaderFixture("https://ghostfol.io"),
			Payload:       payload,
		})

		_, err := NewEncryptedStore(t.TempDir(), nil).Read(context.Background(), ReadRequest{Candidate: candidate, SecurityToken: "token"})
		if !errors.Is(err, ErrUnsupportedStoredDataVersion) {
			t.Fatalf("expected unsupported payload version, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		var baseDir = t.TempDir()
		var payload = encryptedStorePayloadFixture("https://ghostfol.io")
		var candidate = writeEncryptedSnapshotFixture(t, baseDir, encryptedSnapshotFixture{
			SnapshotID:    "success",
			ServerOrigin:  "https://ghostfol.io",
			SecurityToken: "token",
			Header:        storeHeaderFixture("https://ghostfol.io"),
			Payload:       payload,
		})

		loaded, err := NewEncryptedStore(baseDir, nil).Read(context.Background(), ReadRequest{Candidate: candidate, SecurityToken: "token"})
		if err != nil {
			t.Fatalf("read encrypted snapshot: %v", err)
		}
		if loaded.SetupProfile.ServerOrigin != payload.SetupProfile.ServerOrigin {
			t.Fatalf("unexpected loaded payload: %#v", loaded)
		}
	})
}

// TestEncryptedStoreWriteCoversBranches verifies protected payload writes across
// success and failure branches.
// Authored by: OpenCode
func TestEncryptedStoreWriteCoversBranches(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := NewEncryptedStore(t.TempDir(), nil).Write(ctx, WriteRequest{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context, got %v", err)
		}
	})

	t.Run("blank token", func(t *testing.T) {
		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{ServerOrigin: "https://ghostfol.io"})
		if err == nil {
			t.Fatalf("expected blank token to fail")
		}
	})

	t.Run("blank server origin", func(t *testing.T) {
		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SecurityToken: "token"})
		if err == nil {
			t.Fatalf("expected blank server origin to fail")
		}
	})

	t.Run("marshal payload error", func(t *testing.T) {
		originalMarshalPayload := marshalPayload
		marshalPayload = func(any) ([]byte, error) {
			return nil, errors.New("marshal boom")
		}
		defer func() {
			marshalPayload = originalMarshalPayload
		}()

		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err == nil {
			t.Fatalf("expected marshal failure")
		}
	})

	t.Run("new header error", func(t *testing.T) {
		originalReadRandom := readRandom
		readRandom = func([]byte) (int, error) {
			return 0, errors.New("random boom")
		}
		defer func() {
			readRandom = originalReadRandom
		}()

		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err == nil {
			t.Fatalf("expected header-generation failure")
		}
	})

	t.Run("seal ciphertext error", func(t *testing.T) {
		originalSeal := sealEnvelopeCiphertext
		sealEnvelopeCiphertext = func(snapshotmodel.EnvelopeHeader, string, []byte) ([]byte, error) {
			return nil, errors.New("seal boom")
		}
		defer func() {
			sealEnvelopeCiphertext = originalSeal
		}()

		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err == nil {
			t.Fatalf("expected sealing failure")
		}
	})

	t.Run("codec encode error", func(t *testing.T) {
		var baseDir = t.TempDir()
		var codec = stubCodec{encode: func(snapshotmodel.Envelope) ([]byte, error) {
			return nil, errors.New("encode boom")
		}}
		var filesystem = NewFilesystemStore(baseDir, codec)
		var store = &EncryptedStore{filesystem: filesystem, codec: filesystem.codec}

		_, err := store.Write(context.Background(), WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err == nil {
			t.Fatalf("expected codec encode failure")
		}
	})

	t.Run("generated snapshot identifier error", func(t *testing.T) {
		originalReadRandom := readRandom
		var calls int
		readRandom = func(buffer []byte) (int, error) {
			calls++
			if calls == 3 {
				return 0, errors.New("identifier boom")
			}
			for index := range buffer {
				buffer[index] = byte(calls)
			}
			return len(buffer), nil
		}
		defer func() {
			readRandom = originalReadRandom
		}()

		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err == nil {
			t.Fatalf("expected generated identifier failure")
		}
	})

	t.Run("replace file error", func(t *testing.T) {
		originalCreateTempFile := createTempFile
		createTempFile = func(string, string) (temporaryFile, error) {
			return nil, errors.New("temp boom")
		}
		defer func() {
			createTempFile = originalCreateTempFile
		}()

		_, err := NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SnapshotID: "snapshot-1", SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err == nil {
			t.Fatalf("expected atomic replace failure")
		}
	})

	t.Run("success with provided snapshot identifier", func(t *testing.T) {
		var baseDir = t.TempDir()
		var candidate, err = NewEncryptedStore(baseDir, nil).Write(context.Background(), WriteRequest{SnapshotID: "snapshot-1", SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err != nil {
			t.Fatalf("write encrypted snapshot: %v", err)
		}
		if candidate.SnapshotID != "snapshot-1" {
			t.Fatalf("unexpected candidate: %#v", candidate)
		}
	})

	t.Run("success with generated snapshot identifier", func(t *testing.T) {
		var candidate, err = NewEncryptedStore(t.TempDir(), nil).Write(context.Background(), WriteRequest{SecurityToken: "token", ServerOrigin: "https://ghostfol.io", Payload: encryptedStorePayloadFixture("https://ghostfol.io")})
		if err != nil {
			t.Fatalf("write encrypted snapshot with generated identifier: %v", err)
		}
		if candidate.SnapshotID == "" {
			t.Fatalf("expected generated snapshot identifier")
		}
	})
}

// TestEncryptedStoreCandidatesDelegatesToFilesystemStore verifies the
// candidates helper delegates to filesystem discovery.
// Authored by: OpenCode
func TestEncryptedStoreCandidatesDelegatesToFilesystemStore(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := NewEncryptedStore(baseDir, nil)
	writeEncryptedSnapshotFixture(t, baseDir, encryptedSnapshotFixture{
		SnapshotID:    "snapshot-1",
		ServerOrigin:  "https://ghostfol.io",
		SecurityToken: "token",
		Header:        storeHeaderFixture("https://ghostfol.io"),
		Payload:       encryptedStorePayloadFixture("https://ghostfol.io"),
	})

	candidates, err := store.Candidates(context.Background())
	if err != nil {
		t.Fatalf("enumerate encrypted-store candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].SnapshotID != "snapshot-1" {
		t.Fatalf("unexpected encrypted-store candidates: %#v", candidates)
	}
}

// TestNewEnvelopeHeaderCoversBranches verifies authenticated-header generation
// across success and partial random-read failure branches.
// Authored by: OpenCode
func TestNewEnvelopeHeaderCoversBranches(t *testing.T) {
	t.Parallel()

	header, err := newEnvelopeHeader("https://ghostfol.io")
	if err != nil {
		t.Fatalf("new envelope header: %v", err)
	}
	if header.Magic != snapshotmodel.EnvelopeMagic || header.FormatVersion != snapshotmodel.EnvelopeFormatVersion {
		t.Fatalf("unexpected envelope header: %#v", header)
	}
	if len(header.ServerDiscoveryKey) != snapshotmodel.ServerDiscoveryKeyLength {
		t.Fatalf("unexpected discovery-key length: %d", len(header.ServerDiscoveryKey))
	}

	originalReadRandom := readRandom
	defer func() {
		readRandom = originalReadRandom
	}()

	var calls int
	readRandom = func([]byte) (int, error) {
		calls++
		if calls == 2 {
			return 0, errors.New("nonce boom")
		}
		return snapshotmodel.DefaultSaltLength, nil
	}

	if _, err := newEnvelopeHeader("https://ghostfol.io"); err == nil {
		t.Fatalf("expected nonce-generation error")
	}
}

// encryptedSnapshotFixture stores one fully specified protected snapshot fixture
// for encrypted-store tests.
// Authored by: OpenCode
type encryptedSnapshotFixture struct {
	SnapshotID    string
	ServerOrigin  string
	SecurityToken string
	Header        snapshotmodel.EnvelopeHeader
	Payload       snapshotmodel.Payload
}

// encryptedRawPayloadFixture stores one protected snapshot fixture with custom
// raw plaintext bytes for encrypted-store tests.
// Authored by: OpenCode
type encryptedRawPayloadFixture struct {
	SnapshotID    string
	ServerOrigin  string
	SecurityToken string
	Header        snapshotmodel.EnvelopeHeader
	Plaintext     []byte
}

// writeEncryptedSnapshotFixture persists one encrypted snapshot fixture for
// internal store tests.
// Authored by: OpenCode
func writeEncryptedSnapshotFixture(t *testing.T, baseDir string, fixture encryptedSnapshotFixture) Candidate {
	t.Helper()

	payloadBytes, err := json.Marshal(fixture.Payload)
	if err != nil {
		t.Fatalf("marshal encrypted payload fixture: %v", err)
	}
	return writeEncryptedRawPayloadFixture(t, baseDir, encryptedRawPayloadFixture{
		SnapshotID:    fixture.SnapshotID,
		ServerOrigin:  fixture.ServerOrigin,
		SecurityToken: fixture.SecurityToken,
		Header:        fixture.Header,
		Plaintext:     payloadBytes,
	})
}

// writeEncryptedRawPayloadFixture persists one encrypted snapshot fixture with
// custom plaintext bytes for internal store tests.
// Authored by: OpenCode
func writeEncryptedRawPayloadFixture(t *testing.T, baseDir string, fixture encryptedRawPayloadFixture) Candidate {
	t.Helper()

	var store = NewFilesystemStore(baseDir, snapshotenvelope.NewJSONCodec())
	var header = fixture.Header
	if len(header.ServerDiscoveryKey) == 0 {
		header = storeHeaderFixture(fixture.ServerOrigin)
	}

	ciphertext, err := snapshotenvelope.SealCiphertext(header, fixture.SecurityToken, fixture.Plaintext)
	if err != nil {
		t.Fatalf("seal encrypted payload fixture: %v", err)
	}
	rawEnvelope, err := store.codec.Encode(snapshotmodel.Envelope{Header: header, Ciphertext: ciphertext})
	if err != nil {
		t.Fatalf("encode encrypted envelope fixture: %v", err)
	}
	if err := ReplaceFileAtomically(store.SnapshotPath(fixture.SnapshotID), rawEnvelope); err != nil {
		t.Fatalf("write encrypted envelope fixture: %v", err)
	}

	return Candidate{SnapshotID: fixture.SnapshotID, Path: store.SnapshotPath(fixture.SnapshotID), Header: header}
}

// encryptedStorePayloadFixture returns one valid protected payload fixture for
// encrypted-store tests.
// Authored by: OpenCode
func encryptedStorePayloadFixture(serverOrigin string) snapshotmodel.Payload {
	return snapshotmodel.Payload{
		StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""),
		RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{
			LocalUserID:          "user-1",
			CreatedAt:            time.Unix(1, 0).UTC(),
			UpdatedAt:            time.Unix(1, 0).UTC(),
			LastSuccessfulSyncAt: time.Unix(1, 0).UTC(),
		},
		SetupProfile: snapshotmodel.SetupProfile{
			ServerOrigin:      serverOrigin,
			ServerMode:        "custom_origin",
			LastValidatedAt:   time.Unix(1, 0).UTC(),
			SourceAPIBasePath: "api/v1",
		},
		ProtectedActivityCache: syncmodel.ProtectedActivityCache{
			SyncedAt:             time.Unix(1, 0).UTC(),
			RetrievedCount:       0,
			ActivityCount:        0,
			AvailableReportYears: []int{},
			Activities:           []syncmodel.ActivityRecord{},
		},
	}
}
