package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeProgram struct {
	err error
}

type fakeModel struct{}

func (fakeModel) Init() tea.Cmd                       { return nil }
func (fakeModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return fakeModel{}, nil }
func (fakeModel) View() tea.View                      { return tea.NewView("") }

func (f fakeProgram) Run() (tea.Model, error) {
	return nil, f.err
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
	if runner := newProgram(fakeModel{}); runner == nil {
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
