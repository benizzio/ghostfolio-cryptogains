package unit

import (
	"errors"
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
)

func TestNormalizeOriginCanonicalizesValidOrigins(t *testing.T) {
	t.Parallel()

	var origin, err = configmodel.NormalizeOrigin("https://GhostFol.io/", false)
	if err != nil {
		t.Fatalf("normalize origin: %v", err)
	}
	if origin != "https://ghostfol.io" {
		t.Fatalf("origin mismatch: got %q", origin)
	}
}

func TestNormalizeOriginRejectsDisallowedHTTP(t *testing.T) {
	t.Parallel()

	_, err := configmodel.NormalizeOrigin("http://localhost:8080", false)
	if !errors.Is(err, configmodel.ErrDisallowedTransport) {
		t.Fatalf("expected disallowed transport, got %v", err)
	}
}

func TestNormalizeOriginAllowsExplicitDevelopmentHTTP(t *testing.T) {
	t.Parallel()

	var origin, err = configmodel.NormalizeOrigin("http://localhost:8080", true)
	if err != nil {
		t.Fatalf("normalize origin: %v", err)
	}
	if origin != "http://localhost:8080" {
		t.Fatalf("origin mismatch: got %q", origin)
	}
}

func TestValidateStartupReadyRejectsMismatchedAllowDevHTTP(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, "http://localhost:8080", true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	if err := config.ValidateStartupReady(false); !errors.Is(err, configmodel.ErrDisallowedTransport) {
		t.Fatalf("expected invalid startup config, got %v", err)
	}
}
