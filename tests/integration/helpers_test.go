package integration

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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
	"github.com/cockroachdb/apd/v3"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var frameCharacterPattern = regexp.MustCompile(`[╭╮╰╯│─]`)

// assertFlowModel converts the updated Bubble Tea model into the integration
// test's concrete flow model type.
// Authored by: OpenCode
func assertFlowModel(t *testing.T, updated tea.Model) *flow.Model {
	t.Helper()

	var model, ok = updated.(*flow.Model)
	if !ok {
		t.Fatalf("expected updated model to be *flow.Model, got %T", updated)
	}

	return model
}

// newFlowDependencies constructs test workflow dependencies using a temporary
// JSON-backed setup store for the current test.
// Authored by: OpenCode
func newFlowDependencies(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService) flow.Dependencies {
	t.Helper()
	return newFlowDependenciesWithStore(t, startup, allowDevHTTP, syncService, configstore.NewJSONStore(t.TempDir()))
}

// newFlowDependenciesWithStore constructs test workflow dependencies using the
// provided store and the repository's default bootstrap options.
// Authored by: OpenCode
func newFlowDependenciesWithStore(t *testing.T, startup bootstrap.StartupState, allowDevHTTP bool, syncService runtime.SyncService, store configstore.Store) flow.Dependencies {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.AllowDevHTTP = allowDevHTTP

	return flow.Dependencies{
		Options:       options,
		Startup:       startup,
		SetupService:  runtime.NewSetupService(store, allowDevHTTP),
		SyncService:   syncService,
		ReportService: nil,
	}
}

// runtimeBackedFlowHarness stores the real runtime-backed dependencies used by
// report integration tests.
// Authored by: OpenCode
type runtimeBackedFlowHarness struct {
	BaseDir string
	App     *runtime.App
	Config  configmodel.AppSetupConfig
	Store   configstore.Store
	Model   *flow.Model
}

// newRuntimeBackedFlowHarness creates one flow model wired to the real runtime
// assembly so integration tests can exercise report generation end to end.
// Authored by: OpenCode
func newRuntimeBackedFlowHarness(t *testing.T, baseDir string, config configmodel.AppSetupConfig, allowDevHTTP bool) runtimeBackedFlowHarness {
	t.Helper()

	var options = bootstrap.DefaultOptions()
	options.ConfigDir = baseDir
	options.AllowDevHTTP = allowDevHTTP

	var app = runtimeapp.NewWithReportCurrencyRateService(t, options, deterministicIntegrationCurrencyRates{})

	var store = configstore.NewJSONStore(baseDir)
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatalf("save setup config: %v", err)
	}

	var model = flow.NewModel(flow.Dependencies{
		Options:       options,
		Startup:       bootstrap.StartupState{ActiveConfig: &config},
		SetupService:  app.SetupService,
		SyncService:   app.SyncService,
		ReportService: app.ReportService,
	})

	return runtimeBackedFlowHarness{
		BaseDir: baseDir,
		App:     app,
		Config:  config,
		Store:   store,
		Model:   model,
	}
}

// deterministicIntegrationCurrencyRates returns canonical official-rate-shaped
// evidence without calling live providers.
// Authored by: OpenCode
type deterministicIntegrationCurrencyRates struct{}

// LookupRate returns deterministic evidence for runtime-backed integration
// tests.
// Authored by: OpenCode
func (service deterministicIntegrationCurrencyRates) LookupRate(_ context.Context, request currency.RateLookupRequest) (currency.ExchangeRateEvidence, error) {
	var rateDate = integrationRateDate(request.ActivityDate)
	var rateValue = *apd.New(11, -1)
	var authority = currency.RateAuthorityFederalReserve
	var providerID = currency.ProviderIDFederalReserveH10
	var rateKind = currency.RateKindFederalReserveH10NoonBuying
	var quoteDirection = currency.QuoteDirectionSourcePerBase
	var datasetReference = "integration deterministic Federal Reserve H.10 fixture"

	if request.BaseCurrency == currency.BaseCurrencyEUR {
		authority = currency.RateAuthorityEuropeanCentralBank
		providerID = currency.ProviderIDECBEXR
		rateKind = currency.RateKindECBEXRDailyReference
		datasetReference = "EXR/D." + request.SourceCurrency + ".EUR.SP00.A integration deterministic fixture"
	}
	if request.BaseCurrency == currency.BaseCurrencyUSD && request.SourceCurrency == currency.BaseCurrencyEUR {
		quoteDirection = currency.QuoteDirectionBasePerSource
	}

	return currency.NewExchangeRateEvidence(request, rateDate, authority, providerID, rateKind, quoteDirection, rateValue, datasetReference)
}

// ProviderCategoryForBaseCurrency returns deterministic provider metadata for
// runtime-backed integration tests.
// Authored by: OpenCode
func (service deterministicIntegrationCurrencyRates) ProviderCategoryForBaseCurrency(baseCurrency string) string {
	switch baseCurrency {
	case currency.BaseCurrencyEUR:
		return string(currency.ProviderIDECBEXR)
	case currency.BaseCurrencyUSD:
		return string(currency.ProviderIDFederalReserveH10)
	default:
		return ""
	}
}

// integrationRateDate returns the deterministic prior-provider date used by
// conversion integration fixtures.
// Authored by: OpenCode
func integrationRateDate(activityDate time.Time) time.Time {
	var candidate = time.Date(activityDate.Year(), activityDate.Month(), activityDate.Day(), 0, 0, 0, 0, time.UTC)
	if candidate.Format(time.DateOnly) == "2024-01-06" {
		return candidate.AddDate(0, 0, -1)
	}

	return candidate
}

// seedProtectedSnapshot persists one encrypted snapshot for the supplied token
// and protected cache so runtime-backed integration tests can unlock it.
// Authored by: OpenCode
func seedProtectedSnapshot(t *testing.T, harness runtimeBackedFlowHarness, token string, cache syncmodel.ProtectedActivityCache) snapshotstore.Candidate {
	t.Helper()

	var syncedAt = cache.SyncedAt
	if syncedAt.IsZero() {
		syncedAt = time.Date(2026, time.May, 21, 11, 0, 0, 0, time.UTC)
	}

	var candidate, err = harness.App.SnapshotStore.Write(context.Background(), snapshotstore.WriteRequest{
		SecurityToken: token,
		ServerOrigin:  harness.Config.ServerOrigin,
		Payload: snapshotmodel.Payload{
			StoredDataVersion: snapshotmodel.DefaultStoredDataVersion(""),
			RegisteredLocalUser: snapshotmodel.RegisteredLocalUser{
				LocalUserID:          "integration-user",
				CreatedAt:            syncedAt,
				UpdatedAt:            syncedAt,
				LastSuccessfulSyncAt: syncedAt,
			},
			SetupProfile: snapshotmodel.SetupProfile{
				ServerOrigin:      harness.Config.ServerOrigin,
				ServerMode:        harness.Config.ServerMode,
				AllowDevHTTP:      harness.Config.AllowDevHTTP,
				LastValidatedAt:   harness.Config.UpdatedAt,
				SourceAPIBasePath: "api/v1",
			},
			ProtectedActivityCache: cache,
		},
	})
	if err != nil {
		t.Fatalf("seed protected snapshot: %v", err)
	}

	return candidate
}

// unlockSyncReportsContext enters the runtime-backed Sync and Reports context
// and unlocks it with the supplied token.
// Authored by: OpenCode
func unlockSyncReportsContext(t *testing.T, model *flow.Model, token string) *flow.Model {
	t.Helper()

	model = openSyncEntry(t, model)
	model = typeToken(t, model, token)
	model = blurTokenInputFromSyncEntry(t, model)

	var updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync reports menu after unlock, got %s", model.ActiveScreen())
	}

	return model
}

// openReportSelectionFromContext routes the unlocked context into report
// selection.
// Authored by: OpenCode
func openReportSelectionFromContext(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)

	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected report selection screen, got %s", model.ActiveScreen())
	}

	return model
}

// applyBatchCmd runs one Bubble Tea batch command to completion and applies its
// resulting messages to the model.
// Authored by: OpenCode
func applyBatchCmd(t *testing.T, model *flow.Model, cmd tea.Cmd) *flow.Model {
	t.Helper()

	var message = testutil.RunCmd(cmd)
	var batch, ok = message.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected batch command, got %T", message)
	}

	for _, batchCmd := range batch {
		var batchMessage = testutil.RunCmd(batchCmd)
		if batchMessage == nil {
			continue
		}

		var updated, _ = model.Update(batchMessage)
		model = assertFlowModel(t, updated)
	}

	return model
}

// selectReportYear moves the selection cursor to the provided year.
// Authored by: OpenCode
func selectReportYear(t *testing.T, model *flow.Model, year int) *flow.Model {
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

// selectReportMethod moves the method selection cursor to the provided visible
// method label while the report selection screen keeps method focus.
// Authored by: OpenCode
func selectReportMethod(t *testing.T, model *flow.Model, methodLabel string) *flow.Model {
	t.Helper()

	var updated tea.Model
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = assertFlowModel(t, updated)

	var marker = "> " + methodLabel
	for attempt := 0; attempt < 32; attempt++ {
		var content = normalizeRenderedText(model.View().Content)
		if strings.Contains(content, marker) {
			return model
		}

		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = assertFlowModel(t, updated)
	}

	t.Fatalf("expected report method %q to be selected, got %q", methodLabel, model.View().Content)
	return model
}

// startReportGenerationFromSelection advances focus to the action menu and
// starts one report-generation attempt.
// Authored by: OpenCode
func startReportGenerationFromSelection(t *testing.T, model *flow.Model) (*flow.Model, tea.Cmd) {
	t.Helper()

	for attempt := 0; attempt < 3; attempt++ {
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

// mustMarkdownFiles returns generated main Markdown report files in one
// directory.
// Authored by: OpenCode
func mustMarkdownFiles(t *testing.T, dir string) []string {
	t.Helper()

	var files = mustAllMarkdownFiles(t, dir)
	var mainFiles []string
	for _, file := range files {
		if strings.Contains(filepath.Base(file), "-annex-1-") {
			continue
		}
		mainFiles = append(mainFiles, file)
	}

	return mainFiles
}

// mustAllMarkdownFiles returns every generated Markdown file in one directory.
// Authored by: OpenCode
func mustAllMarkdownFiles(t *testing.T, dir string) []string {
	t.Helper()

	var entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files
}

// assertNoCleartextReportInAppStorage verifies that app-managed storage does not
// contain persisted Markdown report content.
// Authored by: OpenCode
func assertNoCleartextReportInAppStorage(t *testing.T, baseDir string) {
	t.Helper()

	for _, path := range mustPersistedArtifactPaths(t, baseDir) {
		if strings.HasSuffix(path, ".md") {
			t.Fatalf("expected no Markdown file in app-managed storage, found %q", path)
		}

		//nolint:gosec // Test scans paths returned by the fixture artifact walker.
		var rawBytes, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("read persisted artifact %q: %v", path, err)
		}
		assertTextOmitted(t, string(rawBytes), "# Ghostfolio Capital Gains And Losses Report", path)
	}
}

// installOpenCommandRecorder places one platform opener stub on PATH and
// records each requested report path in a log file.
// Authored by: OpenCode
func installOpenCommandRecorder(t *testing.T, exitCode int) string {
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
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatalf("mkdir opener bin dir: %v", err)
	}

	var logPath = filepath.Join(fixtureDir, "open.log")
	var scriptPath = filepath.Join(binDir, commandName)
	var script = "#!/bin/sh\n" +
		"printf '%s\\n' \"$1\" >> \"" + logPath + "\"\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	//nolint:gosec // The opener stub must be executable so PATH lookup can run it.
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write opener stub: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// readOpenCommandRequests returns the saved-path requests recorded by the
// platform opener stub.
// Authored by: OpenCode
func readOpenCommandRequests(t *testing.T, logPath string) []string {
	t.Helper()

	//nolint:gosec // Test reads the opener log path returned by installOpenCommandRecorder.
	var raw, err = os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read opener log %q: %v", logPath, err)
	}

	var content = strings.TrimSpace(string(raw))
	if content == "" {
		return nil
	}

	return strings.Split(content, "\n")
}

// normalizeRenderedText removes ANSI styling and collapses wrapped whitespace so
// integration assertions can focus on stable visible text.
// Authored by: OpenCode
func normalizeRenderedText(content string) string {
	var stripped = ansiEscapePattern.ReplaceAllString(content, "")
	stripped = frameCharacterPattern.ReplaceAllString(stripped, " ")
	return strings.Join(strings.Fields(stripped), " ")
}

// mustCloudSetupConfig returns a valid remembered Ghostfolio Cloud setup for
// integration tests that start from the main menu.
// Authored by: OpenCode
func mustCloudSetupConfig(t *testing.T) configmodel.AppSetupConfig {
	t.Helper()

	var config, err = configmodel.NewSetupConfig(configmodel.ServerModeGhostfolioCloud, configmodel.GhostfolioCloudOrigin, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}

	return config
}

func TestMustCloudSetupConfigReturnsValidCloudSetup(t *testing.T) {
	t.Parallel()

	var config = mustCloudSetupConfig(t)
	if config.ServerMode != configmodel.ServerModeGhostfolioCloud {
		t.Fatalf("unexpected server mode: %q", config.ServerMode)
	}
	if config.ServerOrigin != configmodel.GhostfolioCloudOrigin {
		t.Fatalf("unexpected server origin: %q", config.ServerOrigin)
	}
}

// setTokenAwareCurrencyContextFixtures applies one reusable BUG-004 user/body
// permutation to the shared token-aware Ghostfolio test server.
// Authored by: OpenCode
func setTokenAwareCurrencyContextFixtures(server *tokenAwareStorageServer, token string, userBody string, activities ...string) {
	if userBody == "" {
		userBody = testutil.GhostfolioUserBody("USD")
	}
	server.SetTokenUserBody(token, userBody)
	server.SetTokenPages(token, []storagePageFixture{{
		Count:          len(activities),
		ActivitiesJSON: "[" + strings.Join(activities, ",") + "]",
	}})
}
