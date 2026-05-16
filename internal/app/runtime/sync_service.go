// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	ghostfoliodto "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
	ghostfoliomapper "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/mapper"
	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/validator"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/internal/support/redact"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

// Test seams wrap secure random reads so runtime tests can exercise identifier
// generation failures safely.
// Authored by: OpenCode
var readRandom = rand.Read

// Test seams wrap envelope compatibility checks so runtime tests can inject
// stored-data validation failures safely.
// Authored by: OpenCode
var validateEnvelopeHeaderCompatibility = snapshotstore.ValidateEnvelopeCompatibility

// ValidateRequest contains the bootstrap configuration and runtime-only token
// needed for a single Ghostfolio communication check.
//
// Authored by: OpenCode
type ValidateRequest struct {
	Config                   configmodel.AppSetupConfig
	SecurityToken            string
	ConfirmServerReplacement bool
}

// ProtectedDataState summarizes whether readable protected data is active in memory for this run.
// Authored by: OpenCode
type ProtectedDataState struct {
	HasReadableSnapshot bool
	ServerOrigin        string
}

// ServerReplacementCheck reports whether the selected server would replace the active readable snapshot.
// Authored by: OpenCode
type ServerReplacementCheck struct {
	Required             bool
	ActiveServerOrigin   string
	SelectedServerOrigin string
}

// SyncService validates Ghostfolio communication for the currently selected
// bootstrap setup.
//
// Authored by: OpenCode
type SyncService interface {
	// Validate runs the anonymous-auth and one-page activities probe for the given
	// setup and returns a structured semantic result. Successful outcomes never
	// persist tokens or Ghostfolio payload data. Failed outcomes classify the
	// attempt into one supported failure reason without exposing transport-layer
	// implementation types to callers.
	//
	// Authored by: OpenCode
	Validate(context.Context, ValidateRequest) ValidationOutcome

	// GenerateDiagnosticReport writes one local synced-data diagnostic report for an eligible failure.
	// Authored by: OpenCode
	GenerateDiagnosticReport(context.Context, DiagnosticReportRequest) (string, error)

	// ProtectedDataState reports whether a readable protected snapshot is active for this run.
	// Authored by: OpenCode
	ProtectedDataState() ProtectedDataState

	// CheckServerReplacement compares the selected setup server with the active readable snapshot.
	// Authored by: OpenCode
	CheckServerReplacement(configmodel.AppSetupConfig) ServerReplacementCheck
}

// syncService coordinates one validation attempt across the Ghostfolio client
// boundary and response validators.
// Authored by: OpenCode
type syncService struct {
	client         *ghostfolioclient.Client
	requestTimeout time.Duration
	baseConfigDir  string
	allowDevHTTP   bool
	decimalService decimalsupport.Service
	normalizer     syncnormalize.Normalizer
	validator      syncvalidate.Validator
	snapshotStore  snapshotstore.Store
	activeMutex    sync.Mutex
	activeSnapshot activeReadableSnapshot
}

// activeReadableSnapshot stores the current run's successfully unlocked local
// protected snapshot.
// Authored by: OpenCode
type activeReadableSnapshot struct {
	Candidate snapshotstore.Candidate
	Payload   snapshotmodel.Payload
	Present   bool
}

// NewSyncService creates the runtime sync-validation service.
//
// Example:
//
//	service := runtime.NewSyncService(ghostfolioclient.New(nil), 30*time.Second, decimal.NewService(), normalize.NewNormalizer(), validate.NewValidator(), snapshots)
//	_ = service
//
// The returned service depends on the Ghostfolio HTTP client boundary and
// enforces a per-attempt timeout for the full validation workflow.
// Authored by: OpenCode
func NewSyncService(
	client *ghostfolioclient.Client,
	requestTimeout time.Duration,
	baseConfigDir string,
	allowDevHTTP bool,
	decimalService decimalsupport.Service,
	normalizer syncnormalize.Normalizer,
	validatorService syncvalidate.Validator,
	snapshots snapshotstore.Store,
) SyncService {
	if client == nil {
		client = ghostfolioclient.New(nil)
	}
	if decimalService == nil {
		decimalService = decimalsupport.NewService()
	}
	if normalizer == nil {
		normalizer = syncnormalize.NewNormalizer()
	}
	if validatorService == nil {
		validatorService = syncvalidate.NewValidator()
	}

	return &syncService{
		client:         client,
		requestTimeout: requestTimeout,
		baseConfigDir:  baseConfigDir,
		allowDevHTTP:   allowDevHTTP,
		decimalService: decimalService,
		normalizer:     normalizer,
		validator:      validatorService,
		snapshotStore:  snapshots,
	}
}

// Validate executes one full Ghostfolio communication-validation attempt.
//
// Example:
//
//	outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token"})
//	_ = outcome.Success
//
// Validate authenticates anonymously against the configured origin, retrieves
// the full activity history, normalizes and validates the supported data, and
// stores it as a protected snapshot.
// Authored by: OpenCode
func (s *syncService) Validate(ctx context.Context, request ValidateRequest) ValidationOutcome {
	var timedContext, cancel = context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	var now = time.Now().UTC()
	var session = GhostfolioSession{
		ServerOrigin:  request.Config.ServerOrigin,
		SecurityToken: request.SecurityToken,
		StartedAt:     now,
	}
	var attempt = SyncValidationAttempt{
		AttemptID: fmt.Sprintf("attempt-%d", now.UnixNano()),
		Status:    AttemptStatusDiscoveringSnapshot,
		StartedAt: now,
	}

	var unlockedCandidate snapshotstore.Candidate
	var unlockedPayload snapshotmodel.Payload
	var unlocked bool
	var err error
	var diagnosticContext syncmodel.DiagnosticContext

	if s.snapshotStore == nil {
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageProtectedPersistence,
			FailureDetail: "protected snapshot store is unavailable",
		})
	}

	var replacementCheck = s.CheckServerReplacement(request.Config)
	if replacementCheck.Required && !request.ConfirmServerReplacement {
		attempt.Status = AttemptStatusAborted
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureServerReplacementCancelled, syncmodel.DiagnosticContext{})
	}
	if replacementCheck.Required {
		attempt.ServerMismatchConfirmed = true
	}

	var candidates []snapshotstore.Candidate
	candidates, err = snapshotstore.DiscoverServerCandidates(timedContext, s.snapshotStore, request.Config.ServerOrigin)
	if err != nil {
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageStoredDataCompatibility,
			FailureDetail: redact.ErrorText(err, request.SecurityToken),
		})
	}
	for _, candidate := range candidates {
		if err := validateEnvelopeHeaderCompatibility(candidate.Header); err != nil {
			if errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
				return s.finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedStoredDataVersion, syncmodel.DiagnosticContext{
					FailureStage:  syncmodel.DiagnosticFailureStageStoredDataCompatibility,
					FailureDetail: redact.ErrorText(err, request.SecurityToken),
				})
			}
			return s.finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
				FailureStage:  syncmodel.DiagnosticFailureStageStoredDataCompatibility,
				FailureDetail: redact.ErrorText(err, request.SecurityToken),
			})
		}
	}

	attempt.Status = AttemptStatusUnlockingSnapshot
	for _, candidate := range candidates {
		unlockedPayload, err = s.snapshotStore.Read(timedContext, snapshotstore.ReadRequest{
			Candidate:     candidate,
			SecurityToken: request.SecurityToken,
		})
		if err == nil {
			unlockedCandidate = candidate
			unlocked = true
			break
		}
		if errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
			return s.finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedStoredDataVersion, syncmodel.DiagnosticContext{
				FailureStage:  syncmodel.DiagnosticFailureStageStoredDataCompatibility,
				FailureDetail: redact.ErrorText(err, request.SecurityToken),
			})
		}
	}

	attempt.Status = AttemptStatusAuthenticating

	var authResponse = ghostfoliodto.AuthResponse{}
	authResponse, err = s.client.Authenticate(timedContext, session.ServerOrigin, session.SecurityToken)
	if err != nil {
		return finalizeFailure(&session, &attempt, err)
	}
	if err := validator.ValidateAuthResponse(authResponse); err != nil {
		return finalizeValidationFailure(&session, &attempt, err)
	}

	session.AuthToken = authResponse.AuthToken
	session.AuthenticatedAt = time.Now().UTC()
	attempt.Status = AttemptStatusRetrievingHistory

	historyResponse, err := s.client.FetchActivitiesHistory(timedContext, session.ServerOrigin, session.AuthToken)
	if err != nil {
		return finalizeFailure(&session, &attempt, err)
	}

	if err := validator.ValidateActivityPageResponse(historyResponse); err != nil {
		return finalizeValidationFailure(&session, &attempt, err)
	}

	attempt.Status = AttemptStatusNormalizing
	records, err := ghostfoliomapper.MapActivities(historyResponse.Activities, s.decimalService)
	if err != nil {
		diagnosticContext = diagnosticContextFromError(err, syncmodel.DiagnosticFailureStageMapping, request.SecurityToken)
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedActivityHistory, diagnosticContext)
	}

	cache, err := s.normalizer.Normalize(records)
	if err != nil {
		diagnosticContext = diagnosticContextFromError(err, syncmodel.DiagnosticFailureStageNormalization, request.SecurityToken)
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedActivityHistory, diagnosticContext)
	}
	cache.SyncedAt = time.Now().UTC()

	attempt.Status = AttemptStatusValidating
	if err := s.validator.Validate(cache); err != nil {
		diagnosticContext = diagnosticContextFromError(err, syncmodel.DiagnosticFailureStageValidation, request.SecurityToken)
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedActivityHistory, diagnosticContext)
	}

	attempt.Status = AttemptStatusPersisting
	var payload = newSnapshotPayload(request.Config, cache, unlockedPayload, unlocked)
	var persistedCandidate snapshotstore.Candidate
	persistedCandidate, err = s.snapshotStore.Write(timedContext, snapshotstore.WriteRequest{
		SnapshotID:    unlockedCandidate.SnapshotID,
		SecurityToken: request.SecurityToken,
		ServerOrigin:  request.Config.ServerOrigin,
		Payload:       payload,
	})
	if err != nil {
		if errors.Is(err, snapshotstore.ErrIncompatibleStoredData) || errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
			return s.finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
				FailureStage:  syncmodel.DiagnosticFailureStageProtectedPersistence,
				FailureDetail: redact.ErrorText(err, request.SecurityToken),
				Records:       diagnosticContext.Records,
			})
		}
		return s.finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageProtectedPersistence,
			FailureDetail: redact.ErrorText(err, request.SecurityToken),
			Records:       diagnosticContext.Records,
		})
	}
	s.setActiveSnapshot(persistedCandidate, payload)

	attempt.Status = AttemptStatusSuccess
	attempt.CompletedAt = time.Now().UTC()
	clearSessionSecrets(&session)

	return ValidationOutcome{
		Success:      true,
		DetailReason: "activity_data_stored",
		Attempt:      SyncAttempt(attempt),
	}
}

// GenerateDiagnosticReport writes one local synced-data diagnostic report for an eligible failure.
// Authored by: OpenCode
func (s *syncService) GenerateDiagnosticReport(ctx context.Context, request DiagnosticReportRequest) (string, error) {
	var baseConfigDir, err = resolveBaseConfigDir(s.baseConfigDir)
	if err != nil {
		return "", err
	}

	return writeDiagnosticReport(ctx, baseConfigDir, request)
}

// finalizeFailure converts a boundary failure into a user-visible validation outcome.
// Authored by: OpenCode
func finalizeFailure(session *GhostfolioSession, attempt *SyncValidationAttempt, err error) ValidationOutcome {
	attempt.Status = AttemptStatusFailure
	attempt.CompletedAt = time.Now().UTC()

	var category = ValidationFailureConnectivityProblem
	var requestFailure *ghostfolioclient.RequestFailure
	if errors.As(err, &requestFailure) {
		category = validationFailureReasonFromBoundary(requestFailure.Category)
	}
	attempt.FailureReason = category
	clearSessionSecrets(session)

	return ValidationOutcome{
		Success:       false,
		DetailReason:  string(category),
		FailureReason: category,
		Attempt:       SyncAttempt(*attempt),
	}
}

// finalizeValidationFailure converts a payload validation error into the supported incompatible-server outcome.
// Authored by: OpenCode
func finalizeValidationFailure(
	session *GhostfolioSession,
	attempt *SyncValidationAttempt,
	err error,
) ValidationOutcome {
	_ = err
	attempt.Status = AttemptStatusFailure
	attempt.CompletedAt = time.Now().UTC()
	attempt.FailureReason = ValidationFailureIncompatibleServerContract
	clearSessionSecrets(session)

	return ValidationOutcome{
		Success:       false,
		DetailReason:  string(ValidationFailureIncompatibleServerContract),
		FailureReason: ValidationFailureIncompatibleServerContract,
		Attempt:       SyncAttempt(*attempt),
	}
}

// finalizeSyncFailure converts an internal sync failure into one supported user-visible category.
// Authored by: OpenCode
func (s *syncService) finalizeSyncFailure(
	session *GhostfolioSession,
	attempt *SyncValidationAttempt,
	reason SyncFailureReason,
	diagnosticContext syncmodel.DiagnosticContext,
) ValidationOutcome {
	attempt.Status = AttemptStatusFailure
	attempt.CompletedAt = time.Now().UTC()
	attempt.FailureReason = reason
	clearSessionSecrets(session)

	var outcome = ValidationOutcome{
		Success:       false,
		DetailReason:  string(reason),
		FailureReason: reason,
		Attempt:       SyncAttempt(*attempt),
	}
	if !diagnosticEligible(reason) {
		return outcome
	}

	var request = DiagnosticReportRequest{
		FailureReason:           reason,
		ServerOrigin:            session.ServerOrigin,
		Attempt:                 SyncAttempt(*attempt),
		Context:                 diagnosticContext,
		RedactFinancialValues:   !s.allowDevHTTP,
		ExplicitDevelopmentMode: s.allowDevHTTP,
	}
	outcome.Diagnostic = DiagnosticReportState{
		Eligible: true,
		Request:  request,
	}
	if s.allowDevHTTP {
		path, err := s.GenerateDiagnosticReport(context.Background(), request)
		if err == nil {
			outcome.Diagnostic.Path = path
		}
	}

	return outcome
}

// validationFailureReasonFromBoundary maps transport-boundary failures into the
// application-facing validation failure taxonomy.
// Authored by: OpenCode
func validationFailureReasonFromBoundary(category ghostfolioclient.FailureCategory) ValidationFailureReason {
	switch category {
	case ghostfolioclient.FailureRejectedToken:
		return ValidationFailureRejectedToken
	case ghostfolioclient.FailureTimeout:
		return ValidationFailureTimeout
	case ghostfolioclient.FailureConnectivityProblem:
		return ValidationFailureConnectivityProblem
	case ghostfolioclient.FailureUnsuccessfulServerResponse:
		return ValidationFailureUnsuccessfulServerResponse
	case ghostfolioclient.FailureIncompatibleServerContract:
		return ValidationFailureIncompatibleServerContract
	default:
		return ValidationFailureConnectivityProblem
	}
}

// diagnosticEligible reports whether the failure reason supports synced-data diagnostic reports.
// Authored by: OpenCode
func diagnosticEligible(reason SyncFailureReason) bool {
	switch reason {
	case SyncFailureUnsupportedActivityHistory, SyncFailureUnsupportedStoredDataVersion, SyncFailureIncompatibleNewSyncData:
		return true
	default:
		return false
	}
}

// diagnosticContextFromError extracts structured troubleshooting context while keeping secrets redacted.
// Authored by: OpenCode
func diagnosticContextFromError(
	err error,
	defaultStage syncmodel.DiagnosticFailureStage,
	secrets ...string,
) syncmodel.DiagnosticContext {
	var context = syncmodel.DiagnosticContext{
		FailureStage:  defaultStage,
		FailureDetail: redact.ErrorText(err, secrets...),
	}
	if err == nil {
		return context
	}

	var carrier interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	}
	if typed, ok := err.(interface {
		DiagnosticContext() syncmodel.DiagnosticContext
	}); ok {
		carrier = typed
		context = carrier.DiagnosticContext()
		if context.FailureStage == "" {
			context.FailureStage = defaultStage
		}
		context.FailureDetail = redact.Text(context.FailureDetail, secrets...)
		return context
	}

	return context
}

// clearSessionSecrets removes transient secret material from the active session.
// Authored by: OpenCode
func clearSessionSecrets(session *GhostfolioSession) {
	session.SecurityToken = ""
	session.AuthToken = ""
}

// newSnapshotPayload builds the phase-3 protected snapshot payload.
// Authored by: OpenCode
func newSnapshotPayload(
	config configmodel.AppSetupConfig,
	cache syncmodel.ProtectedActivityCache,
	existing snapshotmodel.Payload,
	hasExisting bool,
) snapshotmodel.Payload {
	var now = cache.SyncedAt.UTC()
	var registeredLocalUser snapshotmodel.RegisteredLocalUser
	if hasExisting {
		registeredLocalUser = existing.RegisteredLocalUser
		registeredLocalUser.UpdatedAt = now
		registeredLocalUser.LastSuccessfulSyncAt = now
	} else {
		var localUserID, err = randomIdentifier(16)
		if err != nil {
			localUserID = ""
		}
		registeredLocalUser = snapshotmodel.RegisteredLocalUser{
			LocalUserID:          localUserID,
			CreatedAt:            now,
			UpdatedAt:            now,
			LastSuccessfulSyncAt: now,
		}
	}

	return snapshotmodel.Payload{
		StoredDataVersion:   snapshotmodel.DefaultStoredDataVersion(""),
		RegisteredLocalUser: registeredLocalUser,
		SetupProfile: snapshotmodel.SetupProfile{
			ServerOrigin:      config.ServerOrigin,
			ServerMode:        config.ServerMode,
			AllowDevHTTP:      config.AllowDevHTTP,
			LastValidatedAt:   now,
			SourceAPIBasePath: "api/v1",
		},
		ProtectedActivityCache: cache,
	}
}

// setActiveSnapshot stores the readable protected snapshot for the current run.
// Authored by: OpenCode
func (s *syncService) setActiveSnapshot(candidate snapshotstore.Candidate, payload snapshotmodel.Payload) {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()
	s.activeSnapshot = activeReadableSnapshot{Candidate: candidate, Payload: payload, Present: true}
}

// ProtectedDataState reports whether a readable protected snapshot is active for this run.
// Authored by: OpenCode
func (s *syncService) ProtectedDataState() ProtectedDataState {
	s.activeMutex.Lock()
	defer s.activeMutex.Unlock()

	if !s.activeSnapshot.Present {
		return ProtectedDataState{}
	}

	return ProtectedDataState{
		HasReadableSnapshot: true,
		ServerOrigin:        s.activeSnapshot.Payload.SetupProfile.ServerOrigin,
	}
}

// CheckServerReplacement compares the selected server against the active readable snapshot.
// Authored by: OpenCode
func (s *syncService) CheckServerReplacement(config configmodel.AppSetupConfig) ServerReplacementCheck {
	var state = s.ProtectedDataState()
	if !state.HasReadableSnapshot || state.ServerOrigin == "" || state.ServerOrigin == config.ServerOrigin {
		return ServerReplacementCheck{}
	}

	return ServerReplacementCheck{
		Required:             true,
		ActiveServerOrigin:   state.ServerOrigin,
		SelectedServerOrigin: config.ServerOrigin,
	}
}

// randomIdentifier creates one opaque hexadecimal identifier.
// Authored by: OpenCode
func randomIdentifier(byteLength int) (string, error) {
	var rawValue = make([]byte, byteLength)
	if _, err := readRandom(rawValue); err != nil {
		return "", fmt.Errorf("read secure random bytes: %w", err)
	}

	return hex.EncodeToString(rawValue), nil
}
