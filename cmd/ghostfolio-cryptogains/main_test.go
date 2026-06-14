package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
)

type fakeProgram struct {
	err error
}

type failingWriter struct{}

type fakeModel struct{}

func (fakeModel) Init() tea.Cmd                       { return nil }
func (fakeModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return fakeModel{}, nil }
func (fakeModel) View() tea.View                      { return tea.NewView("") }

func (f fakeProgram) Run() (tea.Model, error) {
	return nil, f.err
}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write boom")
}

func TestRunUsesInjectedProgramRunner(t *testing.T) {
	var previous = newProgram
	defer func() { newProgram = previous }()

	newProgram = func(model tea.Model, options ...tea.ProgramOption) programRunner {
		return fakeProgram{}
	}

	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", t.TempDir()}

	if err := run(); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
}

func TestDefaultNewProgramReturnsRunner(t *testing.T) {
	var previous = newProgram
	defer func() { newProgram = previous }()

	newProgram = previous
	var runner = newProgram(fakeModel{})
	if runner == nil {
		t.Fatalf("expected program runner")
	}
}

func TestRunReturnsProgramError(t *testing.T) {
	var previous = newProgram
	defer func() { newProgram = previous }()

	newProgram = func(model tea.Model, options ...tea.ProgramOption) programRunner {
		return fakeProgram{err: errors.New("boom")}
	}

	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", t.TempDir()}

	if err := run(); err == nil {
		t.Fatalf("expected run error")
	}
}

func TestRunAppliesDecimalPolicyOverrideFromEnvironment(t *testing.T) {
	var previous = newProgram
	defer func() { newProgram = previous }()
	t.Cleanup(func() {
		if err := supportmath.SetActiveDecimalPolicy(supportmath.DefaultDecimalPolicy()); err != nil {
			t.Fatalf("reset active decimal policy: %v", err)
		}
	})

	newProgram = func(model tea.Model, options ...tea.ProgramOption) programRunner {
		return fakeProgram{}
	}
	t.Setenv("GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY", "scale=4,rounding=half_up")

	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", t.TempDir()}

	if err := run(); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
	if got := supportmath.ActiveDecimalPolicy().CanonicalString(); got != "scale=4,rounding=half_up" {
		t.Fatalf("unexpected active decimal policy after startup: %q", got)
	}
}

func TestRunResetsDecimalPolicyToDefaultWhenEnvironmentUnset(t *testing.T) {
	var previous = newProgram
	defer func() { newProgram = previous }()

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

	newProgram = func(model tea.Model, options ...tea.ProgramOption) programRunner {
		return fakeProgram{}
	}

	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", t.TempDir()}

	if err = run(); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}
	if got := supportmath.ActiveDecimalPolicy().CanonicalString(); got != supportmath.DefaultDecimalPolicy().CanonicalString() {
		t.Fatalf("unexpected active decimal policy after startup: %q", got)
	}
}

func TestRunReturnsDecimalPolicyConfigurationError(t *testing.T) {
	t.Cleanup(func() {
		if err := supportmath.SetActiveDecimalPolicy(supportmath.DefaultDecimalPolicy()); err != nil {
			t.Fatalf("reset active decimal policy: %v", err)
		}
	})
	t.Setenv("GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY", "scale=65,rounding=half_up")

	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", t.TempDir()}

	var err = run()
	if err == nil {
		t.Fatalf("expected decimal-policy startup error")
	}
	if got := err.Error(); got == "" || !bytes.Contains([]byte(got), []byte("GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY")) || !bytes.Contains([]byte(got), []byte("exceeds maximum supported scale 64")) {
		t.Fatalf("unexpected decimal-policy startup error: %v", err)
	}
}

func TestRunReturnsParseError(t *testing.T) {
	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains", "--unknown-flag"}

	if err := run(); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestRunReturnsRuntimeAssemblyError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()
	os.Args = []string{"ghostfolio-cryptogains"}

	if err := run(); err == nil {
		t.Fatalf("expected runtime assembly error")
	}
}

func TestRunReturnsStartupLoadError(t *testing.T) {
	var previousArgs = os.Args
	defer func() { os.Args = previousArgs }()

	var configDir = t.TempDir()
	var setupPath = filepath.Join(configDir, "ghostfolio-cryptogains", "setup.json")
	if err := os.MkdirAll(setupPath, 0o700); err != nil {
		t.Fatalf("mkdir setup path: %v", err)
	}
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", configDir}

	if err := run(); err == nil {
		t.Fatalf("expected startup load error")
	}
}

func TestMainWritesErrorAndExits(t *testing.T) {
	var previousWriter = stderrWriter
	var previousExit = exitFunc
	var previousArgs = os.Args
	defer func() {
		stderrWriter = previousWriter
		exitFunc = previousExit
		os.Args = previousArgs
	}()

	var stderr bytes.Buffer
	var exitCode int
	stderrWriter = &stderr
	exitFunc = func(code int) { exitCode = code }
	os.Args = []string{"ghostfolio-cryptogains", "--unknown-flag"}

	main()

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected error to be written to stderr")
	}
}

func TestMainFallsBackToOSStderrWhenConfiguredWriterFails(t *testing.T) {
	var previousWriter = stderrWriter
	var previousExit = exitFunc
	var previousArgs = os.Args
	defer func() {
		stderrWriter = previousWriter
		exitFunc = previousExit
		os.Args = previousArgs
	}()

	var exitCode int
	stderrWriter = failingWriter{}
	exitFunc = func(code int) { exitCode = code }
	os.Args = []string{"ghostfolio-cryptogains", "--unknown-flag"}

	main()

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestMainReturnsWithoutExitOnSuccess(t *testing.T) {
	var previousProgram = newProgram
	var previousWriter = stderrWriter
	var previousExit = exitFunc
	var previousArgs = os.Args
	defer func() {
		newProgram = previousProgram
		stderrWriter = previousWriter
		exitFunc = previousExit
		os.Args = previousArgs
	}()

	var stderr bytes.Buffer
	var exited bool
	newProgram = func(model tea.Model, options ...tea.ProgramOption) programRunner {
		return fakeProgram{}
	}
	stderrWriter = &stderr
	exitFunc = func(code int) { exited = true }
	os.Args = []string{"ghostfolio-cryptogains", "--config-dir", t.TempDir()}

	main()

	if exited {
		t.Fatalf("expected main not to exit on success")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected stderr to stay empty, got %q", stderr.String())
	}
}
