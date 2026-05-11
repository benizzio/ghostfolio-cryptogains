// Package model defines the bootstrap configuration persisted for the current
// application slice.
// Authored by: OpenCode
package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// SchemaVersion is the persisted document version for this bootstrap file.
	SchemaVersion = 1

	// GhostfolioCloudOrigin is the default hosted Ghostfolio origin.
	GhostfolioCloudOrigin = "https://ghostfol.io"

	// ServerModeGhostfolioCloud selects the hosted Ghostfolio service.
	ServerModeGhostfolioCloud = "ghostfolio_cloud"

	// ServerModeCustomOrigin selects a self-hosted Ghostfolio origin.
	ServerModeCustomOrigin = "custom_origin"
)

var (
	// ErrIncompleteSetup indicates that the persisted setup is not complete.
	ErrIncompleteSetup = errors.New("setup is incomplete")

	// ErrInvalidServerMode indicates that the persisted server mode is unknown.
	ErrInvalidServerMode = errors.New("server mode is invalid")

	// ErrInvalidOrigin indicates that an origin is malformed for this slice.
	ErrInvalidOrigin = errors.New("server origin must be an absolute origin with no path, query, fragment, or user info")

	// ErrInvalidServerOrigin indicates that the selected origin does not match the
	// configured server mode.
	ErrInvalidServerOrigin = errors.New("server origin does not match the selected server mode")

	// ErrDisallowedTransport indicates that the selected transport is not allowed.
	ErrDisallowedTransport = errors.New("custom origins must use https unless explicit development mode is enabled")
)

// AppSetupConfig is the machine-local bootstrap configuration loaded before
// any Ghostfolio token is requested.
//
// Authored by: OpenCode
type AppSetupConfig struct {
	SchemaVersion int       `json:"schema_version"`
	SetupComplete bool      `json:"setup_complete"`
	ServerMode    string    `json:"server_mode"`
	ServerOrigin  string    `json:"server_origin"`
	AllowDevHTTP  bool      `json:"allow_dev_http"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewSetupConfig builds a validated bootstrap configuration for persistence.
//
// Example:
//
//	config, err := model.NewSetupConfig(model.ServerModeCustomOrigin, "http://localhost:8080", true, time.Now())
//	if err != nil {
//		panic(err)
//	}
//	_ = config.AllowDevHTTP
//
// Authored by: OpenCode
func NewSetupConfig(serverMode string, origin string, allowDevHTTP bool, now time.Time) (AppSetupConfig, error) {
	var normalizedOrigin, err = NormalizeOrigin(origin, allowDevHTTP)
	if err != nil {
		return AppSetupConfig{}, err
	}

	if serverMode != ServerModeGhostfolioCloud && serverMode != ServerModeCustomOrigin {
		return AppSetupConfig{}, ErrInvalidServerMode
	}
	if err := validateServerModeOrigin(serverMode, normalizedOrigin); err != nil {
		return AppSetupConfig{}, err
	}

	return AppSetupConfig{
		SchemaVersion: SchemaVersion,
		SetupComplete: true,
		ServerMode:    serverMode,
		ServerOrigin:  normalizedOrigin,
		AllowDevHTTP:  strings.HasPrefix(normalizedOrigin, "http://"),
		UpdatedAt:     now.UTC(),
	}, nil
}

// ValidateStartupReady validates the stored bootstrap configuration for a new
// application launch.
//
// Example:
//
//	err := config.ValidateStartupReady(false)
//	if err != nil {
//		panic(err)
//	}
//
// Authored by: OpenCode
func (c AppSetupConfig) ValidateStartupReady(allowDevHTTP bool) error {
	if c.SchemaVersion <= 0 {
		return fmt.Errorf("schema version must be positive")
	}
	if !c.SetupComplete {
		return ErrIncompleteSetup
	}
	if c.ServerMode != ServerModeGhostfolioCloud && c.ServerMode != ServerModeCustomOrigin {
		return ErrInvalidServerMode
	}
	if c.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}

	var normalizedOrigin, err = NormalizeOrigin(c.ServerOrigin, allowDevHTTP)
	if err != nil {
		return err
	}
	if normalizedOrigin != c.ServerOrigin {
		return fmt.Errorf("stored origin is not canonical")
	}
	if c.AllowDevHTTP != strings.HasPrefix(c.ServerOrigin, "http://") {
		return fmt.Errorf("allow_dev_http does not match server origin")
	}
	if err := validateServerModeOrigin(c.ServerMode, normalizedOrigin); err != nil {
		return err
	}

	return nil
}

// validateServerModeOrigin enforces that the canonical origin matches the
// selected startup server mode.
// Authored by: OpenCode
func validateServerModeOrigin(serverMode string, normalizedOrigin string) error {
	switch serverMode {
	case ServerModeGhostfolioCloud:
		if normalizedOrigin != GhostfolioCloudOrigin {
			return ErrInvalidServerOrigin
		}
	case ServerModeCustomOrigin:
		if normalizedOrigin == GhostfolioCloudOrigin {
			return ErrInvalidServerOrigin
		}
	default:
		return ErrInvalidServerMode
	}

	return nil
}

// NormalizeOrigin canonicalizes and validates a selectable Ghostfolio origin.
//
// Example:
//
//	origin, err := model.NormalizeOrigin("https://GhostFol.io/", false)
//	if err != nil {
//		panic(err)
//	}
//	_ = origin
//
// Authored by: OpenCode
func NormalizeOrigin(rawOrigin string, allowDevHTTP bool) (string, error) {
	var trimmed = strings.TrimSpace(rawOrigin)
	if trimmed == "" {
		return "", ErrInvalidOrigin
	}

	var parsedOrigin, err = url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse origin: %w", err)
	}
	if parsedOrigin.Scheme == "" || parsedOrigin.Host == "" || parsedOrigin.Opaque != "" {
		return "", ErrInvalidOrigin
	}
	if parsedOrigin.Path != "" && parsedOrigin.Path != "/" {
		return "", ErrInvalidOrigin
	}
	if parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" || parsedOrigin.User != nil {
		return "", ErrInvalidOrigin
	}

	var scheme = strings.ToLower(parsedOrigin.Scheme)
	if scheme != "https" && !(allowDevHTTP && scheme == "http") {
		return "", ErrDisallowedTransport
	}

	var hostname = strings.ToLower(parsedOrigin.Hostname())
	if hostname == "" {
		return "", ErrInvalidOrigin
	}

	var host = hostname
	if port := parsedOrigin.Port(); port != "" {
		host = host + ":" + port
	}

	return (&url.URL{Scheme: scheme, Host: host}).String(), nil
}
