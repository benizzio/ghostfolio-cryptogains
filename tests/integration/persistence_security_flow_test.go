// Package integration verifies black-box workflow behavior for the current
// slice, including persisted-artifact security checks across setup, snapshot,
// and diagnostic-report files.
// Authored by: OpenCode
package integration

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
)

const (
	persistedArtifactTokenSentinel = "TOKEN_SHOULD_NEVER_APPEAR_IN_PERSISTED_ARTIFACTS_20260516"
	rawPayloadSentinelFragment     = `"type":"TRANSFER","quantity":123,"valueInBaseCurrency":456,"unitPriceInAssetProfileCurrency":789`
)

var transientResultSentinels = []string{
	"Sync and secure storage did not succeed.",
	"You can generate a synced-data diagnostic report for this failure from this screen.",
	"The retrieved activity history is not supported safely by this slice, so no protected data was stored.",
}

var productionDiagnosticFinancialSentinels = []string{
	`"quantity":`,
	`"unit_price":`,
	`"gross_value":`,
	`"fee_amount":`,
}

// TestPersistenceSecurityFlowArtifactsStayFreeOfSecretsAndTransientFailureText
// verifies that bootstrap files, protected snapshots, and production diagnostic
// reports omit tokens, raw payload fragments, transient result text, and
// production-disallowed financial-value fields.
// Authored by: OpenCode
func TestPersistenceSecurityFlowArtifactsStayFreeOfSecretsAndTransientFailureText(t *testing.T) {
	t.Parallel()

	var baseDir = t.TempDir()
	var setupStore = configstore.NewJSONStore(baseDir)

	var successServer = newGhostfolioStorageTLSServer(t, []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	var successConfig = mustProductionArtifactConfig(t, successServer.URL)
	if err := setupStore.Save(context.Background(), successConfig); err != nil {
		t.Fatalf("save success setup config: %v", err)
	}

	var successService = newProductionArtifactSyncService(baseDir, successServer.Client())
	var outcome = successService.Validate(context.Background(), runtime.ValidateRequest{
		Config:        successConfig,
		SecurityToken: persistedArtifactTokenSentinel,
	})
	if !outcome.Success {
		t.Fatalf("expected successful sync before artifact inspection, got %#v", outcome)
	}

	assertDecryptedSnapshotOmitsForbiddenText(t, baseDir, successServer.URL, persistedArtifactTokenSentinel)

	var failureServer = newGhostfolioStorageTLSServer(t, []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"unsupported-1","date":"2024-01-02T10:00:00Z","type":"TRANSFER","quantity":123,"valueInBaseCurrency":456,"unitPriceInAssetProfileCurrency":789,"comment":"RAW_NOT_PERSISTED","SymbolProfile":{"symbol":"BTC","name":"Bitcoin"}}]`,
	}})
	var failureConfig = mustProductionArtifactConfig(t, failureServer.URL)
	if err := setupStore.Save(context.Background(), failureConfig); err != nil {
		t.Fatalf("save failure setup config: %v", err)
	}

	var failureService = newProductionArtifactSyncService(baseDir, failureServer.Client())
	outcome = failureService.Validate(context.Background(), runtime.ValidateRequest{
		Config:        failureConfig,
		SecurityToken: persistedArtifactTokenSentinel,
	})
	if outcome.FailureReason != runtime.SyncFailureUnsupportedActivityHistory {
		t.Fatalf("expected unsupported activity history failure, got %#v", outcome)
	}
	if !outcome.Diagnostic.Eligible {
		t.Fatalf("expected production failure to offer a diagnostic report, got %#v", outcome)
	}

	var reportPath, err = failureService.GenerateDiagnosticReport(context.Background(), outcome.Diagnostic.Request)
	if err != nil {
		t.Fatalf("generate diagnostic report: %v", err)
	}
	var reportBytes, readErr = os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("read diagnostic report: %v", readErr)
	}
	var reportText = string(reportBytes)
	for _, forbidden := range productionDiagnosticFinancialSentinels {
		assertTextOmitted(t, reportText, forbidden, "production diagnostic report")
	}

	for _, path := range mustPersistedArtifactPaths(t, baseDir) {
		var rawBytes, artifactErr = os.ReadFile(path)
		if artifactErr != nil {
			t.Fatalf("read persisted artifact %q: %v", path, artifactErr)
		}
		var rawText = string(rawBytes)
		assertTextOmitted(t, rawText, persistedArtifactTokenSentinel, path)
		assertTextOmitted(t, rawText, rawPayloadSentinelFragment, path)
		for _, forbidden := range transientResultSentinels {
			assertTextOmitted(t, rawText, forbidden, path)
		}
		for _, forbidden := range productionDiagnosticFinancialSentinels {
			assertTextOmitted(t, rawText, forbidden, path)
		}
	}
}

// newProductionArtifactSyncService creates one HTTPS-only sync service for
// persisted-artifact security verification.
// Authored by: OpenCode
func newProductionArtifactSyncService(baseDir string, client *http.Client) runtime.SyncService {
	return runtime.NewSyncService(
		ghostfolioclient.New(client),
		5*time.Second,
		baseDir,
		false,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
}

// mustProductionArtifactConfig builds one production-mode bootstrap setup for
// artifact security verification.
// Authored by: OpenCode
func mustProductionArtifactConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	return config
}

// assertDecryptedSnapshotOmitsForbiddenText reads the stored protected snapshot
// and verifies that its decrypted payload does not persist secrets or transient
// screen text.
// Authored by: OpenCode
func assertDecryptedSnapshotOmitsForbiddenText(t *testing.T, baseDir string, serverOrigin string, securityToken string) {
	t.Helper()

	var inspector = snapshotstore.NewEncryptedStore(baseDir, nil)
	var candidates, err = snapshotstore.DiscoverServerCandidates(context.Background(), inspector, serverOrigin)
	if err != nil {
		t.Fatalf("discover snapshot candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one protected snapshot, got %d", len(candidates))
	}

	var payload, readErr = inspector.Read(context.Background(), snapshotstore.ReadRequest{
		Candidate:     candidates[0],
		SecurityToken: securityToken,
	})
	if readErr != nil {
		t.Fatalf("read protected snapshot payload: %v", readErr)
	}

	var payloadBytes, marshalErr = json.Marshal(payload)
	if marshalErr != nil {
		t.Fatalf("marshal protected snapshot payload: %v", marshalErr)
	}
	var payloadText = string(payloadBytes)
	assertTextOmitted(t, payloadText, securityToken, "decrypted snapshot payload")
	assertTextOmitted(t, payloadText, rawPayloadSentinelFragment, "decrypted snapshot payload")
	for _, forbidden := range transientResultSentinels {
		assertTextOmitted(t, payloadText, forbidden, "decrypted snapshot payload")
	}
}

// mustPersistedArtifactPaths enumerates the plaintext application artifacts
// written during the current integration scenario.
// Authored by: OpenCode
func mustPersistedArtifactPaths(t *testing.T, baseDir string) []string {
	t.Helper()

	var root = filepath.Join(baseDir, "ghostfolio-cryptogains")
	var paths []string
	var err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk persisted artifacts: %v", err)
	}

	sort.Strings(paths)
	return paths
}

// assertTextOmitted fails the test when one persisted artifact contains a
// forbidden plaintext marker.
// Authored by: OpenCode
func assertTextOmitted(t *testing.T, content string, forbidden string, location string) {
	t.Helper()

	if strings.Contains(content, forbidden) {
		t.Fatalf("expected %s to omit %q, got %q", location, forbidden, content)
	}
}
