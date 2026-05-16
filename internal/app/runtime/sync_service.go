// Package runtime assembles application dependencies for the TUI runtime.
// Authored by: OpenCode
package runtime

import (
	"context"
	"errors"
	"fmt"
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

// SyncRequest contains the bootstrap configuration and runtime-only token
// needed for one full-history sync attempt.
//
// Authored by: OpenCode
type SyncRequest struct {
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

// SyncService runs Ghostfolio activity sync for the currently selected
// bootstrap setup.
//
// Authored by: OpenCode
type SyncService interface {
	// Run authenticates anonymously, retrieves the full supported activity
	// history, normalizes and validates it, and stores the result as a protected
	// local snapshot.
	//
	// Authored by: OpenCode
	Run(context.Context, SyncRequest) SyncOutcome

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

// syncService coordinates one full-history sync attempt across the Ghostfolio
// client boundary, domain services, and runtime collaborators.
// Authored by: OpenCode
type syncService struct {
	client            *ghostfolioclient.Client
	requestTimeout    time.Duration
	allowDevHTTP      bool
	decimalService    decimalsupport.Service
	normalizer        syncnormalize.Normalizer
	validator         syncvalidate.Validator
	snapshots         *snapshotLifecycle
	diagnosticReports diagnosticReportService
}

// NewSyncService creates the runtime sync service.
//
// Example:
//
//	service := runtime.NewSyncService(ghostfolioclient.New(nil), 30*time.Second, decimal.NewService(), normalize.NewNormalizer(), validate.NewValidator(), snapshots)
//	_ = service
//
// The returned service depends on the Ghostfolio HTTP client boundary and
// enforces a per-attempt timeout for the full sync workflow.
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
		client:            client,
		requestTimeout:    requestTimeout,
		allowDevHTTP:      allowDevHTTP,
		decimalService:    decimalService,
		normalizer:        normalizer,
		validator:         validatorService,
		snapshots:         newSnapshotLifecycle(snapshots, newActiveSnapshotState(), protectedPayloadBuilder{}),
		diagnosticReports: newDiagnosticReportService(baseConfigDir),
	}
}

// Run executes one full Ghostfolio activity sync attempt.
//
// Example:
//
//	outcome := service.Run(context.Background(), runtime.SyncRequest{Config: config, SecurityToken: "token"})
//	_ = outcome.Success
//
// Run authenticates anonymously against the configured origin, retrieves the
// full activity history, normalizes and validates the supported data, and
// stores it as a protected snapshot.
// Authored by: OpenCode
func (s *syncService) Run(ctx context.Context, request SyncRequest) SyncOutcome {
	var timedContext, cancel = context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	var session, attempt = newSyncAttemptState(request)
	var unlockedSnapshot, outcome, ok = s.prepareSyncAttempt(timedContext, request, &session, &attempt)
	if !ok {
		return outcome
	}

	attempt.Status = AttemptStatusAuthenticating

	authResponse, err := s.client.Authenticate(timedContext, session.ServerOrigin, session.SecurityToken)
	if err != nil {
		return finalizeBoundaryFailure(&session, &attempt, err)
	}
	if err := validator.ValidateAuthResponse(authResponse); err != nil {
		return finalizeContractFailure(&session, &attempt, err)
	}

	session.AuthToken = authResponse.AuthToken
	session.AuthenticatedAt = time.Now().UTC()
	attempt.Status = AttemptStatusRetrievingHistory

	historyResponse, err := s.client.FetchActivitiesHistory(timedContext, session.ServerOrigin, session.AuthToken)
	if err != nil {
		return finalizeBoundaryFailure(&session, &attempt, err)
	}

	if err := validator.ValidateActivityPageResponse(historyResponse); err != nil {
		return finalizeContractFailure(&session, &attempt, err)
	}

	cache, outcome, ok := s.buildProtectedActivityCache(historyResponse, request.SecurityToken, &session, &attempt)
	if !ok {
		return outcome
	}
	cache.SyncedAt = time.Now().UTC()

	attempt.Status = AttemptStatusValidating
	if err := s.validator.Validate(cache); err != nil {
		return s.finalizeSyncFailure(
			&session,
			&attempt,
			SyncFailureUnsupportedActivityHistory,
			diagnosticContextFromError(err, syncmodel.DiagnosticFailureStageValidation, request.SecurityToken),
		)
	}

	attempt.Status = AttemptStatusPersisting
	err = s.snapshots.Persist(timedContext, snapshotPersistRequest{
		Config:        request.Config,
		SecurityToken: request.SecurityToken,
		Cache:         cache,
		Existing:      unlockedSnapshot,
	})
	if err != nil {
		return s.finalizePersistenceFailure(&session, &attempt, err, request.SecurityToken)
	}

	attempt.Status = AttemptStatusSuccess
	attempt.CompletedAt = time.Now().UTC()
	clearSessionSecrets(&session)

	return SyncOutcome{
		Success:      true,
		DetailReason: "activity_data_stored",
		Attempt:      attempt,
	}
}

// newSyncAttemptState creates the transient session and attempt state for one sync run.
// Authored by: OpenCode
func newSyncAttemptState(request SyncRequest) (GhostfolioSession, SyncAttempt) {
	var now = time.Now().UTC()

	return GhostfolioSession{
			ServerOrigin:  request.Config.ServerOrigin,
			SecurityToken: request.SecurityToken,
			StartedAt:     now,
		}, SyncAttempt{
			AttemptID: fmt.Sprintf("attempt-%d", now.UnixNano()),
			Status:    AttemptStatusDiscoveringSnapshot,
			StartedAt: now,
		}
}

// prepareSyncAttempt validates snapshot availability, replacement intent, and selected-server unlock state.
// Authored by: OpenCode
func (s *syncService) prepareSyncAttempt(
	ctx context.Context,
	request SyncRequest,
	session *GhostfolioSession,
	attempt *SyncAttempt,
) (snapshotUnlockResult, SyncOutcome, bool) {
	if s.snapshots == nil {
		return snapshotUnlockResult{}, s.finalizeSyncFailure(session, attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
			FailureStage:  syncmodel.DiagnosticFailureStageProtectedPersistence,
			FailureDetail: "protected snapshot store is unavailable",
		}), false
	}

	var outcome, ok = s.confirmServerReplacement(request, session, attempt)
	if !ok {
		return snapshotUnlockResult{}, outcome, false
	}

	unlockedSnapshot, err := s.snapshots.DiscoverAndUnlock(ctx, request.Config.ServerOrigin, request.SecurityToken)
	if err == nil {
		return unlockedSnapshot, SyncOutcome{}, true
	}

	return snapshotUnlockResult{}, s.finalizeUnlockFailure(session, attempt, err, request.SecurityToken), false
}

// confirmServerReplacement handles the active-snapshot replacement gate before retrieval starts.
// Authored by: OpenCode
func (s *syncService) confirmServerReplacement(
	request SyncRequest,
	session *GhostfolioSession,
	attempt *SyncAttempt,
) (SyncOutcome, bool) {
	var replacementCheck = s.CheckServerReplacement(request.Config)
	if !replacementCheck.Required {
		return SyncOutcome{}, true
	}
	if !request.ConfirmServerReplacement {
		attempt.Status = AttemptStatusAborted
		return s.finalizeSyncFailure(session, attempt, SyncFailureServerReplacementCancelled, syncmodel.DiagnosticContext{}), false
	}

	attempt.ServerMismatchConfirmed = true
	return SyncOutcome{}, true
}

// finalizeUnlockFailure maps snapshot discovery and unlock failures into supported sync outcomes.
// Authored by: OpenCode
func (s *syncService) finalizeUnlockFailure(
	session *GhostfolioSession,
	attempt *SyncAttempt,
	err error,
	securityToken string,
) SyncOutcome {
	var reason = SyncFailureIncompatibleNewSyncData
	if errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
		reason = SyncFailureUnsupportedStoredDataVersion
	}

	return s.finalizeSyncFailure(session, attempt, reason, syncmodel.DiagnosticContext{
		FailureStage:  syncmodel.DiagnosticFailureStageStoredDataCompatibility,
		FailureDetail: redact.ErrorText(err, securityToken),
	})
}

// buildProtectedActivityCache maps, normalizes, and returns the protected activity cache for one retrieved history.
// Authored by: OpenCode
func (s *syncService) buildProtectedActivityCache(
	historyResponse ghostfoliodto.ActivityPageResponse,
	securityToken string,
	session *GhostfolioSession,
	attempt *SyncAttempt,
) (syncmodel.ProtectedActivityCache, SyncOutcome, bool) {
	attempt.Status = AttemptStatusNormalizing

	records, err := ghostfoliomapper.MapActivities(historyResponse.Activities, s.decimalService)
	if err != nil {
		return syncmodel.ProtectedActivityCache{}, s.finalizeSyncFailure(
			session,
			attempt,
			SyncFailureUnsupportedActivityHistory,
			diagnosticContextFromError(err, syncmodel.DiagnosticFailureStageMapping, securityToken),
		), false
	}

	cache, err := s.normalizer.Normalize(records)
	if err != nil {
		return syncmodel.ProtectedActivityCache{}, s.finalizeSyncFailure(
			session,
			attempt,
			SyncFailureUnsupportedActivityHistory,
			diagnosticContextFromError(err, syncmodel.DiagnosticFailureStageNormalization, securityToken),
		), false
	}

	return cache, SyncOutcome{}, true
}

// finalizePersistenceFailure converts protected-write failures into one supported sync outcome.
// Authored by: OpenCode
func (s *syncService) finalizePersistenceFailure(
	session *GhostfolioSession,
	attempt *SyncAttempt,
	err error,
	securityToken string,
) SyncOutcome {
	return s.finalizeSyncFailure(session, attempt, SyncFailureIncompatibleNewSyncData, syncmodel.DiagnosticContext{
		FailureStage:  syncmodel.DiagnosticFailureStageProtectedPersistence,
		FailureDetail: redact.ErrorText(err, securityToken),
	})
}

// GenerateDiagnosticReport writes one local synced-data diagnostic report for an eligible failure.
// Authored by: OpenCode
func (s *syncService) GenerateDiagnosticReport(ctx context.Context, request DiagnosticReportRequest) (string, error) {
	return s.diagnosticReports.Write(ctx, request)
}

// finalizeBoundaryFailure converts a boundary failure into a user-visible sync outcome.
// Authored by: OpenCode
func finalizeBoundaryFailure(session *GhostfolioSession, attempt *SyncAttempt, err error) SyncOutcome {
	attempt.Status = AttemptStatusFailed
	attempt.CompletedAt = time.Now().UTC()

	var category = SyncFailureConnectivityProblem
	var requestFailure *ghostfolioclient.RequestFailure
	if errors.As(err, &requestFailure) {
		category = syncFailureReasonFromBoundary(requestFailure.Category)
	}
	attempt.FailureReason = category
	clearSessionSecrets(session)

	return SyncOutcome{
		Success:       false,
		DetailReason:  string(category),
		FailureReason: category,
		Attempt:       *attempt,
	}
}

// finalizeContractFailure converts a contract compatibility error into the
// supported incompatible-server outcome.
// Authored by: OpenCode
func finalizeContractFailure(
	session *GhostfolioSession,
	attempt *SyncAttempt,
	err error,
) SyncOutcome {
	_ = err
	attempt.Status = AttemptStatusFailed
	attempt.CompletedAt = time.Now().UTC()
	attempt.FailureReason = SyncFailureIncompatibleServerContract
	clearSessionSecrets(session)

	return SyncOutcome{
		Success:       false,
		DetailReason:  string(SyncFailureIncompatibleServerContract),
		FailureReason: SyncFailureIncompatibleServerContract,
		Attempt:       *attempt,
	}
}

// finalizeSyncFailure converts an internal sync failure into one supported user-visible category.
// Authored by: OpenCode
func (s *syncService) finalizeSyncFailure(
	session *GhostfolioSession,
	attempt *SyncAttempt,
	reason SyncFailureReason,
	diagnosticContext syncmodel.DiagnosticContext,
) SyncOutcome {
	attempt.Status = AttemptStatusFailed
	attempt.CompletedAt = time.Now().UTC()
	attempt.FailureReason = reason
	clearSessionSecrets(session)

	var outcome = SyncOutcome{
		Success:       false,
		DetailReason:  string(reason),
		FailureReason: reason,
		Attempt:       *attempt,
	}
	if !diagnosticEligible(reason) {
		return outcome
	}

	var request = DiagnosticReportRequest{
		FailureReason:           reason,
		ServerOrigin:            session.ServerOrigin,
		Attempt:                 *attempt,
		Context:                 diagnosticContext,
		RedactFinancialValues:   !s.allowDevHTTP,
		ExplicitDevelopmentMode: s.allowDevHTTP,
	}
	outcome.Diagnostic = s.diagnosticReports.PrepareState(request)

	return outcome
}

// syncFailureReasonFromBoundary maps transport-boundary failures into the
// application-facing sync failure taxonomy.
// Authored by: OpenCode
func syncFailureReasonFromBoundary(category ghostfolioclient.FailureCategory) SyncFailureReason {
	switch category {
	case ghostfolioclient.FailureRejectedToken:
		return SyncFailureRejectedToken
	case ghostfolioclient.FailureTimeout:
		return SyncFailureTimeout
	case ghostfolioclient.FailureConnectivityProblem:
		return SyncFailureConnectivityProblem
	case ghostfolioclient.FailureUnsuccessfulServerResponse:
		return SyncFailureUnsuccessfulServerResponse
	case ghostfolioclient.FailureIncompatibleServerContract:
		return SyncFailureIncompatibleServerContract
	default:
		return SyncFailureConnectivityProblem
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

// setActiveSnapshot stores the readable protected snapshot for the current run.
// Authored by: OpenCode
func (s *syncService) setActiveSnapshot(candidate snapshotstore.Candidate, payload snapshotmodel.Payload) {
	if s.snapshots == nil {
		return
	}

	s.snapshots.SetActiveSnapshot(candidate, payload)
}

// ProtectedDataState reports whether a readable protected snapshot is active for this run.
// Authored by: OpenCode
func (s *syncService) ProtectedDataState() ProtectedDataState {
	if s.snapshots == nil {
		return ProtectedDataState{}
	}

	return s.snapshots.ProtectedDataState()
}

// CheckServerReplacement compares the selected server against the active readable snapshot.
// Authored by: OpenCode
func (s *syncService) CheckServerReplacement(config configmodel.AppSetupConfig) ServerReplacementCheck {
	if s.snapshots == nil {
		return ServerReplacementCheck{}
	}

	return s.snapshots.CheckServerReplacement(config)
}
