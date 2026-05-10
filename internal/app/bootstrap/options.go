// Package bootstrap contains startup configuration and bootstrap helpers for
// the terminal application.
// Authored by: OpenCode
package bootstrap

import (
	"flag"
	"fmt"
	"time"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultWindowWidth    = 100
	defaultWindowHeight   = 32
)

// Options contains process-level runtime options used during application
// bootstrap.
//
// Example:
//
//	opts, err := bootstrap.ParseOptions([]string{"--dev-mode"})
//	if err != nil {
//		panic(err)
//	}
//	_ = opts.AllowDevHTTP
//
// Authored by: OpenCode
type Options struct {
	ConfigDir           string
	AllowDevHTTP        bool
	RequestTimeout      time.Duration
	InitialWindowWidth  int
	InitialWindowHeight int
}

// DefaultOptions returns the process defaults used by the application.
//
// Example:
//
//	opts := bootstrap.DefaultOptions()
//	_ = opts.RequestTimeout
//
// Authored by: OpenCode
func DefaultOptions() Options {
	return Options{
		RequestTimeout:      defaultRequestTimeout,
		InitialWindowWidth:  defaultWindowWidth,
		InitialWindowHeight: defaultWindowHeight,
	}
}

// ParseOptions parses command-line arguments into application bootstrap
// options.
//
// Example:
//
//	opts, err := bootstrap.ParseOptions([]string{"--config-dir", "/tmp/app"})
//	if err != nil {
//		panic(err)
//	}
//	_ = opts.ConfigDir
//
// ParseOptions supports `--config-dir`, `--dev-mode`, `--request-timeout`,
// `--window-width`, and `--window-height`. The returned `Options` start from
// `DefaultOptions`, then apply any supplied overrides. Unknown flags, malformed
// durations, and non-positive request timeouts return an error without starting
// runtime assembly. `--dev-mode` only allows custom `http` origins for the
// current process and does not change any persisted setup on its own.
// Authored by: OpenCode
func ParseOptions(args []string) (Options, error) {
	var opts = DefaultOptions()
	var requestTimeout string

	var flags = flag.NewFlagSet("ghostfolio-cryptogains", flag.ContinueOnError)
	flags.StringVar(&opts.ConfigDir, "config-dir", opts.ConfigDir, "override the base config directory")
	flags.BoolVar(&opts.AllowDevHTTP, "dev-mode", opts.AllowDevHTTP, "allow http custom origins for development use")
	flags.StringVar(&requestTimeout, "request-timeout", opts.RequestTimeout.String(), "validation request timeout")
	flags.IntVar(&opts.InitialWindowWidth, "window-width", opts.InitialWindowWidth, "initial test-friendly window width")
	flags.IntVar(&opts.InitialWindowHeight, "window-height", opts.InitialWindowHeight, "initial test-friendly window height")

	var err = flags.Parse(args)
	if err != nil {
		return Options{}, err
	}

	var parsedTimeout time.Duration
	parsedTimeout, err = time.ParseDuration(requestTimeout)
	if err != nil {
		return Options{}, fmt.Errorf("parse request timeout: %w", err)
	}
	if parsedTimeout <= 0 {
		return Options{}, fmt.Errorf("request timeout must be positive")
	}

	opts.RequestTimeout = parsedTimeout
	return opts, nil
}
