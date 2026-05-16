package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/cockroachdb/apd/v3"
)

// runtimeSnapshotStore is a test-only protected-snapshot store implementation
// for runtime-service coverage.
// Authored by: OpenCode
type runtimeSnapshotStore struct {
	candidates    []snapshotstore.Candidate
	candidatesErr error
	read          func(context.Context, snapshotstore.ReadRequest) (snapshotmodel.Payload, error)
	write         func(context.Context, snapshotstore.WriteRequest) (snapshotstore.Candidate, error)
}

// Candidates returns injected snapshot candidates or an injected discovery
// error.
// Authored by: OpenCode
func (s runtimeSnapshotStore) Candidates(context.Context) ([]snapshotstore.Candidate, error) {
	if s.candidatesErr != nil {
		return nil, s.candidatesErr
	}
	return s.candidates, nil
}

// Read returns an injected protected payload or an injected unlock error.
// Authored by: OpenCode
func (s runtimeSnapshotStore) Read(ctx context.Context, request snapshotstore.ReadRequest) (snapshotmodel.Payload, error) {
	if s.read != nil {
		return s.read(ctx, request)
	}
	return snapshotmodel.Payload{}, errors.New("read not configured")
}

// Write returns an injected protected-write result or the default candidate for
// the request.
// Authored by: OpenCode
func (s runtimeSnapshotStore) Write(ctx context.Context, request snapshotstore.WriteRequest) (snapshotstore.Candidate, error) {
	if s.write != nil {
		return s.write(ctx, request)
	}
	return snapshotstore.Candidate{SnapshotID: request.SnapshotID, Path: filepath.Join("/tmp", request.SnapshotID)}, nil
}

// runtimeDiagnosticCarrierError exposes diagnostic context for runtime helper
// coverage.
// Authored by: OpenCode
type runtimeDiagnosticCarrierError struct {
	context syncmodel.DiagnosticContext
}

type runtimeFailingDecimalService struct{}

func (runtimeFailingDecimalService) ParseString(string) (apd.Decimal, string, error) {
	return apd.Decimal{}, "", errors.New("parse string boom")
}

func (runtimeFailingDecimalService) ParseNumber(json.Number) (apd.Decimal, string, error) {
	return apd.Decimal{}, "", errors.New("parse number boom")
}

func (runtimeFailingDecimalService) CanonicalString(apd.Decimal) (string, error) {
	return "", errors.New("canonical boom")
}

func (runtimeFailingDecimalService) CanonicalStringPointer(*apd.Decimal) (string, error) {
	return "", errors.New("canonical pointer boom")
}

// Error returns the test failure message.
// Authored by: OpenCode
func (e runtimeDiagnosticCarrierError) Error() string {
	return "carrier boom"
}

// DiagnosticContext returns the injected troubleshooting context.
// Authored by: OpenCode
func (e runtimeDiagnosticCarrierError) DiagnosticContext() syncmodel.DiagnosticContext {
	return e.context
}

// TestWriteDiagnosticReportCoversBranches verifies structured diagnostic-report
// writing across validation, encoding, persistence, and success branches.
// Authored by: OpenCode
func TestWriteDiagnosticReportCoversBranches(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := writeDiagnosticReport(ctx, t.TempDir(), runtimeDiagnosticRequestFixture())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled context, got %v", err)
		}
	})

	t.Run("missing failure reason", func(t *testing.T) {
		request := runtimeDiagnosticRequestFixture()
		request.FailureReason = SyncFailureNone
		if _, err := writeDiagnosticReport(context.Background(), t.TempDir(), request); err == nil {
			t.Fatalf("expected missing failure reason to fail")
		}
	})

	t.Run("missing server origin", func(t *testing.T) {
		request := runtimeDiagnosticRequestFixture()
		request.ServerOrigin = ""
		if _, err := writeDiagnosticReport(context.Background(), t.TempDir(), request); err == nil {
			t.Fatalf("expected missing server origin to fail")
		}
	})

	t.Run("random identifier error", func(t *testing.T) {
		originalReadRandom := readRandom
		readRandom = func([]byte) (int, error) {
			return 0, errors.New("random boom")
		}
		defer func() {
			readRandom = originalReadRandom
		}()

		if _, err := writeDiagnosticReport(context.Background(), t.TempDir(), runtimeDiagnosticRequestFixture()); err == nil {
			t.Fatalf("expected random identifier error")
		}
	})

	t.Run("marshal error", func(t *testing.T) {
		originalMarshal := marshalDiagnosticReport
		marshalDiagnosticReport = func(any, string, string) ([]byte, error) {
			return nil, errors.New("marshal boom")
		}
		defer func() {
			marshalDiagnosticReport = originalMarshal
		}()

		if _, err := writeDiagnosticReport(context.Background(), t.TempDir(), runtimeDiagnosticRequestFixture()); err == nil {
			t.Fatalf("expected marshal error")
		}
	})

	t.Run("atomic replace error", func(t *testing.T) {
		var baseConfigPath = filepath.Join(t.TempDir(), "base.file")
		if err := os.WriteFile(baseConfigPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("write base config file: %v", err)
		}

		if _, err := writeDiagnosticReport(context.Background(), baseConfigPath, runtimeDiagnosticRequestFixture()); err == nil {
			t.Fatalf("expected atomic replace error")
		}
	})

	t.Run("success", func(t *testing.T) {
		var baseDir = t.TempDir()
		request := runtimeDiagnosticRequestFixture()
		request.RedactFinancialValues = true

		path, err := writeDiagnosticReport(context.Background(), baseDir, request)
		if err != nil {
			t.Fatalf("write diagnostic report: %v", err)
		}
		if !strings.Contains(path, filepath.Join(applicationDirectoryName, diagnosticsDirectoryName)) {
			t.Fatalf("expected diagnostics path, got %q", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read diagnostic report: %v", err)
		}
		if !strings.HasSuffix(string(raw), "\n") {
			t.Fatalf("expected newline-terminated diagnostic report")
		}
		if strings.Contains(string(raw), "\"quantity\": \"1\"") {
			t.Fatalf("expected financial values to be redacted, got %s", raw)
		}
	})
}

// TestResolveBaseConfigDirAndGenerateDiagnosticReportCoverBranches verifies
// base-directory resolution and service-level report generation branches.
// Authored by: OpenCode
func TestResolveBaseConfigDirAndGenerateDiagnosticReportCoverBranches(t *testing.T) {
	if got, err := resolveBaseConfigDir("/tmp/config"); err != nil || got != "/tmp/config" {
		t.Fatalf("expected explicit config dir to be preserved, got %q err=%v", got, err)
	}

	originalResolveUserConfigDir := resolveUserConfigDir
	defer func() {
		resolveUserConfigDir = originalResolveUserConfigDir
	}()

	resolveUserConfigDir = func() (string, error) {
		return "", errors.New("resolve boom")
	}
	if _, err := resolveBaseConfigDir(""); err == nil {
		t.Fatalf("expected user-config-dir resolution error")
	}

	var tempDir = t.TempDir()
	resolveUserConfigDir = func() (string, error) {
		return tempDir, nil
	}
	if got, err := resolveBaseConfigDir(""); err != nil || got != tempDir {
		t.Fatalf("expected resolved user config dir, got %q err=%v", got, err)
	}

	var service = NewSyncService(nil, time.Second, "", true, nil, nil, nil, runtimeSnapshotStore{}).(*syncService)
	path, err := service.GenerateDiagnosticReport(context.Background(), runtimeDiagnosticRequestFixture())
	if err != nil {
		t.Fatalf("generate diagnostic report: %v", err)
	}
	if path == "" {
		t.Fatalf("expected generated diagnostic report path")
	}

	resolveUserConfigDir = func() (string, error) {
		return "", errors.New("resolve boom")
	}
	if _, err := service.GenerateDiagnosticReport(context.Background(), runtimeDiagnosticRequestFixture()); err == nil {
		t.Fatalf("expected generate-diagnostic-report resolution error")
	}
}

// TestNewSyncServiceAndHelperFunctionsCoverBranches verifies constructor
// defaulting and local helper branches.
// Authored by: OpenCode
func TestNewSyncServiceAndHelperFunctionsCoverBranches(t *testing.T) {
	service := NewSyncService(nil, time.Second, "/tmp/config", true, nil, nil, nil, runtimeSnapshotStore{}).(*syncService)
	if service.client == nil || service.decimalService == nil || service.normalizer == nil || service.validator == nil {
		t.Fatalf("expected nil dependencies to be defaulted: %#v", service)
	}

	eligibleCases := []SyncFailureReason{
		SyncFailureUnsupportedActivityHistory,
		SyncFailureUnsupportedStoredDataVersion,
		SyncFailureIncompatibleNewSyncData,
	}
	for _, reason := range eligibleCases {
		if !diagnosticEligible(reason) {
			t.Fatalf("expected %q to be diagnostic eligible", reason)
		}
	}
	if diagnosticEligible(SyncFailureTimeout) {
		t.Fatalf("expected timeout to be diagnostic ineligible")
	}

	contextFromNil := diagnosticContextFromError(nil, syncmodel.DiagnosticFailureStageValidation, "token")
	if contextFromNil.FailureStage != syncmodel.DiagnosticFailureStageValidation || contextFromNil.FailureDetail != "" {
		t.Fatalf("expected default diagnostic context for nil error, got %#v", contextFromNil)
	}

	contextFromPlain := diagnosticContextFromError(errors.New("token boom"), syncmodel.DiagnosticFailureStageValidation, "token")
	if strings.Contains(contextFromPlain.FailureDetail, "token") {
		t.Fatalf("expected secret to be redacted from plain error, got %#v", contextFromPlain)
	}

	contextFromCarrier := diagnosticContextFromError(runtimeDiagnosticCarrierError{context: syncmodel.DiagnosticContext{FailureDetail: "token detail"}}, syncmodel.DiagnosticFailureStageNormalization, "token")
	if contextFromCarrier.FailureStage != syncmodel.DiagnosticFailureStageNormalization {
		t.Fatalf("expected default stage to be applied, got %#v", contextFromCarrier)
	}
	if strings.Contains(contextFromCarrier.FailureDetail, "token") {
		t.Fatalf("expected carrier detail to be redacted, got %#v", contextFromCarrier)
	}

	carrierWithStage := diagnosticContextFromError(runtimeDiagnosticCarrierError{context: syncmodel.DiagnosticContext{FailureStage: syncmodel.DiagnosticFailureStageMapping, FailureDetail: "mapped"}}, syncmodel.DiagnosticFailureStageNormalization)
	if carrierWithStage.FailureStage != syncmodel.DiagnosticFailureStageMapping {
		t.Fatalf("expected explicit carrier stage to be preserved, got %#v", carrierWithStage)
	}

	var cache = runtimeCacheFixture()
	var config = runtimeSetupConfigFixture(t, "https://ghostfol.io", true)

	payload := (protectedPayloadBuilder{}).Build(protectedPayloadBuildRequest{Config: config, Cache: cache})
	if payload.RegisteredLocalUser.LocalUserID == "" {
		t.Fatalf("expected generated local user id")
	}

	existing := snapshotmodel.Payload{RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{LocalUserID: "user-1", CreatedAt: time.Unix(1, 0).UTC()}}
	payload = (protectedPayloadBuilder{}).Build(protectedPayloadBuildRequest{Config: config, Cache: cache, ExistingPayload: existing, HasExisting: true})
	if payload.RegisteredLocalUser.LocalUserID != "user-1" || payload.RegisteredLocalUser.LastSuccessfulSyncAt != cache.SyncedAt.UTC() {
		t.Fatalf("expected existing local-user identity to be reused, got %#v", payload.RegisteredLocalUser)
	}

	originalReadRandom := readRandom
	readRandom = func([]byte) (int, error) {
		return 0, errors.New("random boom")
	}
	defer func() {
		readRandom = originalReadRandom
	}()

	payload = (protectedPayloadBuilder{}).Build(protectedPayloadBuildRequest{Config: config, Cache: cache})
	if payload.RegisteredLocalUser.LocalUserID != "" {
		t.Fatalf("expected empty local-user id when identifier generation fails, got %#v", payload.RegisteredLocalUser)
	}
	if _, err := randomIdentifier(8); err == nil {
		t.Fatalf("expected random identifier failure")
	}

	readRandom = func(buffer []byte) (int, error) {
		for index := range buffer {
			buffer[index] = byte(index + 1)
		}
		return len(buffer), nil
	}
	id, err := randomIdentifier(4)
	if err != nil {
		t.Fatalf("generate random identifier: %v", err)
	}
	if len(id) != 8 {
		t.Fatalf("expected hex identifier, got %q", id)
	}
}

// TestFinalizeSyncFailureCoversDiagnosticBranches verifies result construction
// for ineligible failures, manual diagnostic eligibility, and dev-mode
// automatic report generation.
// Authored by: OpenCode
func TestFinalizeSyncFailureCoversDiagnosticBranches(t *testing.T) {
	t.Run("ineligible failure", func(t *testing.T) {
		service := &syncService{}
		session := &GhostfolioSession{ServerOrigin: "https://ghostfol.io", SecurityToken: "token", AuthToken: "jwt"}
		attempt := &SyncAttempt{}

		outcome := service.finalizeSyncFailure(session, attempt, SyncFailureTimeout, syncmodel.DiagnosticContext{})
		if outcome.Diagnostic.Eligible {
			t.Fatalf("expected timeout failure to remain diagnostic ineligible")
		}
		if session.SecurityToken != "" || session.AuthToken != "" {
			t.Fatalf("expected secrets to be cleared after finalizeSyncFailure")
		}
	})

	t.Run("eligible failure in production mode", func(t *testing.T) {
		service := &syncService{allowDevHTTP: false}
		outcome := service.finalizeSyncFailure(&GhostfolioSession{ServerOrigin: "https://ghostfol.io"}, &SyncAttempt{}, SyncFailureUnsupportedActivityHistory, syncmodel.DiagnosticContext{})
		if !outcome.Diagnostic.Eligible || outcome.Diagnostic.Path != "" {
			t.Fatalf("expected manual diagnostic eligibility, got %#v", outcome.Diagnostic)
		}
		if !outcome.Diagnostic.Request.RedactFinancialValues || outcome.Diagnostic.Request.ExplicitDevelopmentMode {
			t.Fatalf("expected production-mode redaction request, got %#v", outcome.Diagnostic.Request)
		}
	})

	t.Run("eligible failure in development mode writes report", func(t *testing.T) {
		service := &syncService{allowDevHTTP: true, diagnosticReports: newDiagnosticReportService(t.TempDir())}
		outcome := service.finalizeSyncFailure(&GhostfolioSession{ServerOrigin: "https://ghostfol.io"}, &SyncAttempt{}, SyncFailureIncompatibleNewSyncData, runtimeDiagnosticRequestFixture().Context)
		if !outcome.Diagnostic.Eligible || outcome.Diagnostic.Path == "" {
			t.Fatalf("expected automatic diagnostic report path, got %#v", outcome.Diagnostic)
		}
	})

	t.Run("development mode ignores report write errors", func(t *testing.T) {
		var baseConfigPath = filepath.Join(t.TempDir(), "base.file")
		if err := os.WriteFile(baseConfigPath, []byte("content"), 0o600); err != nil {
			t.Fatalf("write base config file: %v", err)
		}

		service := &syncService{allowDevHTTP: true, diagnosticReports: newDiagnosticReportService(baseConfigPath)}
		outcome := service.finalizeSyncFailure(&GhostfolioSession{ServerOrigin: "https://ghostfol.io"}, &SyncAttempt{}, SyncFailureUnsupportedStoredDataVersion, runtimeDiagnosticRequestFixture().Context)
		if !outcome.Diagnostic.Eligible || outcome.Diagnostic.Path != "" {
			t.Fatalf("expected dev-mode report failure to be ignored, got %#v", outcome.Diagnostic)
		}
	})
}

// TestValidateCoversProtectedStorageFailureBranches verifies runtime sync
// outcomes across snapshot discovery, compatibility, unlock, and persistence
// failure branches.
// Authored by: OpenCode
func TestValidateCoversProtectedStorageFailureBranches(t *testing.T) {
	t.Run("snapshot store unavailable", func(t *testing.T) {
		service := NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, nil).(*syncService)
		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, "https://ghostfol.io", true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected unavailable snapshot store failure, got %#v", outcome)
		}
	})

	t.Run("server replacement cancelled", func(t *testing.T) {
		var payload = snapshotmodel.Payload{SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: "https://old.example"}}
		service := NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, runtimeSnapshotStore{}).(*syncService)
		service.snapshots.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "snapshot-1"}, payload)

		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, "https://new.example", true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureServerReplacementCancelled || outcome.Attempt.Status != AttemptStatusFailed {
			t.Fatalf("expected replacement cancellation failure, got %#v", outcome)
		}
	})

	t.Run("candidate discovery error", func(t *testing.T) {
		service := NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, runtimeSnapshotStore{candidatesErr: errors.New("discover boom")}).(*syncService)
		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, "https://ghostfol.io", true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected discovery failure outcome, got %#v", outcome)
		}
	})

	t.Run("unsupported envelope version", func(t *testing.T) {
		config := runtimeSetupConfigFixture(t, "https://ghostfol.io", true)
		originalValidateEnvelopeCompatibility := validateSnapshotEnvelopeCompatibility
		validateSnapshotEnvelopeCompatibility = func(snapshotmodel.EnvelopeHeader) error {
			return snapshotstore.ErrUnsupportedStoredDataVersion
		}
		defer func() {
			validateSnapshotEnvelopeCompatibility = originalValidateEnvelopeCompatibility
		}()

		store := runtimeSnapshotStore{candidates: []snapshotstore.Candidate{runtimeSnapshotCandidateFixture(config.ServerOrigin, "snapshot-1")}}
		service := NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, store).(*syncService)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedStoredDataVersion {
			t.Fatalf("expected unsupported stored-data version, got %#v", outcome)
		}
	})

	t.Run("generic envelope compatibility error", func(t *testing.T) {
		config := runtimeSetupConfigFixture(t, "https://ghostfol.io", true)
		originalValidateEnvelopeCompatibility := validateSnapshotEnvelopeCompatibility
		validateSnapshotEnvelopeCompatibility = func(snapshotmodel.EnvelopeHeader) error {
			return errors.New("compatibility boom")
		}
		defer func() {
			validateSnapshotEnvelopeCompatibility = originalValidateEnvelopeCompatibility
		}()

		store := runtimeSnapshotStore{candidates: []snapshotstore.Candidate{runtimeSnapshotCandidateFixture(config.ServerOrigin, "snapshot-1")}}
		service := NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, store).(*syncService)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected incompatible new sync data failure, got %#v", outcome)
		}
	})

	t.Run("unlock finds unsupported payload version", func(t *testing.T) {
		config := runtimeSetupConfigFixture(t, "https://ghostfol.io", true)
		store := runtimeSnapshotStore{
			candidates: []snapshotstore.Candidate{runtimeSnapshotCandidateFixture(config.ServerOrigin, "snapshot-1")},
			read: func(context.Context, snapshotstore.ReadRequest) (snapshotmodel.Payload, error) {
				return snapshotmodel.Payload{}, snapshotstore.ErrUnsupportedStoredDataVersion
			},
		}
		service := NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, store).(*syncService)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedStoredDataVersion {
			t.Fatalf("expected unsupported stored-data version from unlock, got %#v", outcome)
		}
	})
}

// TestValidateCoversProtectedWriteBranches verifies runtime sync outcomes across
// successful unlock reuse and persistence failure categories.
// Authored by: OpenCode
func TestValidateCoversProtectedWriteBranches(t *testing.T) {
	t.Run("reuse unlocked snapshot and confirm replacement", func(t *testing.T) {
		var wroteRequest snapshotstore.WriteRequest
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Path {
			case "/api/v1/auth/anonymous":
				_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
			case "/api/v1/activities":
				_, _ = writer.Write([]byte(`{"activities":[],"count":0}`))
			default:
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		config := runtimeSetupConfigFixture(t, server.URL, true)
		store := runtimeSnapshotStore{
			candidates: []snapshotstore.Candidate{runtimeSnapshotCandidateFixture(config.ServerOrigin, "snapshot-1")},
			read: func(context.Context, snapshotstore.ReadRequest) (snapshotmodel.Payload, error) {
				return snapshotmodel.Payload{RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{LocalUserID: "user-1"}}, nil
			},
			write: func(_ context.Context, request snapshotstore.WriteRequest) (snapshotstore.Candidate, error) {
				wroteRequest = request
				return snapshotstore.Candidate{SnapshotID: request.SnapshotID, Path: filepath.Join(t.TempDir(), request.SnapshotID)}, nil
			},
		}
		service := NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), store).(*syncService)
		service.snapshots.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "active"}, snapshotmodel.Payload{SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: "https://old.example"}})

		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token", ConfirmServerReplacement: true})
		if !outcome.Success || !outcome.Attempt.ServerMismatchConfirmed {
			t.Fatalf("expected successful confirmed replacement outcome, got %#v", outcome)
		}
		if wroteRequest.SnapshotID != "snapshot-1" {
			t.Fatalf("expected existing snapshot identifier reuse, got %#v", wroteRequest)
		}
	})

	t.Run("write incompatible stored data error", func(t *testing.T) {
		store := runtimeSnapshotStore{write: func(context.Context, snapshotstore.WriteRequest) (snapshotstore.Candidate, error) {
			return snapshotstore.Candidate{}, snapshotstore.ErrIncompatibleStoredData
		}}
		service, config := runtimeServiceWithHistoryServer(t, store, true, `{"activities":[],"count":0}`)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected incompatible stored data failure, got %#v", outcome)
		}
	})

	t.Run("write generic error", func(t *testing.T) {
		store := runtimeSnapshotStore{write: func(context.Context, snapshotstore.WriteRequest) (snapshotstore.Candidate, error) {
			return snapshotstore.Candidate{}, errors.New("write boom")
		}}
		service, config := runtimeServiceWithHistoryServer(t, store, true, `{"activities":[],"count":0}`)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected generic write failure, got %#v", outcome)
		}
	})
}

// TestRunCoversMappingNormalizationAndValidationFailures verifies the
// remaining unsupported-history failure branches inside the runtime sync path.
// Authored by: OpenCode
func TestRunCoversMappingNormalizationAndValidationFailures(t *testing.T) {
	t.Parallel()

	t.Run("mapping failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Path {
			case "/api/v1/auth/anonymous":
				_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
			case "/api/v1/activities":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"},"quantity":1,"unitPriceInAssetProfileCurrency":1,"value":1,"feeInBaseCurrency":0}],"count":1}`))
			default:
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		config := runtimeSetupConfigFixture(t, server.URL, true)
		service := NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), true, runtimeFailingDecimalService{}, syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), runtimeSnapshotStore{}).(*syncService)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedActivityHistory || !outcome.Diagnostic.Eligible {
			t.Fatalf("expected unsupported-history mapping failure, got %#v", outcome)
		}
	})

	t.Run("normalization failure", func(t *testing.T) {
		service, config := runtimeServiceWithHistoryServer(t, runtimeSnapshotStore{}, true, `{"activities":[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"},"quantity":1,"unitPriceInAssetProfileCurrency":1,"value":1,"feeInBaseCurrency":0},{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"},"quantity":2,"unitPriceInAssetProfileCurrency":1,"value":2,"feeInBaseCurrency":0}],"count":2}`)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedActivityHistory || !outcome.Diagnostic.Eligible {
			t.Fatalf("expected unsupported-history normalization failure, got %#v", outcome)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		service, config := runtimeServiceWithHistoryServer(t, runtimeSnapshotStore{}, true, `{"activities":[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"},"quantity":1,"unitPriceInAssetProfileCurrency":0,"value":1,"feeInBaseCurrency":0}],"count":1}`)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedActivityHistory || !outcome.Diagnostic.Eligible {
			t.Fatalf("expected unsupported-history validation failure, got %#v", outcome)
		}
	})

	t.Run("write unsupported stored-data version error", func(t *testing.T) {
		store := runtimeSnapshotStore{write: func(context.Context, snapshotstore.WriteRequest) (snapshotstore.Candidate, error) {
			return snapshotstore.Candidate{}, snapshotstore.ErrUnsupportedStoredDataVersion
		}}
		service, config := runtimeServiceWithHistoryServer(t, store, true, `{"activities":[],"count":0}`)
		outcome := service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected incompatible-new-sync-data outcome, got %#v", outcome)
		}
	})
}

// runtimeDiagnosticRequestFixture returns one structured diagnostic-report
// request for runtime internal tests.
// Authored by: OpenCode
func runtimeDiagnosticRequestFixture() DiagnosticReportRequest {
	return DiagnosticReportRequest{
		FailureReason: SyncFailureUnsupportedActivityHistory,
		ServerOrigin:  "https://ghostfol.io",
		Attempt: SyncAttempt{
			AttemptID:   "attempt-1",
			Status:      AttemptStatusFailed,
			StartedAt:   time.Unix(1, 0).UTC(),
			CompletedAt: time.Unix(2, 0).UTC(),
		},
		Context: syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageValidation,
			FailureDetail: "token detail",
			Records: []syncmodel.DiagnosticRecord{{
				SourceID:   "activity-1",
				Quantity:   "1",
				UnitPrice:  "2",
				GrossValue: "3",
				FeeAmount:  "4",
			}},
		},
	}
}

// runtimeCacheFixture returns one protected activity cache fixture for runtime
// internal tests.
// Authored by: OpenCode
func runtimeCacheFixture() syncmodel.ProtectedActivityCache {
	return syncmodel.ProtectedActivityCache{
		SyncedAt:             time.Unix(10, 0).UTC(),
		RetrievedCount:       0,
		ActivityCount:        0,
		AvailableReportYears: []int{},
		Activities:           []syncmodel.ActivityRecord{},
	}
}

// runtimeSetupConfigFixture returns one valid setup configuration for runtime
// internal tests.
// Authored by: OpenCode
func runtimeSetupConfigFixture(t *testing.T, serverOrigin string, allowDevHTTP bool) configmodel.AppSetupConfig {
	t.Helper()

	serverMode := configmodel.ServerModeCustomOrigin
	if serverOrigin == configmodel.GhostfolioCloudOrigin {
		serverMode = configmodel.ServerModeGhostfolioCloud
	}

	config, err := configmodel.NewSetupConfig(serverMode, serverOrigin, allowDevHTTP, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	return config
}

// runtimeSnapshotCandidateFixture returns one server-scoped snapshot candidate
// for runtime internal tests.
// Authored by: OpenCode
func runtimeSnapshotCandidateFixture(serverOrigin string, snapshotID string) snapshotstore.Candidate {
	return snapshotstore.Candidate{
		SnapshotID: snapshotID,
		Header: snapshotmodel.EnvelopeHeader{
			FormatVersion:      snapshotmodel.EnvelopeFormatVersion,
			ServerDiscoveryKey: snapshotenvelope.DeriveServerDiscoveryKey(serverOrigin),
		},
	}
}

// runtimeServiceWithHistoryServer returns one sync service backed by a test
// Ghostfolio server and matching setup configuration.
// Authored by: OpenCode
func runtimeServiceWithHistoryServer(
	t *testing.T,
	store snapshotstore.Store,
	allowDevHTTP bool,
	activitiesResponse string,
) (*syncService, configmodel.AppSetupConfig) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/activities":
			_, _ = writer.Write([]byte(activitiesResponse))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	service := NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), allowDevHTTP, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), store).(*syncService)
	return service, runtimeSetupConfigFixture(t, server.URL, allowDevHTTP)
}
