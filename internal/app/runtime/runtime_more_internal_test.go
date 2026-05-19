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

// runtimeNormalizerFunc adapts one test function to the runtime normalization
// contract.
// Authored by: OpenCode
type runtimeNormalizerFunc func([]syncmodel.ActivityRecord) (syncmodel.ProtectedActivityCache, error)

// Normalize returns the injected normalized cache or error.
// Authored by: OpenCode
func (f runtimeNormalizerFunc) Normalize(records []syncmodel.ActivityRecord) (syncmodel.ProtectedActivityCache, error) {
	return f(records)
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

// TestBuildDiagnosticReportDocumentPreservesCurrencyContextWhileRedactingFinancialValues
// verifies BUG-003 currency diagnostics survive production redaction.
// Authored by: OpenCode
func TestBuildDiagnosticReportDocumentPreservesCurrencyContextWhileRedactingFinancialValues(t *testing.T) {
	t.Parallel()

	request := runtimeDiagnosticRequestFixture()
	request.Context = syncmodel.DiagnosticContext{
		FailureStage:  syncmodel.DiagnosticFailureStageValidation,
		FailureDetail: `activity "buy-1" unit price currency context is uninformed across order, asset-profile, and base tiers`,
		Records: []syncmodel.DiagnosticRecord{{
			SourceID:              "buy-1",
			OrderCurrency:         "CHF",
			AssetProfileCurrency:  "EUR",
			BaseCurrency:          "USD",
			UnitPrice:             "95",
			UnitPriceCurrency:     "EUR",
			GrossValue:            "90",
			GrossValueCurrency:    "CHF",
			FeeAmount:             "1.8",
			FeeAmountCurrency:     "EUR",
			OrderUnitPrice:        "",
			AssetProfileUnitPrice: "95",
			OrderGrossValue:       "90",
		}},
	}

	request.RedactFinancialValues = true
	redacted := buildDiagnosticReportDocument(request, time.Unix(3, 0).UTC())
	if !redacted.FinancialValuesRedacted {
		t.Fatalf("expected production diagnostic report to mark financial values as redacted")
	}
	if len(redacted.Records) != 1 {
		t.Fatalf("expected one redacted diagnostic record, got %#v", redacted)
	}
	redactedRecord := redacted.Records[0]
	if redactedRecord.OrderCurrency != "CHF" || redactedRecord.AssetProfileCurrency != "EUR" || redactedRecord.BaseCurrency != "USD" || redactedRecord.UnitPriceCurrency != "EUR" || redactedRecord.FeeAmountCurrency != "EUR" {
		t.Fatalf("expected currency context to remain visible after redaction, got %#v", redactedRecord)
	}
	if redactedRecord.UnitPrice != "" || redactedRecord.GrossValue != "" || redactedRecord.FeeAmount != "" || redactedRecord.AssetProfileUnitPrice != "" {
		t.Fatalf("expected financial values to be cleared, got %#v", redactedRecord)
	}

	request.RedactFinancialValues = false
	unredacted := buildDiagnosticReportDocument(request, time.Unix(3, 0).UTC())
	unredactedRecord := unredacted.Records[0]
	if unredactedRecord.UnitPrice != "95" || unredactedRecord.GrossValue != "90" || unredactedRecord.FeeAmount != "1.8" || unredactedRecord.AssetProfileUnitPrice != "95" {
		t.Fatalf("expected explicit-development-mode diagnostics to retain financial values, got %#v", unredactedRecord)
	}
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

	var service = requireSyncService(t, NewSyncService(nil, time.Second, "", true, nil, nil, nil, runtimeSnapshotStore{}))
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
	service := requireSyncService(t, NewSyncService(nil, time.Second, "/tmp/config", true, nil, nil, nil, runtimeSnapshotStore{}))
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

	payload, err := (protectedPayloadBuilder{}).Build(protectedPayloadBuildRequest{Config: config, Cache: cache})
	if err != nil {
		t.Fatalf("build protected payload: %v", err)
	}
	if payload.RegisteredLocalUser.LocalUserID == "" {
		t.Fatalf("expected generated local user id")
	}

	existing := snapshotmodel.Payload{RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{LocalUserID: "user-1", CreatedAt: time.Unix(1, 0).UTC()}}
	payload, err = (protectedPayloadBuilder{}).Build(protectedPayloadBuildRequest{Config: config, Cache: cache, ExistingPayload: existing, HasExisting: true})
	if err != nil {
		t.Fatalf("build existing protected payload: %v", err)
	}
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

	if _, err := (protectedPayloadBuilder{}).Build(protectedPayloadBuildRequest{Config: config, Cache: cache}); err == nil {
		t.Fatalf("expected protected payload build failure when identifier generation fails")
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

		outcome := service.finalizeSyncFailure(context.Background(), session, attempt, SyncFailureTimeout, syncmodel.DiagnosticContext{})
		if outcome.Diagnostic.Eligible {
			t.Fatalf("expected timeout failure to remain diagnostic ineligible")
		}
		if session.SecurityToken != "" || session.AuthToken != "" {
			t.Fatalf("expected secrets to be cleared after finalizeSyncFailure")
		}
	})

	t.Run("eligible failure in production mode", func(t *testing.T) {
		service := &syncService{allowDevHTTP: false}
		outcome := service.finalizeSyncFailure(context.Background(), &GhostfolioSession{ServerOrigin: "https://ghostfol.io"}, &SyncAttempt{}, SyncFailureUnsupportedActivityHistory, syncmodel.DiagnosticContext{})
		if !outcome.Diagnostic.Eligible || outcome.Diagnostic.Path != "" {
			t.Fatalf("expected manual diagnostic eligibility, got %#v", outcome.Diagnostic)
		}
		if !outcome.Diagnostic.Request.RedactFinancialValues || outcome.Diagnostic.Request.ExplicitDevelopmentMode {
			t.Fatalf("expected production-mode redaction request, got %#v", outcome.Diagnostic.Request)
		}
	})

	t.Run("eligible failure in development mode writes report", func(t *testing.T) {
		service := &syncService{allowDevHTTP: true, diagnosticReports: newDiagnosticReportService(t.TempDir())}
		outcome := service.finalizeSyncFailure(context.Background(), &GhostfolioSession{ServerOrigin: "https://ghostfol.io"}, &SyncAttempt{}, SyncFailureIncompatibleNewSyncData, runtimeDiagnosticRequestFixture().Context)
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
		outcome := service.finalizeSyncFailure(context.Background(), &GhostfolioSession{ServerOrigin: "https://ghostfol.io"}, &SyncAttempt{}, SyncFailureUnsupportedStoredDataVersion, runtimeDiagnosticRequestFixture().Context)
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
		service := requireSyncService(t, NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, nil))
		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, "https://ghostfol.io", true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected unavailable snapshot store failure, got %#v", outcome)
		}
	})

	t.Run("server replacement cancelled", func(t *testing.T) {
		var payload = snapshotmodel.Payload{SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: "https://old.example"}}
		service := requireSyncService(t, NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, runtimeSnapshotStore{}))
		service.snapshots.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "snapshot-1"}, payload)

		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, "https://new.example", true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureServerReplacementCancelled || outcome.Attempt.Status != AttemptStatusFailed {
			t.Fatalf("expected replacement cancellation failure, got %#v", outcome)
		}
	})

	t.Run("candidate discovery error", func(t *testing.T) {
		service := requireSyncService(t, NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, runtimeSnapshotStore{candidatesErr: errors.New("discover boom")}))
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
		service := requireSyncService(t, NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, store))
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
		service := requireSyncService(t, NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, store))
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
		service := requireSyncService(t, NewSyncService(nil, time.Second, t.TempDir(), true, nil, nil, nil, store))
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
			case "/api/v1/user":
				_, _ = writer.Write([]byte(`{"settings":{"baseCurrency":"USD"}}`))
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
		service := requireSyncService(t, NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), store))
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
			case "/api/v1/user":
				_, _ = writer.Write([]byte(`{"settings":{"baseCurrency":"USD"}}`))
			case "/api/v1/activities":
				_, _ = writer.Write([]byte(`{"activities":[{"id":"activity-1","date":"2024-01-01T10:00:00Z","type":"BUY","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"},"quantity":1,"unitPriceInAssetProfileCurrency":1,"value":1,"feeInBaseCurrency":0}],"count":1}`))
			default:
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		config := runtimeSetupConfigFixture(t, server.URL, true)
		service := requireSyncService(t, NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), true, runtimeFailingDecimalService{}, syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), runtimeSnapshotStore{}))
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
		if outcome.FailureReason != SyncFailureUnsupportedStoredDataVersion {
			t.Fatalf("expected unsupported stored-data version persistence outcome, got %#v", outcome)
		}
	})
}

// TestRunPreservesIncompleteCurrencyContextDiagnosticContext verifies that validation
// failures caused by incomplete explicit currency identity keep offending-record
// context available for production and explicit-development diagnostic reports.
// Authored by: OpenCode
func TestRunPreservesIncompleteCurrencyContextDiagnosticContext(t *testing.T) {
	t.Parallel()

	var expectedDetail = `activity "buy-1" gross value currency context is uninformed across order, asset-profile, and base tiers`

	t.Run("production mode redacts persisted financial values", func(t *testing.T) {
		var service, config = runtimeServiceWithHistoryServer(t, runtimeSnapshotStore{}, false, `{"activities":[],"count":0}`)
		service.normalizer = runtimeNormalizerFunc(func([]syncmodel.ActivityRecord) (syncmodel.ProtectedActivityCache, error) {
			return runtimeCurrencyMismatchCacheFixture(t), nil
		})

		var outcome = service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedActivityHistory || !outcome.Diagnostic.Eligible {
			t.Fatalf("expected diagnostic-eligible unsupported-history failure, got %#v", outcome)
		}
		if outcome.Diagnostic.Path != "" {
			t.Fatalf("expected production mode to defer report creation, got %#v", outcome.Diagnostic)
		}
		if !outcome.Diagnostic.Request.RedactFinancialValues || outcome.Diagnostic.Request.ExplicitDevelopmentMode {
			t.Fatalf("expected production-mode diagnostic request flags, got %#v", outcome.Diagnostic.Request)
		}

		var diagnosticContext = outcome.Diagnostic.Request.Context
		if diagnosticContext.FailureStage != syncmodel.DiagnosticFailureStageValidation || diagnosticContext.FailureDetail != expectedDetail {
			t.Fatalf("expected validation-stage currency mismatch detail, got %#v", diagnosticContext)
		}
		if len(diagnosticContext.Records) != 1 {
			t.Fatalf("expected one diagnostic record, got %#v", diagnosticContext)
		}
		var requestRecord = diagnosticContext.Records[0]
		if requestRecord.OrderCurrency != "" || requestRecord.AssetProfileCurrency != "" || requestRecord.BaseCurrency != "" || requestRecord.UnitPriceCurrency != "" || requestRecord.GrossValueCurrency != "" {
			t.Fatalf("expected preserved currency identifiers in diagnostic request, got %#v", requestRecord)
		}
		if requestRecord.UnitPrice != "" || requestRecord.OrderUnitPrice != "" || requestRecord.AssetProfileUnitPrice != "95" || requestRecord.BaseGrossValue != "100" || requestRecord.OrderGrossValue != "90" {
			t.Fatalf("expected unredacted offending-record values in diagnostic request, got %#v", requestRecord)
		}

		var reportPath, err = service.GenerateDiagnosticReport(context.Background(), outcome.Diagnostic.Request)
		if err != nil {
			t.Fatalf("generate production diagnostic report: %v", err)
		}
		var document = mustRuntimeDiagnosticReportDocument(t, reportPath)
		if !document.FinancialValuesRedacted || document.ExplicitDevelopmentMode {
			t.Fatalf("expected production diagnostic redaction metadata, got %#v", document)
		}
		if document.FailureStage != syncmodel.DiagnosticFailureStageValidation || document.FailureDetail != expectedDetail {
			t.Fatalf("expected production diagnostic failure detail to be preserved, got %#v", document)
		}
		if len(document.Records) != 1 {
			t.Fatalf("expected one persisted diagnostic record, got %#v", document)
		}
		var persistedRecord = document.Records[0]
		if persistedRecord.OrderCurrency != "" || persistedRecord.AssetProfileCurrency != "" || persistedRecord.BaseCurrency != "" || persistedRecord.UnitPriceCurrency != "" || persistedRecord.GrossValueCurrency != "" {
			t.Fatalf("expected production diagnostic report to preserve currency identifiers, got %#v", persistedRecord)
		}
		if persistedRecord.UnitPrice != "" || persistedRecord.AssetProfileUnitPrice != "" || persistedRecord.BaseGrossValue != "" || persistedRecord.OrderGrossValue != "" || persistedRecord.FeeAmount != "" {
			t.Fatalf("expected production diagnostic report to redact financial values, got %#v", persistedRecord)
		}
	})

	t.Run("development mode keeps offending-record values", func(t *testing.T) {
		var service, config = runtimeServiceWithHistoryServer(t, runtimeSnapshotStore{}, true, `{"activities":[],"count":0}`)
		service.normalizer = runtimeNormalizerFunc(func([]syncmodel.ActivityRecord) (syncmodel.ProtectedActivityCache, error) {
			return runtimeCurrencyMismatchCacheFixture(t), nil
		})

		var outcome = service.Run(context.Background(), SyncRequest{Config: config, SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsupportedActivityHistory || !outcome.Diagnostic.Eligible {
			t.Fatalf("expected diagnostic-eligible unsupported-history failure, got %#v", outcome)
		}
		if outcome.Diagnostic.Path == "" {
			t.Fatalf("expected explicit development mode to auto-write the diagnostic report, got %#v", outcome.Diagnostic)
		}
		if outcome.Diagnostic.Request.RedactFinancialValues || !outcome.Diagnostic.Request.ExplicitDevelopmentMode {
			t.Fatalf("expected explicit-development diagnostic request flags, got %#v", outcome.Diagnostic.Request)
		}

		var document = mustRuntimeDiagnosticReportDocument(t, outcome.Diagnostic.Path)
		if document.FinancialValuesRedacted || !document.ExplicitDevelopmentMode {
			t.Fatalf("expected development diagnostic metadata without redaction, got %#v", document)
		}
		if document.FailureStage != syncmodel.DiagnosticFailureStageValidation || document.FailureDetail != expectedDetail {
			t.Fatalf("expected development diagnostic failure detail to be preserved, got %#v", document)
		}
		if len(document.Records) != 1 {
			t.Fatalf("expected one persisted diagnostic record, got %#v", document)
		}
		var persistedRecord = document.Records[0]
		if persistedRecord.OrderCurrency != "" || persistedRecord.AssetProfileCurrency != "" || persistedRecord.BaseCurrency != "" || persistedRecord.UnitPriceCurrency != "" || persistedRecord.GrossValueCurrency != "" {
			t.Fatalf("expected development diagnostic report to preserve currency identifiers, got %#v", persistedRecord)
		}
		if persistedRecord.UnitPrice != "" || persistedRecord.OrderUnitPrice != "" || persistedRecord.OrderGrossValue != "90" || persistedRecord.AssetProfileUnitPrice != "95" || persistedRecord.BaseGrossValue != "100" {
			t.Fatalf("expected development diagnostic report to preserve offending-record values, got %#v", persistedRecord)
		}
	})
}

// TestSnapshotLifecycleAndWrapperNilBranches verifies the remaining nil-guard
// and payload-build branches around protected snapshot state helpers.
// Authored by: OpenCode
func TestSnapshotLifecycleAndWrapperNilBranches(t *testing.T) {
	t.Parallel()

	var nilActiveState *activeSnapshotState
	nilActiveState.Set(snapshotstore.Candidate{SnapshotID: "ignored"}, snapshotmodel.Payload{})
	if state := nilActiveState.ProtectedDataState(); state != (ProtectedDataState{}) {
		t.Fatalf("expected nil active snapshot state to report zero value, got %#v", state)
	}

	lifecycle := newSnapshotLifecycle(runtimeSnapshotStore{}, nil, protectedPayloadBuilder{})
	if lifecycle == nil || lifecycle.state == nil {
		t.Fatalf("expected new snapshot lifecycle to create default state, got %#v", lifecycle)
	}
	lifecycle.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "active"}, snapshotmodel.Payload{SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: "https://ghostfol.io"}})
	if state := lifecycle.ProtectedDataState(); !state.HasReadableSnapshot || state.ServerOrigin != "https://ghostfol.io" {
		t.Fatalf("expected non-nil snapshot lifecycle state, got %#v", state)
	}

	var nilLifecycle *snapshotLifecycle
	if state := nilLifecycle.ProtectedDataState(); state != (ProtectedDataState{}) {
		t.Fatalf("expected nil snapshot lifecycle to report zero protected data state, got %#v", state)
	}
	if check := nilLifecycle.CheckServerReplacement(runtimeSetupConfigFixture(t, "https://ghostfol.io", true)); check != (ServerReplacementCheck{}) {
		t.Fatalf("expected nil snapshot lifecycle to report no replacement requirement, got %#v", check)
	}
	nilLifecycle.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "ignored"}, snapshotmodel.Payload{})

	err := (&snapshotLifecycle{}).Persist(context.Background(), snapshotPersistRequest{})
	if !errors.Is(err, errSnapshotStoreUnavailable) {
		t.Fatalf("expected missing store to fail persist, got %v", err)
	}

	originalReadRandom := readRandom
	readRandom = func([]byte) (int, error) {
		return 0, errors.New("random boom")
	}
	defer func() {
		readRandom = originalReadRandom
	}()

	err = lifecycle.Persist(context.Background(), snapshotPersistRequest{
		Config:        runtimeSetupConfigFixture(t, "https://ghostfol.io", true),
		SecurityToken: "token",
		Cache:         runtimeCacheFixture(),
	})
	if err == nil || !strings.Contains(err.Error(), "build protected payload local user id") {
		t.Fatalf("expected payload build failure from random identifier error, got %v", err)
	}
}

// TestSyncServiceUserFetchAndSnapshotWrapperBranches verifies the remaining
// sync-service wrapper and user-fetch failure branches.
// Authored by: OpenCode
func TestSyncServiceUserFetchAndSnapshotWrapperBranches(t *testing.T) {
	t.Parallel()

	t.Run("wrapper methods handle nil snapshots", func(t *testing.T) {
		config := runtimeSetupConfigFixture(t, "https://ghostfol.io", true)
		service := &syncService{}
		lifecycle := newSnapshotLifecycle(runtimeSnapshotStore{}, newActiveSnapshotState(), protectedPayloadBuilder{})
		lifecycle.SetActiveSnapshot(snapshotstore.Candidate{SnapshotID: "active"}, snapshotmodel.Payload{SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: "https://ghostfol.io"}})
		if state := service.ProtectedDataState(); state != (ProtectedDataState{}) {
			t.Fatalf("expected zero protected data state when snapshots are absent, got %#v", state)
		}
		if check := service.CheckServerReplacement(config); check != (ServerReplacementCheck{}) {
			t.Fatalf("expected zero server replacement check when snapshots are absent, got %#v", check)
		}

		session, attempt := newSyncAttemptState(SyncRequest{Config: config, SecurityToken: "token"})
		_, outcome, ok := service.prepareSyncAttempt(context.Background(), SyncRequest{Config: config, SecurityToken: "token"}, &session, &attempt)
		if ok || outcome.FailureReason != SyncFailureIncompatibleNewSyncData {
			t.Fatalf("expected prepareSyncAttempt to fail without snapshot lifecycle, got ok=%v outcome=%#v", ok, outcome)
		}

		service.snapshots = lifecycle
		if state := service.ProtectedDataState(); !state.HasReadableSnapshot || state.ServerOrigin != "https://ghostfol.io" {
			t.Fatalf("expected sync-service wrapper to delegate protected data state, got %#v", state)
		}
	})

	t.Run("run handles authenticated user request failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Path {
			case "/api/v1/auth/anonymous":
				_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
			case "/api/v1/user":
				writer.WriteHeader(http.StatusUnauthorized)
			default:
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		service := requireSyncService(t, NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), runtimeSnapshotStore{}))
		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, server.URL, true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureUnsuccessfulServerResponse {
			t.Fatalf("expected user-request failure to map to unsuccessful response, got %#v", outcome)
		}
	})

	t.Run("run handles authenticated user contract failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			switch request.URL.Path {
			case "/api/v1/auth/anonymous":
				_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
			case "/api/v1/user":
				_, _ = writer.Write([]byte(`{"settings":null}`))
			default:
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		service := requireSyncService(t, NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), true, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), runtimeSnapshotStore{}))
		outcome := service.Run(context.Background(), SyncRequest{Config: runtimeSetupConfigFixture(t, server.URL, true), SecurityToken: "token"})
		if outcome.FailureReason != SyncFailureIncompatibleServerContract {
			t.Fatalf("expected user-response contract failure, got %#v", outcome)
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

// runtimeCurrencyMismatchCacheFixture returns one normalized cache whose
// preserved gross-value basis stays uninformed across all tracked currency tiers.
// Authored by: OpenCode
func runtimeCurrencyMismatchCacheFixture(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	return syncmodel.ProtectedActivityCache{
		ActivityCount: 1,
		Activities:    []syncmodel.ActivityRecord{runtimeCurrencyMismatchRecordFixture(t)},
	}
}

// runtimeCurrencyMismatchRecordFixture returns one normalized activity record
// whose preserved gross-value basis stays uninformed across all tracked tiers.
// Authored by: OpenCode
func runtimeCurrencyMismatchRecordFixture(t *testing.T) syncmodel.ActivityRecord {
	t.Helper()

	var quantity, _, err = decimalsupport.ParseString("1")
	if err != nil {
		t.Fatalf("parse quantity: %v", err)
	}
	var grossValue apd.Decimal
	grossValue, _, err = decimalsupport.ParseString("90")
	if err != nil {
		t.Fatalf("parse gross value: %v", err)
	}
	var assetProfileUnitPrice apd.Decimal
	assetProfileUnitPrice, _, err = decimalsupport.ParseString("95")
	if err != nil {
		t.Fatalf("parse asset-profile unit price: %v", err)
	}
	var baseGrossValue apd.Decimal
	baseGrossValue, _, err = decimalsupport.ParseString("100")
	if err != nil {
		t.Fatalf("parse base gross value: %v", err)
	}

	return syncmodel.ActivityRecord{
		SourceID:              "buy-1",
		OccurredAt:            "2024-01-01T10:00:00Z",
		ActivityType:          syncmodel.ActivityTypeBuy,
		AssetSymbol:           "BTC",
		AssetName:             "Bitcoin",
		OrderCurrency:         "",
		AssetProfileCurrency:  "",
		BaseCurrency:          "",
		Quantity:              quantity,
		OrderUnitPrice:        nil,
		OrderGrossValue:       &grossValue,
		AssetProfileUnitPrice: &assetProfileUnitPrice,
		BaseGrossValue:        &baseGrossValue,
		RawHash:               "buy-1",
	}
}

// mustRuntimeDiagnosticReportDocument reads one persisted diagnostic report and
// unmarshals it into the local runtime document model.
// Authored by: OpenCode
func mustRuntimeDiagnosticReportDocument(t *testing.T, path string) diagnosticReportDocument {
	t.Helper()

	var raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read diagnostic report: %v", err)
	}
	var document diagnosticReportDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal diagnostic report: %v", err)
	}

	return document
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

	var handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/user":
			_, _ = writer.Write([]byte(`{"settings":{"baseCurrency":"USD"}}`))
		case "/api/v1/activities":
			_, _ = writer.Write([]byte(activitiesResponse))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	})

	var server *httptest.Server
	if allowDevHTTP {
		server = httptest.NewServer(handler)
	} else {
		server = httptest.NewTLSServer(handler)
	}
	t.Cleanup(server.Close)

	service := requireSyncService(t, NewSyncService(ghostfolioclient.New(server.Client()), time.Second, t.TempDir(), allowDevHTTP, decimalsupport.NewService(), syncnormalize.NewNormalizer(), syncvalidate.NewValidator(), store))
	return service, runtimeSetupConfigFixture(t, server.URL, allowDevHTTP)
}

// requireSyncService converts the public SyncService interface to the concrete
// runtime service type for internal tests.
// Authored by: OpenCode
func requireSyncService(t *testing.T, service SyncService) *syncService {
	t.Helper()

	concrete, ok := service.(*syncService)
	if !ok {
		t.Fatalf("expected *syncService, got %T", service)
	}

	return concrete
}
