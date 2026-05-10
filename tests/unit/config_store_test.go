package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

func TestJSONStoreSaveLoadDelete(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = configstore.NewJSONStore(tempDir)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save: %v", err)
	}

	var loaded configmodel.AppSetupConfig
	loaded, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ServerOrigin != config.ServerOrigin {
		t.Fatalf("server origin mismatch: got %q want %q", loaded.ServerOrigin, config.ServerOrigin)
	}

	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Load(context.Background()); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

func TestJSONStoreUsesApplicationPath(t *testing.T) {
	t.Parallel()

	var store = configstore.NewJSONStore("/tmp/example")
	var expected = filepath.Join("/tmp/example", "ghostfolio-cryptogains", "setup.json")
	if store.Path() != expected {
		t.Fatalf("path mismatch: got %q want %q", store.Path(), expected)
	}
}

func TestJSONStoreCreatesParentDirectory(t *testing.T) {
	t.Parallel()

	var tempDir = t.TempDir()
	var store = configstore.NewJSONStore(tempDir)
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(store.Path())); err != nil {
		t.Fatalf("expected parent directory to exist: %v", err)
	}
}
