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
	if !options.AllowDevHTTP || options.ConfigDir != "/tmp/config" || options.RequestTimeout != 10*time.Second || options.InitialWindowWidth != 120 || options.InitialWindowHeight != 40 {
		t.Fatalf("parsed options mismatch: %#v", options)
	}
}

func TestParseOptionsRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()

	if _, err := bootstrap.ParseOptions([]string{"--request-timeout", "0s"}); err == nil {
		t.Fatalf("expected invalid timeout error")
	}
}
