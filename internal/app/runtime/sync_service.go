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
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

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

	if s.snapshotStore == nil {
		return finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData)
	}

	var replacementCheck = s.CheckServerReplacement(request.Config)
	if replacementCheck.Required && !request.ConfirmServerReplacement {
		attempt.Status = AttemptStatusAborted
		return finalizeSyncFailure(&session, &attempt, SyncFailureServerReplacementCancelled)
	}
	if replacementCheck.Required {
		attempt.ServerMismatchConfirmed = true
	}

	var candidates []snapshotstore.Candidate
	candidates, err = snapshotstore.DiscoverServerCandidates(timedContext, s.snapshotStore, request.Config.ServerOrigin)
	if err != nil {
		return finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData)
	}
	for _, candidate := range candidates {
		if err := snapshotstore.ValidateEnvelopeCompatibility(candidate.Header); err != nil {
			if errors.Is(err, snapshotstore.ErrUnsupportedStoredDataVersion) {
				return finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedStoredDataVersion)
			}
			return finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData)
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
			return finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedStoredDataVersion)
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
		return finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedActivityHistory)
	}

	cache, err := s.normalizer.Normalize(records)
	if err != nil {
		return finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedActivityHistory)
	}
	cache.SyncedAt = time.Now().UTC()

	attempt.Status = AttemptStatusValidating
	if err := s.validator.Validate(cache); err != nil {
		return finalizeSyncFailure(&session, &attempt, SyncFailureUnsupportedActivityHistory)
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
			return finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData)
		}
		return finalizeSyncFailure(&session, &attempt, SyncFailureIncompatibleNewSyncData)
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
func finalizeSyncFailure(
	session *GhostfolioSession,
	attempt *SyncValidationAttempt,
	reason SyncFailureReason,
) ValidationOutcome {
	attempt.Status = AttemptStatusFailure
	attempt.CompletedAt = time.Now().UTC()
	attempt.FailureReason = reason
	clearSessionSecrets(session)

	return ValidationOutcome{
		Success:       false,
		DetailReason:  string(reason),
		FailureReason: reason,
		Attempt:       SyncAttempt(*attempt),
	}
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
	if _, err := rand.Read(rawValue); err != nil {
		return "", fmt.Errorf("read secure random bytes: %w", err)
	}

	return hex.EncodeToString(rawValue), nil
}
