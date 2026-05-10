package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

type fakeStore struct {
	loadFunc func(context.Context) (configmodel.AppSetupConfig, error)
}

func (f fakeStore) Load(ctx context.Context) (configmodel.AppSetupConfig, error) {
	return f.loadFunc(ctx)
}

func (fakeStore) Save(context.Context, configmodel.AppSetupConfig) error { return nil }
func (fakeStore) Delete(context.Context) error                           { return nil }
func (fakeStore) Path() string                                           { return "" }

func TestParseOptionsRejectsUnknownFlag(t *testing.T) {
	_, err := ParseOptions([]string{"--unknown-flag"})
	if err == nil {
		t.Fatalf("expected parse error for unknown flag")
	}
}

func TestParseOptionsRejectsInvalidDuration(t *testing.T) {
	_, err := ParseOptions([]string{"--request-timeout", "not-a-duration"})
	if err == nil {
		t.Fatalf("expected parse error for invalid duration")
	}
}

func TestParseOptionsRejectsNonPositiveDuration(t *testing.T) {
	t.Parallel()

	_, err := ParseOptions([]string{"--request-timeout", "0s"})
	if err == nil {
		t.Fatalf("expected parse error for non-positive duration")
	}
}

func TestParseOptionsAcceptsSupportedFlags(t *testing.T) {
	t.Parallel()

	var options, err = ParseOptions([]string{"--config-dir", "/tmp/test", "--dev-mode", "--request-timeout", "45s", "--window-width", "120", "--window-height", "40"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if options.ConfigDir != "/tmp/test" || !options.AllowDevHTTP || options.RequestTimeout != 45*time.Second || options.InitialWindowWidth != 120 || options.InitialWindowHeight != 40 {
		t.Fatalf("unexpected parsed options: %#v", options)
	}
}

func TestLoadStartupStateReturnsInvalidRememberedSetupMessage(t *testing.T) {
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, "http://localhost:8080", true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var state StartupState
	state, err = LoadStartupState(context.Background(), fakeStore{loadFunc: func(context.Context) (configmodel.AppSetupConfig, error) {
		return config, nil
	}}, false)
	if err != nil {
		t.Fatalf("load startup state: %v", err)
	}
	if !state.NeedsSetup || state.InvalidSetupMessage == "" {
		t.Fatalf("expected invalid remembered setup fallback: %#v", state)
	}
}

func TestLoadStartupStatePropagatesStoreError(t *testing.T) {
	var expected = errors.New("boom")
	_, err := LoadStartupState(context.Background(), fakeStore{loadFunc: func(context.Context) (configmodel.AppSetupConfig, error) {
		return configmodel.AppSetupConfig{}, expected
	}}, false)
	if !errors.Is(err, expected) {
		t.Fatalf("expected wrapped store error, got %v", err)
	}
}

func TestLoadStartupStateHandlesNotFound(t *testing.T) {
	var state, err = LoadStartupState(context.Background(), fakeStore{loadFunc: func(context.Context) (configmodel.AppSetupConfig, error) {
		return configmodel.AppSetupConfig{}, configstore.ErrNotFound
	}}, false)
	if err != nil {
		t.Fatalf("load startup state: %v", err)
	}
	if !state.NeedsSetup {
		t.Fatalf("expected setup to be required")
	}
}

func TestLoadStartupStateReturnsActiveConfigWhenValid(t *testing.T) {
	t.Parallel()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	var state StartupState
	state, err = LoadStartupState(context.Background(), fakeStore{loadFunc: func(context.Context) (configmodel.AppSetupConfig, error) {
		return config, nil
	}}, false)
	if err != nil {
		t.Fatalf("load startup state: %v", err)
	}
	if state.ActiveConfig == nil || state.ActiveConfig.ServerOrigin != config.ServerOrigin {
		t.Fatalf("expected active config to be returned: %#v", state)
	}
}
