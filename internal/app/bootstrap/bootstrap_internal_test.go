package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	supporttext "github.com/benizzio/ghostfolio-cryptogains/internal/support/text"
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
	if options.ConfigDir != "/tmp/test" {
		t.Fatalf("unexpected ConfigDir: got %q want %q", options.ConfigDir, "/tmp/test")
	}
	if !options.AllowDevHTTP {
		t.Fatalf("expected AllowDevHTTP to be true")
	}
	if options.RequestTimeout != 45*time.Second {
		t.Fatalf("unexpected RequestTimeout: got %v want %v", options.RequestTimeout, 45*time.Second)
	}
	if options.InitialWindowWidth != 120 {
		t.Fatalf("unexpected InitialWindowWidth: got %d want %d", options.InitialWindowWidth, 120)
	}
	if options.InitialWindowHeight != 40 {
		t.Fatalf("unexpected InitialWindowHeight: got %d want %d", options.InitialWindowHeight, 40)
	}
}

func TestConfigureProcessDecimalPolicyUsesDefaultWhenUnset(t *testing.T) {
	var previousLookupEnv = lookupEnv
	defer func() {
		lookupEnv = previousLookupEnv
	}()

	var customPolicy, err = supportmath.ParseDecimalPolicy("scale=4,rounding=half_up")
	if err != nil {
		t.Fatalf("parse custom decimal policy: %v", err)
	}
	if err = supportmath.SetActiveDecimalPolicy(customPolicy); err != nil {
		t.Fatalf("set active decimal policy: %v", err)
	}
	t.Cleanup(func() {
		if resetErr := supportmath.SetActiveDecimalPolicy(supportmath.DefaultDecimalPolicy()); resetErr != nil {
			t.Fatalf("reset active decimal policy: %v", resetErr)
		}
	})

	lookupEnv = func(string) (string, bool) {
		return "", false
	}

	var policy supportmath.DecimalPolicy
	policy, err = ConfigureProcessDecimalPolicy()
	if err != nil {
		t.Fatalf("configure process decimal policy: %v", err)
	}
	if got := policy.CanonicalString(); got != supportmath.DefaultDecimalPolicy().CanonicalString() {
		t.Fatalf("unexpected configured policy: %q", got)
	}
	if got := supportmath.ActiveDecimalPolicy().CanonicalString(); got != supportmath.DefaultDecimalPolicy().CanonicalString() {
		t.Fatalf("unexpected active decimal policy: %q", got)
	}
}

func TestConfigureProcessDecimalPolicyAppliesEnvironmentOverride(t *testing.T) {
	var previousLookupEnv = lookupEnv
	defer func() {
		lookupEnv = previousLookupEnv
	}()
	t.Cleanup(func() {
		if resetErr := supportmath.SetActiveDecimalPolicy(supportmath.DefaultDecimalPolicy()); resetErr != nil {
			t.Fatalf("reset active decimal policy: %v", resetErr)
		}
	})

	lookupEnv = func(name string) (string, bool) {
		if name != reportDecimalPolicyEnvironmentVariable {
			t.Fatalf("unexpected environment-variable lookup: %q", name)
		}
		return "scale=4,rounding=half_up", true
	}

	var policy, err = ConfigureProcessDecimalPolicy()
	if err != nil {
		t.Fatalf("configure process decimal policy: %v", err)
	}
	if got := policy.CanonicalString(); got != "scale=4,rounding=half_up" {
		t.Fatalf("unexpected configured policy: %q", got)
	}
	if got := supportmath.ActiveDecimalPolicy().CanonicalString(); got != "scale=4,rounding=half_up" {
		t.Fatalf("unexpected active decimal policy: %q", got)
	}
}

func TestConfigureProcessDecimalPolicyRejectsInvalidEnvironmentOverride(t *testing.T) {
	var previousLookupEnv = lookupEnv
	defer func() {
		lookupEnv = previousLookupEnv
	}()
	t.Cleanup(func() {
		if resetErr := supportmath.SetActiveDecimalPolicy(supportmath.DefaultDecimalPolicy()); resetErr != nil {
			t.Fatalf("reset active decimal policy: %v", resetErr)
		}
	})

	var customPolicy, err = supportmath.ParseDecimalPolicy("scale=4,rounding=half_up")
	if err != nil {
		t.Fatalf("parse custom decimal policy: %v", err)
	}
	if err = supportmath.SetActiveDecimalPolicy(customPolicy); err != nil {
		t.Fatalf("set active decimal policy: %v", err)
	}

	lookupEnv = func(string) (string, bool) {
		return "scale=65,rounding=half_up", true
	}

	_, err = ConfigureProcessDecimalPolicy()
	if err == nil {
		t.Fatalf("expected decimal-policy startup error")
	}
	if got := err.Error(); got == "" || !supporttext.ContainsAll(got, reportDecimalPolicyEnvironmentVariable, "scale=65,rounding=half_up", "exceeds maximum supported scale 64") {
		t.Fatalf("unexpected decimal-policy startup error: %v", err)
	}
	if got := supportmath.ActiveDecimalPolicy().CanonicalString(); got != "scale=4,rounding=half_up" {
		t.Fatalf("expected active decimal policy to stay unchanged after failure, got %q", got)
	}
}

func TestConfigureProcessDecimalPolicyPropagatesActivationError(t *testing.T) {
	var previousLookupEnv = lookupEnv
	var previousSetActiveDecimalPolicy = setActiveDecimalPolicy
	defer func() {
		lookupEnv = previousLookupEnv
		setActiveDecimalPolicy = previousSetActiveDecimalPolicy
	}()

	lookupEnv = func(string) (string, bool) {
		return "", false
	}
	setActiveDecimalPolicy = func(policy supportmath.DecimalPolicy) error {
		if got := policy.CanonicalString(); got != supportmath.DefaultDecimalPolicy().CanonicalString() {
			t.Fatalf("unexpected activation policy: %q", got)
		}

		return errors.New("activation failed")
	}

	_, err := ConfigureProcessDecimalPolicy()
	if err == nil {
		t.Fatalf("expected decimal-policy activation error")
	}
	if got := err.Error(); got == "" || !supporttext.ContainsAll(got, reportDecimalPolicyEnvironmentVariable, "activation failed") {
		t.Fatalf("unexpected decimal-policy activation error: %v", err)
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
	if !state.NeedsSetup || state.SetupRequirementReason != SetupRequirementInvalidRememberedSetup {
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
	if !state.NeedsSetup || state.SetupRequirementReason != SetupRequirementMissing {
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
