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
	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/validator"
)

// AttemptStatus identifies the current phase of a validation attempt.
//
// Example:
//
//	var status runtime.AttemptStatus = runtime.AttemptStatusAuthenticating
//	_ = status
//
// Authored by: OpenCode
type AttemptStatus string

const (
	// AttemptStatusIdle indicates that no validation attempt is running.
	AttemptStatusIdle AttemptStatus = "idle"

	// AttemptStatusAuthenticating indicates that anonymous auth is in flight.
	AttemptStatusAuthenticating AttemptStatus = "authenticating"

	// AttemptStatusRequestingActivities indicates that the activities request is in flight.
	AttemptStatusRequestingActivities AttemptStatus = "requesting_activities"

	// AttemptStatusValidatingPayload indicates that payload validation is running.
	AttemptStatusValidatingPayload AttemptStatus = "validating_payload"

	// AttemptStatusSuccess indicates a completed successful validation attempt.
	AttemptStatusSuccess AttemptStatus = "success"

	// AttemptStatusFailure indicates a completed failed validation attempt.
	AttemptStatusFailure AttemptStatus = "failure"
)

// GhostfolioSession is the transient authenticated runtime state for one
// validation attempt.
//
// Example:
//
//	session := runtime.GhostfolioSession{ServerOrigin: "https://ghostfol.io"}
//	_ = session.ServerOrigin
//
// Authored by: OpenCode
type GhostfolioSession struct {
	ServerOrigin    string
	SecurityToken   string
	AuthToken       string
	StartedAt       time.Time
	AuthenticatedAt time.Time
}

// SyncValidationAttempt is the transient workflow state for one sync-data
// validation run.
//
// Example:
//
//	attempt := runtime.SyncValidationAttempt{AttemptID: "attempt-1", Status: runtime.AttemptStatusIdle}
//	_ = attempt.AttemptID
//
// Authored by: OpenCode
type SyncValidationAttempt struct {
	AttemptID     string
	Status        AttemptStatus
	FailureReason ghostfolioclient.FailureCategory
	StartedAt     time.Time
	CompletedAt   time.Time
}

// ValidationOutcome is the user-visible result of a completed validation
// attempt.
//
// Example:
//
//	outcome := runtime.ValidationOutcome{Success: true, DetailReason: "communication_ok"}
//	_ = outcome.Success
//
// Authored by: OpenCode
type ValidationOutcome struct {
	Success         bool
	SummaryMessage  string
	DetailReason    string
	FollowUpNote    string
	FailureCategory ghostfolioclient.FailureCategory
	Attempt         SyncValidationAttempt
}

// SyncService validates Ghostfolio communication for the currently selected
// bootstrap setup.
//
// Example:
//
//	var service runtime.SyncService
//	_ = service
//
// Authored by: OpenCode
type SyncService interface {
	Validate(context.Context, configmodel.AppSetupConfig, string) ValidationOutcome
}

type syncService struct {
	client         *ghostfolioclient.Client
	requestTimeout time.Duration
}

// NewSyncService creates the runtime sync-validation service.
//
// Example:
//
//	service := runtime.NewSyncService(ghostfolioclient.New(nil), 30*time.Second)
//	_ = service
//
// Authored by: OpenCode
func NewSyncService(client *ghostfolioclient.Client, requestTimeout time.Duration) SyncService {
	return &syncService{client: client, requestTimeout: requestTimeout}
}

// Validate executes one full Ghostfolio communication-validation attempt.
//
// Example:
//
//	outcome := service.Validate(context.Background(), config, "token")
//	_ = outcome.Success
//
// Authored by: OpenCode
func (s *syncService) Validate(ctx context.Context, config configmodel.AppSetupConfig, securityToken string) ValidationOutcome {
	var timedContext, cancel = context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	var now = time.Now().UTC()
	var session = GhostfolioSession{
		ServerOrigin:  config.ServerOrigin,
		SecurityToken: securityToken,
		StartedAt:     now,
	}
	var attempt = SyncValidationAttempt{
		AttemptID: fmt.Sprintf("attempt-%d", now.UnixNano()),
		Status:    AttemptStatusAuthenticating,
		StartedAt: now,
	}

	var authResponse, err = s.client.Authenticate(timedContext, session.ServerOrigin, session.SecurityToken)
	if err != nil {
		return finalizeFailure(&session, &attempt, err)
	}
	if err := validator.ValidateAuthResponse(authResponse); err != nil {
		return finalizeValidationFailure(&session, &attempt, err)
	}

	session.AuthToken = authResponse.AuthToken
	session.AuthenticatedAt = time.Now().UTC()
	attempt.Status = AttemptStatusRequestingActivities

	var probeResponse, errProbe = s.client.FetchActivitiesProbe(timedContext, session.ServerOrigin, session.AuthToken)
	if errProbe != nil {
		return finalizeFailure(&session, &attempt, errProbe)
	}

	attempt.Status = AttemptStatusValidatingPayload
	if err := validator.ValidateActivitiesProbeResponse(probeResponse); err != nil {
		return finalizeValidationFailure(&session, &attempt, err)
	}

	attempt.Status = AttemptStatusSuccess
	attempt.CompletedAt = time.Now().UTC()
	clearSessionSecrets(&session)

	return ValidationOutcome{
		Success:        true,
		SummaryMessage: "Communication with the selected Ghostfolio server is working.",
		DetailReason:   "communication_ok",
		FollowUpNote:   "No Ghostfolio data was stored locally, and reporting is not available in this slice.",
		Attempt:        attempt,
	}
}

// finalizeFailure converts a boundary failure into a user-visible validation outcome.
// Authored by: OpenCode
func finalizeFailure(session *GhostfolioSession, attempt *SyncValidationAttempt, err error) ValidationOutcome {
	attempt.Status = AttemptStatusFailure
	attempt.CompletedAt = time.Now().UTC()

	var category = ghostfolioclient.FailureConnectivityProblem
	var requestFailure *ghostfolioclient.RequestFailure
	if errors.As(err, &requestFailure) {
		category = requestFailure.Category
	}
	attempt.FailureReason = category
	clearSessionSecrets(session)

	return ValidationOutcome{
		Success:         false,
		SummaryMessage:  "Communication validation did not succeed.",
		DetailReason:    string(category),
		FollowUpNote:    "Validate again or return to the main menu. No Ghostfolio data was stored locally.",
		FailureCategory: category,
		Attempt:         *attempt,
	}
}

// finalizeValidationFailure converts a payload validation error into the supported incompatible-server outcome.
// Authored by: OpenCode
func finalizeValidationFailure(session *GhostfolioSession, attempt *SyncValidationAttempt, err error) ValidationOutcome {
	_ = err
	attempt.Status = AttemptStatusFailure
	attempt.CompletedAt = time.Now().UTC()
	attempt.FailureReason = ghostfolioclient.FailureIncompatibleServerContract
	clearSessionSecrets(session)

	return ValidationOutcome{
		Success:         false,
		SummaryMessage:  "Communication validation did not succeed.",
		DetailReason:    string(ghostfolioclient.FailureIncompatibleServerContract),
		FollowUpNote:    "The selected server responded, but it did not satisfy the supported contract for this slice.",
		FailureCategory: ghostfolioclient.FailureIncompatibleServerContract,
		Attempt:         *attempt,
	}
}

// clearSessionSecrets removes transient secret material from the active session.
// Authored by: OpenCode
func clearSessionSecrets(session *GhostfolioSession) {
	session.SecurityToken = ""
	session.AuthToken = ""
}
