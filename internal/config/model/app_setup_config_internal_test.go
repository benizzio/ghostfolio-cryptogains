package model

import (
	"errors"
	"testing"
	"time"
)

func TestNewSetupConfigRejectsInvalidServerMode(t *testing.T) {
	t.Parallel()

	_, err := NewSetupConfig("invalid", GhostfolioCloudOrigin, false, time.Now())
	if !errors.Is(err, ErrInvalidServerMode) {
		t.Fatalf("expected invalid server mode, got %v", err)
	}
}

func TestNewSetupConfigBuildsCanonicalValues(t *testing.T) {
	t.Parallel()

	var now = time.Now()
	var config, err = NewSetupConfig(ServerModeCustomOrigin, "http://LOCALHOST:8080/", true, now)
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	if config.ServerOrigin != "http://localhost:8080" || !config.AllowDevHTTP || !config.SetupComplete || config.SchemaVersion != SchemaVersion {
		t.Fatalf("unexpected setup config: %#v", config)
	}
	if !config.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("expected UTC timestamp, got %v want %v", config.UpdatedAt, now.UTC())
	}
}

func TestValidateStartupReadyRejectsInvalidSchema(t *testing.T) {
	t.Parallel()

	var config = AppSetupConfig{SchemaVersion: 0}
	if err := config.ValidateStartupReady(false); err == nil {
		t.Fatalf("expected schema validation error")
	}
}

func TestValidateStartupReadyRejectsMissingTimestamp(t *testing.T) {
	t.Parallel()

	var config = AppSetupConfig{SchemaVersion: SchemaVersion, SetupComplete: true, ServerMode: ServerModeGhostfolioCloud, ServerOrigin: GhostfolioCloudOrigin}
	if err := config.ValidateStartupReady(false); err == nil {
		t.Fatalf("expected updated_at validation error")
	}
}

func TestValidateStartupReadyRejectsIncompleteSetupAndModeMismatch(t *testing.T) {
	t.Parallel()

	var incomplete = AppSetupConfig{SchemaVersion: SchemaVersion, SetupComplete: false, ServerMode: ServerModeGhostfolioCloud, ServerOrigin: GhostfolioCloudOrigin, UpdatedAt: time.Now()}
	if err := incomplete.ValidateStartupReady(false); !errors.Is(err, ErrIncompleteSetup) {
		t.Fatalf("expected incomplete setup error, got %v", err)
	}

	var invalidMode = AppSetupConfig{SchemaVersion: SchemaVersion, SetupComplete: true, ServerMode: "bad", ServerOrigin: GhostfolioCloudOrigin, UpdatedAt: time.Now()}
	if err := invalidMode.ValidateStartupReady(false); !errors.Is(err, ErrInvalidServerMode) {
		t.Fatalf("expected invalid mode error, got %v", err)
	}
}

func TestValidateStartupReadyRejectsNonCanonicalOrigin(t *testing.T) {
	t.Parallel()

	var config = AppSetupConfig{SchemaVersion: SchemaVersion, SetupComplete: true, ServerMode: ServerModeGhostfolioCloud, ServerOrigin: "https://GhostFol.io/", UpdatedAt: time.Now(), AllowDevHTTP: false}
	if err := config.ValidateStartupReady(false); err == nil {
		t.Fatalf("expected non-canonical origin error")
	}
}

func TestNormalizeOriginRejectsPathQueryFragmentAndUserInfo(t *testing.T) {
	t.Parallel()

	for _, origin := range []string{"https://ghostfol.io/path", "https://ghostfol.io?query=1", "https://ghostfol.io#fragment", "https://user@ghostfol.io"} {
		if _, err := NormalizeOrigin(origin, false); !errors.Is(err, ErrInvalidOrigin) {
			t.Fatalf("expected invalid origin for %q, got %v", origin, err)
		}
	}
}

func TestNormalizeOriginRejectsMissingHost(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeOrigin("https:///", false); !errors.Is(err, ErrInvalidOrigin) {
		t.Fatalf("expected invalid origin, got %v", err)
	}
}

func TestValidateStartupReadyAcceptsCanonicalConfig(t *testing.T) {
	t.Parallel()

	var config = AppSetupConfig{
		SchemaVersion: SchemaVersion,
		SetupComplete: true,
		ServerMode:    ServerModeGhostfolioCloud,
		ServerOrigin:  GhostfolioCloudOrigin,
		AllowDevHTTP:  false,
		UpdatedAt:     time.Now(),
	}
	if err := config.ValidateStartupReady(false); err != nil {
		t.Fatalf("expected startup config to be valid: %v", err)
	}
}

func TestValidateStartupReadyRejectsAllowDevHTTPMismatch(t *testing.T) {
	t.Parallel()

	var config = AppSetupConfig{
		SchemaVersion: SchemaVersion,
		SetupComplete: true,
		ServerMode:    ServerModeCustomOrigin,
		ServerOrigin:  "http://localhost:8080",
		AllowDevHTTP:  false,
		UpdatedAt:     time.Now(),
	}
	if err := config.ValidateStartupReady(true); err == nil {
		t.Fatalf("expected allow_dev_http mismatch error")
	}
}

func TestNormalizeOriginRejectsBlankAndMalformedOrigins(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeOrigin("   ", false); !errors.Is(err, ErrInvalidOrigin) {
		t.Fatalf("expected blank origin error, got %v", err)
	}
	if _, err := NormalizeOrigin("://bad", false); err == nil {
		t.Fatalf("expected malformed origin parse error")
	}
	if _, err := NormalizeOrigin("ghostfol.io", false); !errors.Is(err, ErrInvalidOrigin) {
		t.Fatalf("expected absolute-origin validation error, got %v", err)
	}
}

func TestNewSetupConfigRejectsInvalidOrigin(t *testing.T) {
	t.Parallel()

	if _, err := NewSetupConfig(ServerModeGhostfolioCloud, "", false, time.Now()); !errors.Is(err, ErrInvalidOrigin) {
		t.Fatalf("expected invalid origin error, got %v", err)
	}
}

func TestNewSetupConfigRejectsServerModeOriginMismatch(t *testing.T) {
	t.Parallel()

	if _, err := NewSetupConfig(ServerModeGhostfolioCloud, "https://example.com", false, time.Now()); !errors.Is(err, ErrInvalidServerOrigin) {
		t.Fatalf("expected invalid server origin for cloud mode, got %v", err)
	}
	if _, err := NewSetupConfig(ServerModeCustomOrigin, GhostfolioCloudOrigin, false, time.Now()); !errors.Is(err, ErrInvalidServerOrigin) {
		t.Fatalf("expected invalid server origin for custom mode, got %v", err)
	}
}

func TestValidateStartupReadyRejectsStoredOriginNormalizationFailure(t *testing.T) {
	t.Parallel()

	var config = AppSetupConfig{
		SchemaVersion: SchemaVersion,
		SetupComplete: true,
		ServerMode:    ServerModeGhostfolioCloud,
		ServerOrigin:  "https:///",
		AllowDevHTTP:  false,
		UpdatedAt:     time.Now(),
	}
	if err := config.ValidateStartupReady(false); !errors.Is(err, ErrInvalidOrigin) {
		t.Fatalf("expected invalid origin error, got %v", err)
	}
}

func TestValidateStartupReadyRejectsServerModeOriginMismatch(t *testing.T) {
	t.Parallel()

	var invalidCloudConfig = AppSetupConfig{
		SchemaVersion: SchemaVersion,
		SetupComplete: true,
		ServerMode:    ServerModeGhostfolioCloud,
		ServerOrigin:  "https://example.com",
		AllowDevHTTP:  false,
		UpdatedAt:     time.Now(),
	}
	if err := invalidCloudConfig.ValidateStartupReady(false); !errors.Is(err, ErrInvalidServerOrigin) {
		t.Fatalf("expected invalid server origin for cloud mode, got %v", err)
	}

	var invalidCustomConfig = AppSetupConfig{
		SchemaVersion: SchemaVersion,
		SetupComplete: true,
		ServerMode:    ServerModeCustomOrigin,
		ServerOrigin:  GhostfolioCloudOrigin,
		AllowDevHTTP:  false,
		UpdatedAt:     time.Now(),
	}
	if err := invalidCustomConfig.ValidateStartupReady(false); !errors.Is(err, ErrInvalidServerOrigin) {
		t.Fatalf("expected invalid server origin for custom mode, got %v", err)
	}
}

func TestValidateServerModeOriginCoversAllBranches(t *testing.T) {
	t.Parallel()

	if err := validateServerModeOrigin(ServerModeGhostfolioCloud, GhostfolioCloudOrigin); err != nil {
		t.Fatalf("expected cloud origin to be valid for cloud mode: %v", err)
	}
	if err := validateServerModeOrigin(ServerModeCustomOrigin, "https://example.com"); err != nil {
		t.Fatalf("expected custom origin to be valid for custom mode: %v", err)
	}
	if err := validateServerModeOrigin(ServerModeGhostfolioCloud, "https://example.com"); !errors.Is(err, ErrInvalidServerOrigin) {
		t.Fatalf("expected invalid origin for cloud mode, got %v", err)
	}
	if err := validateServerModeOrigin(ServerModeCustomOrigin, GhostfolioCloudOrigin); !errors.Is(err, ErrInvalidServerOrigin) {
		t.Fatalf("expected invalid origin for custom mode, got %v", err)
	}
	if err := validateServerModeOrigin("invalid", GhostfolioCloudOrigin); !errors.Is(err, ErrInvalidServerMode) {
		t.Fatalf("expected invalid server mode, got %v", err)
	}
}

func TestNormalizeOriginRejectsUnsupportedSchemeAndEmptyHostname(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeOrigin("ftp://ghostfol.io", true); !errors.Is(err, ErrDisallowedTransport) {
		t.Fatalf("expected disallowed transport, got %v", err)
	}
	if _, err := NormalizeOrigin("https://:8080", false); !errors.Is(err, ErrInvalidOrigin) {
		t.Fatalf("expected invalid empty-hostname origin, got %v", err)
	}
}
