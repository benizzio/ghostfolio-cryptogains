// Package model defines the protected snapshot data structures shared across
// snapshot packages.
// Authored by: OpenCode
package model

import (
	"time"

	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
)

const (
	// PayloadSchemaVersion is the supported protected payload layout version.
	PayloadSchemaVersion = 1

	// ActivityModelVersion is the supported normalized protected-activity model version.
	// Version 4 already persists AssetIdentityKey in the protected payload, so the
	// corrected upstream `SymbolProfile.id` mapping does not require a new on-disk
	// model version.
	ActivityModelVersion = 4
)

// Payload stores the encrypted protected snapshot document.
// Authored by: OpenCode
type Payload struct {
	StoredDataVersion      StoredDataVersion                `json:"stored_data_version"`
	RegisteredLocalUser    RegisteredLocalUser              `json:"registered_local_user"`
	SetupProfile           SetupProfile                     `json:"setup_profile"`
	ProtectedActivityCache syncmodel.ProtectedActivityCache `json:"protected_activity_cache"`
}

// StoredDataVersion tracks protected stored-data compatibility markers.
// Authored by: OpenCode
type StoredDataVersion struct {
	EnvelopeFormatVersion int    `json:"envelope_format_version"`
	PayloadSchemaVersion  int    `json:"payload_schema_version"`
	ActivityModelVersion  int    `json:"activity_model_version"`
	WrittenByAppVersion   string `json:"written_by_app_version"`
}

// SetupProfile stores the selected Ghostfolio server context inside the
// encrypted payload.
// Authored by: OpenCode
type SetupProfile struct {
	ServerOrigin      string    `json:"server_origin"`
	ServerMode        string    `json:"server_mode"`
	AllowDevHTTP      bool      `json:"allow_dev_http"`
	LastValidatedAt   time.Time `json:"last_validated_at"`
	SourceAPIBasePath string    `json:"source_api_base_path"`
}

// RegisteredLocalUser stores the local protected-context identity without
// persisting the Ghostfolio token.
// Authored by: OpenCode
type RegisteredLocalUser struct {
	LocalUserID          string    `json:"local_user_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	LastSuccessfulSyncAt time.Time `json:"last_successful_sync_at"`
}

// DefaultStoredDataVersion returns the supported protected payload version set
// for this slice.
//
// Example:
//
//	version := model.DefaultStoredDataVersion("")
//	_ = version.PayloadSchemaVersion
//
// Authored by: OpenCode
func DefaultStoredDataVersion(writtenByAppVersion string) StoredDataVersion {
	return StoredDataVersion{
		EnvelopeFormatVersion: EnvelopeFormatVersion,
		PayloadSchemaVersion:  PayloadSchemaVersion,
		ActivityModelVersion:  ActivityModelVersion,
		WrittenByAppVersion:   writtenByAppVersion,
	}
}
