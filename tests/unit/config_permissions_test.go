package unit

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
)

func TestJSONStoreUsesRestrictiveFilePermissionsWhereSupported(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on windows")
	}

	var store = configstore.NewJSONStore(t.TempDir())
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save: %v", err)
	}

	var info os.FileInfo
	info, err = os.Stat(store.Path())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions mismatch: got %#o want %#o", info.Mode().Perm(), 0o600)
	}
}
