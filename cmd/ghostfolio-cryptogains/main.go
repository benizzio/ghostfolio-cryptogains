// Command ghostfolio-cryptogains launches the first runnable terminal slice of
// the Ghostfolio capital-gains workflow.
// Authored by: OpenCode
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
)

// programRunner abstracts Bubble Tea program execution so tests can replace the
// real program runner without starting an interactive terminal session.
// Authored by: OpenCode
type programRunner interface {
	Run() (tea.Model, error)
}

var newProgram = func(model tea.Model, options ...tea.ProgramOption) programRunner {
	return tea.NewProgram(model, options...)
}

var stderrWriter io.Writer = os.Stderr

var exitFunc = os.Exit

// main parses options, assembles the runtime, and starts the Bubble Tea
// program.
// Authored by: OpenCode
func main() {
	var err = run()
	if err != nil {
		fmt.Fprintln(stderrWriter, err)
		exitFunc(1)
	}
}

// run starts the application runtime and returns startup errors to the caller.
// Authored by: OpenCode
func run() error {
	var options, err = bootstrap.ParseOptions(os.Args[1:])
	if err != nil {
		return err
	}

	var app *runtime.App
	app, err = runtime.New(options)
	if err != nil {
		return err
	}

	var startupState bootstrap.StartupState
	startupState, err = bootstrap.LoadStartupState(context.Background(), app.ConfigStore, app.Options.AllowDevHTTP)
	if err != nil {
		return err
	}

	var program = newProgram(
		flow.NewModel(flow.Dependencies{
			Options:      app.Options,
			Startup:      startupState,
			SetupService: app.SetupService,
			SyncService:  app.SyncService,
		}),
		tea.WithWindowSize(options.InitialWindowWidth, options.InitialWindowHeight),
	)

	_, err = program.Run()
	return err
}
