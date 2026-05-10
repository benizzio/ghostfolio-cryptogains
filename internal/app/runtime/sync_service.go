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

// ValidationFailureReason identifies the supported structured failure outcome
// for a communication-validation attempt.
//
// Authored by: OpenCode
type ValidationFailureReason string

// Validation failure reasons define the supported user-visible failure taxonomy
// for one communication-validation attempt.
// Authored by: OpenCode
const (
	// ValidationFailureNone indicates that validation completed successfully.
	ValidationFailureNone ValidationFailureReason = ""

	// ValidationFailureRejectedToken indicates that Ghostfolio rejected the
	// supplied access token.
	ValidationFailureRejectedToken ValidationFailureReason = "rejected token"

	// ValidationFailureTimeout indicates that the validation request exceeded the
	// allowed runtime deadline.
	ValidationFailureTimeout ValidationFailureReason = "timeout"

	// ValidationFailureConnectivityProblem indicates a transport-level reachability
	// problem before a supported success response could be confirmed.
	ValidationFailureConnectivityProblem ValidationFailureReason = "connectivity problem"

	// ValidationFailureUnsuccessfulServerResponse indicates that the server was
	// reachable but returned an unsuccessful HTTP response.
	ValidationFailureUnsuccessfulServerResponse ValidationFailureReason = "unsuccessful server response"

	// ValidationFailureIncompatibleServerContract indicates that the server
	// response shape or behavior does not satisfy this slice's supported contract.
	ValidationFailureIncompatibleServerContract ValidationFailureReason = "incompatible server contract"
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

// Attempt statuses define the observable lifecycle phases for one validation
// attempt.
// Authored by: OpenCode
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
	FailureReason ValidationFailureReason
	StartedAt     time.Time
	CompletedAt   time.Time
}

// ValidationOutcome is the structured result of a completed validation
// attempt.
//
// Example:
//
//	outcome := runtime.ValidationOutcome{Success: true, DetailReason: "communication_ok"}
//	_ = outcome.Success
//
// The application layer returns outcome semantics only. Presentation layers are
// expected to convert the result into final user-facing wording.
// Authored by: OpenCode
type ValidationOutcome struct {
	Success       bool
	DetailReason  string
	FailureReason ValidationFailureReason
	Attempt       SyncValidationAttempt
}

// ValidateRequest contains the bootstrap configuration and runtime-only token
// needed for a single Ghostfolio communication check.
//
// Example:
//
//	request := runtime.ValidateRequest{Config: config, SecurityToken: "token"}
//	_ = request.SecurityToken
//
// Authored by: OpenCode
type ValidateRequest struct {
	Config        configmodel.AppSetupConfig
	SecurityToken string
}

// SyncService validates Ghostfolio communication for the currently selected
// bootstrap setup.
//
// Example:
//
//	outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token"})
//	_ = outcome.DetailReason
//
// Validate runs the anonymous-auth and one-page activities probe for the given
// setup and returns a structured semantic result. Successful outcomes never
// persist tokens or Ghostfolio payload data. Failed outcomes classify the
// attempt into one supported failure reason without exposing transport-layer
// implementation types to callers.
// Authored by: OpenCode
type SyncService interface {
	Validate(context.Context, ValidateRequest) ValidationOutcome
}

// syncService coordinates one validation attempt across the Ghostfolio client
// boundary and response validators.
// Authored by: OpenCode
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
// The returned service depends on the Ghostfolio HTTP client boundary and
// enforces a per-attempt timeout for the full validation workflow.
// Authored by: OpenCode
func NewSyncService(client *ghostfolioclient.Client, requestTimeout time.Duration) SyncService {
	return &syncService{client: client, requestTimeout: requestTimeout}
}

// Validate executes one full Ghostfolio communication-validation attempt.
//
// Example:
//
//	outcome := service.Validate(context.Background(), runtime.ValidateRequest{Config: config, SecurityToken: "token"})
//	_ = outcome.Success
//
// Validate authenticates anonymously against the configured origin, performs
// the one-page activities probe, validates both response contracts, and maps
// any failure into a structured application-facing result.
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
		Success:      true,
		DetailReason: "communication_ok",
		Attempt:      attempt,
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
		Attempt:       *attempt,
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
		Attempt:       *attempt,
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
