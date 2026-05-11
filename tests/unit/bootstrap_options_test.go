package unit

import (
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	var options, err = bootstrap.ParseOptions([]string{"--dev-mode", "--config-dir", "/tmp/config", "--request-timeout", "10s", "--window-width", "120", "--window-height", "40"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if !options.AllowDevHTTP {
		t.Fatalf("expected AllowDevHTTP to be true")
	}
	if options.ConfigDir != "/tmp/config" {
		t.Fatalf("unexpected ConfigDir: got %q want %q", options.ConfigDir, "/tmp/config")
	}
	if options.RequestTimeout != 10*time.Second {
		t.Fatalf("unexpected RequestTimeout: got %v want %v", options.RequestTimeout, 10*time.Second)
	}
	if options.InitialWindowWidth != 120 {
		t.Fatalf("unexpected InitialWindowWidth: got %d want %d", options.InitialWindowWidth, 120)
	}
	if options.InitialWindowHeight != 40 {
		t.Fatalf("unexpected InitialWindowHeight: got %d want %d", options.InitialWindowHeight, 40)
	}
}

func TestParseOptionsRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()

	if _, err := bootstrap.ParseOptions([]string{"--request-timeout", "0s"}); err == nil {
		t.Fatalf("expected invalid timeout error")
	}
}
