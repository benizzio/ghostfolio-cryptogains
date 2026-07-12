// Package runtimeflow provides reusable runtime-backed black-box fixtures for
// repository test suites.
//
// Authored by: OpenCode
package runtimeflow

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/cockroachdb/apd/v3"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	configstore "github.com/benizzio/ghostfolio-cryptogains/internal/config/store"
	"github.com/benizzio/ghostfolio-cryptogains/internal/integration/currency"
	snapshotmodel "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/model"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeapp"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var frameCharacterPattern = regexp.MustCompile(`[╭╮╰╯│─]`)

// RuntimeBackedFlowHarness groups the real application services and TUI model
// used by black-box report workflows. For example, construct it with
// NewRuntimeBackedFlowHarness, seed a snapshot, then drive Model through a report flow.
// Authored by: OpenCode
type RuntimeBackedFlowHarness struct {
	BaseDir string
	App     *runtime.App
	Config  configmodel.AppSetupConfig
	Store   configstore.Store
	Model   *flow.Model
}

// NewRuntimeBackedFlowHarness creates an application-backed TUI harness with
// deterministic currency-rate evidence. For example, pass a temporary base
// directory and MustCloudSetupConfig(t) before seeding a protected snapshot.
// Authored by: OpenCode
func NewRuntimeBackedFlowHarness(t *testing.T, baseDir string, config configmodel.AppSetupConfig, allowDevHTTP bool) RuntimeBackedFlowHarness {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.ConfigDir = baseDir
	options.AllowDevHTTP = allowDevHTTP
	var app = runtimeapp.NewWithReportCurrencyRateService(t, options, deterministicCurrencyRates{})
	var store = configstore.NewJSONStore(baseDir)
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save setup config: %v", err)
	}
	var model = flow.NewModel(flow.Dependencies{Options: options, Startup: bootstrap.StartupState{ActiveConfig: &config}, SetupService: app.SetupService, SyncService: app.SyncService, ReportService: app.ReportService})
	return RuntimeBackedFlowHarness{BaseDir: baseDir, App: app, Config: config, Store: store, Model: model}
}

type deterministicCurrencyRates struct{}

func (deterministicCurrencyRates) LookupRate(_ context.Context, request currency.RateLookupRequest) (currency.ExchangeRateEvidence, error) {
	var rateDate = time.Date(request.ActivityDate.Year(), request.ActivityDate.Month(), request.ActivityDate.Day(), 0, 0, 0, 0, time.UTC)
	if rateDate.Format(time.DateOnly) == "2024-01-06" {
		rateDate = rateDate.AddDate(0, 0, -1)
	}
	var authority = currency.RateAuthorityFederalReserve
	var providerID = currency.ProviderIDFederalReserveH10
	var rateKind = currency.RateKindFederalReserveH10NoonBuying
	var quoteDirection = currency.QuoteDirectionSourcePerBase
	var reference = "integration deterministic Federal Reserve H.10 fixture"
	if request.BaseCurrency == currency.BaseCurrencyEUR {
		authority = currency.RateAuthorityEuropeanCentralBank
		providerID = currency.ProviderIDECBEXR
		rateKind = currency.RateKindECBEXRDailyReference
		reference = "EXR/D." + request.SourceCurrency + ".EUR.SP00.A integration deterministic fixture"
	}
	if request.BaseCurrency == currency.BaseCurrencyUSD && request.SourceCurrency == currency.BaseCurrencyEUR {
		quoteDirection = currency.QuoteDirectionBasePerSource
	}
	return currency.NewExchangeRateEvidence(request, rateDate, authority, providerID, rateKind, quoteDirection, *apd.New(11, -1), reference)
}

func (deterministicCurrencyRates) ProviderCategoryForBaseCurrency(baseCurrency string) string {
	if baseCurrency == currency.BaseCurrencyEUR {
		return string(currency.ProviderIDECBEXR)
	}
	if baseCurrency == currency.BaseCurrencyUSD {
		return string(currency.ProviderIDFederalReserveH10)
	}
	return ""
}

// SeedProtectedSnapshot writes a protected cache that the harness can unlock.
// For example, call SeedProtectedSnapshot(t, harness, token, cache) before
// driving the Sync and Reports workflow.
// Authored by: OpenCode
func SeedProtectedSnapshot(t *testing.T, harness RuntimeBackedFlowHarness, token string, cache syncmodel.ProtectedActivityCache) snapshotstore.Candidate {
	t.Helper()
	var syncedAt = cache.SyncedAt
	if syncedAt.IsZero() {
		syncedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	}
	var candidate, err = harness.App.SnapshotStore.Write(context.Background(), snapshotstore.WriteRequest{SecurityToken: token, ServerOrigin: harness.Config.ServerOrigin, Payload: snapshotmodel.Payload{StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""), RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{LocalUserID: "integration-user", CreatedAt: syncedAt, UpdatedAt: syncedAt, LastSuccessfulSyncAt: syncedAt}, SetupProfile: snapshotmodel.SetupProfile{ServerOrigin: harness.Config.ServerOrigin, ServerMode: harness.Config.ServerMode, AllowDevHTTP: harness.Config.AllowDevHTTP, LastValidatedAt: harness.Config.UpdatedAt, SourceAPIBasePath: "api/v1"}, ProtectedActivityCache: cache}})
	if err != nil {
		t.Fatalf("seed protected snapshot: %v", err)
	}
	return candidate
}

// UnlockSyncReportsContext unlocks the supplied model into the Sync and Reports menu.
// Authored by: OpenCode
func UnlockSyncReportsContext(t *testing.T, model *flow.Model, token string) *flow.Model {
	t.Helper()
	var updated tea.Model
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected sync reports unlock screen, got %s", model.ActiveScreen())
	}
	// The visible token input accepts text through its dedicated input model.
	for _, character := range token {
		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Text: string(character), Code: character}))
		model = assertFlowModel(t, updated)
	}
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = assertFlowModel(t, updated)
	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync reports menu after unlock, got %s", model.ActiveScreen())
	}
	return model
}

// OpenReportSelection opens report selection from an unlocked context.
// Authored by: OpenCode
func OpenReportSelection(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	var updated tea.Model
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected report selection screen, got %s", model.ActiveScreen())
	}
	return model
}

// SelectReportYear moves report selection to year.
// Authored by: OpenCode
func SelectReportYear(t *testing.T, model *flow.Model, year int) *flow.Model {
	t.Helper()
	var marker = "> " + strconv.Itoa(year)
	for attempt := 0; attempt < 32; attempt++ {
		if strings.Contains(model.View().Content, marker) {
			return model
		}
		var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = assertFlowModel(t, updated)
	}
	t.Fatalf("expected report year %d to be selected, got %q", year, model.View().Content)
	return model
}

// StartReportGeneration starts report generation after a report base currency is selected.
// Authored by: OpenCode
func StartReportGeneration(t *testing.T, model *flow.Model) (*flow.Model, tea.Cmd) {
	t.Helper()
	for attempt := 0; attempt < 4; attempt++ {
		var updated tea.Model
		var cmd tea.Cmd
		updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		model = assertFlowModel(t, updated)
		if model.ActiveScreen() == "report_busy" {
			return model, cmd
		}
	}
	t.Fatalf("expected report busy screen, got %s", model.ActiveScreen())
	return model, nil
}

// ApplyBatchCmd completes a Bubble Tea batch command against model.
// Authored by: OpenCode
func ApplyBatchCmd(t *testing.T, model *flow.Model, cmd tea.Cmd) *flow.Model {
	t.Helper()
	var message = testutil.RunCmd(cmd)
	var batch, ok = message.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batch command, got %T", message)
	}
	for _, batchCmd := range batch {
		if batchMessage := testutil.RunCmd(batchCmd); batchMessage != nil {
			var updated tea.Model
			updated, _ = model.Update(batchMessage)
			model = assertFlowModel(t, updated)
		}
	}
	return model
}

// NormalizeRenderedText removes presentation formatting from a rendered TUI view.
// Authored by: OpenCode
func NormalizeRenderedText(content string) string {
	return strings.Join(strings.Fields(frameCharacterPattern.ReplaceAllString(ansiEscapePattern.ReplaceAllString(content, ""), " ")), " ")
}

// MarkdownFiles returns generated Markdown files in dir.
// Authored by: OpenCode
func MarkdownFiles(t *testing.T, dir string) []string {
	t.Helper()
	var entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files
}

// InstallOpenCommandRecorder installs a local opener stub and returns its request log.
// Authored by: OpenCode
func InstallOpenCommandRecorder(t *testing.T, exitCode int) string {
	t.Helper()
	var commandName string
	switch stdruntime.GOOS {
	case "linux":
		commandName = "xdg-open"
	case "darwin":
		commandName = "open"
	default:
		t.Skipf("automatic-open integration is unsupported on %s", stdruntime.GOOS)
	}
	var fixtureDir = t.TempDir()
	var binDir = filepath.Join(fixtureDir, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("mkdir opener bin dir: %v", err)
	}
	var logPath = filepath.Join(fixtureDir, "open.log")
	var script = "#!/bin/sh\nprintf '%s\\n' \"$1\" >> \"" + logPath + "\"\nexit " + strconv.Itoa(exitCode) + "\n"
	// #nosec G306 -- the test fixture must be executable by the current user.
	if err := os.WriteFile(filepath.Join(binDir, commandName), []byte(script), 0o700); err != nil {
		t.Fatalf("write opener stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// ReadOpenCommandRequests returns paths received by the configured opener stub.
// Authored by: OpenCode
func ReadOpenCommandRequests(t *testing.T, logPath string) []string {
	t.Helper()
	// #nosec G304 -- logPath is created by InstallOpenCommandRecorder for this test.
	var raw, err = os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read opener log %q: %v", logPath, err)
	}
	var content = strings.TrimSpace(string(raw))
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// MustCloudSetupConfig creates a valid Cloud setup fixture.
// Authored by: OpenCode
func MustCloudSetupConfig(t *testing.T) configmodel.AppSetupConfig {
	t.Helper()
	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	return config
}

// AssertNoCleartextReportInAppStorage verifies app-managed artifacts do not contain a report.
// Authored by: OpenCode
func AssertNoCleartextReportInAppStorage(t *testing.T, baseDir string) {
	t.Helper()
	var root = filepath.Join(baseDir, "ghostfolio-cryptogains")
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk persisted artifacts: %v", err)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if strings.HasSuffix(path, ".md") {
			t.Fatalf("expected no Markdown file in app-managed storage, found %q", path)
		}
		// #nosec G304 -- paths are enumerated under the test-owned temporary app directory.
		var raw, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("read persisted artifact %q: %v", path, err)
		}
		if strings.Contains(string(raw), "# Ghostfolio Capital Gains And Losses Report") {
			t.Fatalf("expected %q to omit cleartext report content", path)
		}
	}
}

func assertFlowModel(t *testing.T, updated tea.Model) *flow.Model {
	t.Helper()
	var model, ok = updated.(*flow.Model)
	if !ok {
		t.Fatalf("expected updated model to be *flow.Model, got %T", updated)
	}
	return model
}
