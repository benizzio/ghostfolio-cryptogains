// Package integration verifies black-box workflow behavior for the current
// slice, including runtime-backed report generation flows.
// Authored by: OpenCode
package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
	"github.com/cockroachdb/apd/v3"
)

// TestReportGenerationSuccessWritesMarkdownAndReturnsToUnlockedContext verifies
// one end-to-end report save through the real runtime-backed workflow.
// Authored by: OpenCode
func TestReportGenerationSuccessWritesMarkdownAndReturnsToUnlockedContext(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Saved Markdown Path: ") {
		t.Fatalf("expected saved Markdown path in result view, got %q", content)
	}
	if !strings.Contains(content, "Selected Year: 2024") || !strings.Contains(content, "Cost Basis Method: FIFO") {
		t.Fatalf("expected selected year and method in result view, got %q", content)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}

	var reportPath = files[0]
	testutil.AssertPathWithin(t, reportPath, reportIO.DocumentsDir)
	testutil.AssertRegularFile(t, reportPath)
	if !strings.HasPrefix(filepath.Base(reportPath), "ghostfolio-capital-gains-2024-fifo-") {
		t.Fatalf("expected FIFO report filename slug, got %q", filepath.Base(reportPath))
	}

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"# Ghostfolio Capital Gains And Losses Report",
		"- Year: 2024",
		"- Cost Basis Method: FIFO",
		"- Report Calculation Currency: NOT APPLICABLE",
		"## Gains-And-Losses Summary",
		"## Reference Section",
		"| Overall Yearly Net Total |",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected saved report to contain %q, got %q", expected, reportText)
		}
	}
	assertTextOmitted(t, reportText, "token-123", reportPath)
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected result dismissal to return to sync and reports menu, got %s", model.ActiveScreen())
	}
	var syncReportsContent = normalizeRenderedText(model.View().Content)
	if strings.Contains(syncReportsContent, reportPath) {
		t.Fatalf("expected no report history after result dismissal, got %q", syncReportsContent)
	}

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected unlocked context to reopen report selection without another token prompt, got %s", model.ActiveScreen())
	}
}

// TestReportGenerationOpenWarningPreservesSavedReportAndAllowsAnotherRun
// verifies that an opener failure stays non-fatal and keeps the workflow in the
// unlocked report context.
// Authored by: OpenCode
func TestReportGenerationOpenWarningPreservesSavedReportAndAllowsAnotherRun(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 7)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)

	seedProtectedSnapshot(t, harness, "token-123", fixture.ProtectedActivityCache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.PrimaryReportYear)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file after opener warning, got %#v", files)
	}
	var reportPath = files[0]
	testutil.AssertRegularFile(t, reportPath)

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Success With Warning: automatic open failed after save") {
		t.Fatalf("expected opener warning headline, got %q", content)
	}
	if !strings.Contains(content, "automatic opening failed") || !strings.Contains(content, "Open the file manually") {
		t.Fatalf("expected actionable opener warning, got %q", content)
	}

	var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = assertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if model.ActiveScreen() != "report_selection" {
		t.Fatalf("expected Generate Another Report to return to report selection, got %s", model.ActiveScreen())
	}
}

// TestReportGenerationSkipsCurrencylessOrderTier verifies the BUG-007
// regression path where later explicit-currency values remain usable when a
// higher-priority order tier carries financial values but no currency label.
// Authored by: OpenCode
func TestReportGenerationSkipsCurrencylessOrderTier(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var fixture = testutil.DeterministicReportLedgerFixture()
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var cache = fixture.ProtectedActivityCache
	var filteredActivities = cache.Activities[:0]
	for _, activity := range cache.Activities {
		if activity.SourceID == "doge-buy-2025-incomplete-001" {
			continue
		}
		filteredActivities = append(filteredActivities, activity)
	}
	cache.Activities = filteredActivities
	cache.ActivityCount = len(filteredActivities)
	cache.RetrievedCount = len(filteredActivities)

	seedProtectedSnapshot(t, harness, "token-123", cache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, fixture.CurrencylessOrderReportYear)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}
	var content = normalizeRenderedText(model.View().Content)
	if !strings.Contains(content, "Saved Markdown Path:") {
		t.Fatalf("expected successful report result, got %q", content)
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	testutil.AssertRegularFile(t, reportPath)

	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	if !strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 80 | 4000 | 0.5 | EUR |") {
		t.Fatalf("expected saved report to show the later explicit-currency asset tier, got %q", reportText)
	}
	if strings.Contains(reportText, "| sol-buy-2026-asset-tier-001 | BUY | 50 | 81 | 4050 | 1 |") {
		t.Fatalf("expected saved report to skip the currencyless order-tier monetary values, got %q", reportText)
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationRoundsSameTierUnitPriceDerivation verifies that
// repeating same-tier division succeeds and the rendered report reuses the
// rounded 16-decimal internal result without extra boundary rounding.
// Authored by: OpenCode
func TestReportGenerationRoundsSameTierUnitPriceDerivation(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var harness = newRuntimeBackedFlowHarness(t, t.TempDir(), mustCloudSetupConfig(t), false)
	var cache = roundedUnitPriceProtectedActivityCache(t)

	seedProtectedSnapshot(t, harness, "token-123", cache)

	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"| unit-buy-2024-001 | BUY | 3 | 0.3333333333333333 | 1 | 0 | USD | 1 | NOT APPLICABLE | 3 |",
		"| unit-sell-2024-001 | SELL | 1 | 1 | 1 | 0 | USD | 0.6666666666666667 | NOT APPLICABLE | 2 |",
		"| unit-sell-2024-001 | 1 | USD | 0.3333333333333333 | 1 | 0.6666666666666667 | NOT APPLICABLE |",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected rounded report to contain %q, got %q", expected, reportText)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// TestReportGenerationAfterSyncAllowsRepeatingGrossValueOnlyUnitPriceDerivation
// verifies that a raw Ghostfolio activity shape with repeating same-tier
// gross-value-only unit-price derivation survives sync-boundary validation and
// remains reportable through the runtime-backed report workflow.
// Authored by: OpenCode
func TestReportGenerationAfterSyncAllowsRepeatingGrossValueOnlyUnitPriceDerivation(t *testing.T) {
	var reportIO = testutil.NewReportIOFixture(t)
	var openLogPath = installOpenCommandRecorder(t, 0)
	var baseDir = t.TempDir()
	var server = newTokenAwareStorageServer(t)
	server.SetTokenPages("token-123", []storagePageFixture{{
		Count: 2,
		ActivitiesJSON: `[
			{"id":"repeat-buy-1","date":"2024-01-01T10:00:00Z","type":"BUY","quantity":3,"valueInBaseCurrency":1,"feeInBaseCurrency":0,"baseCurrency":"USD","SymbolProfile":{"id":"asset-repeat-report-flow-001","symbol":"RPT","name":"Repeat Asset","currency":"USD"}},
			{"id":"repeat-sell-1","date":"2024-02-01T10:00:00Z","type":"SELL","quantity":1,"valueInBaseCurrency":1,"unitPriceInAssetProfileCurrency":1,"feeInBaseCurrency":0,"baseCurrency":"USD","SymbolProfile":{"id":"asset-repeat-report-flow-001","symbol":"RPT","name":"Repeat Asset","currency":"USD"}}
		]`,
	}})

	var syncConfig = mustReportGenerationSyncConfig(t, server.URL())
	var syncService = runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		baseDir,
		true,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
	var syncOutcome = syncService.Run(context.Background(), runtime.SyncRequest{Config: syncConfig, SecurityToken: "token-123"})
	if !syncOutcome.Success {
		t.Fatalf("expected sync success for repeating gross-value-only report fixture, got %#v", syncOutcome)
	}

	var harness = newRuntimeBackedFlowHarness(t, baseDir, syncConfig, true)
	var model = unlockSyncReportsContext(t, harness.Model, "token-123")
	model = openReportSelectionFromContext(t, model)
	model = selectReportYear(t, model, 2024)
	model, cmd := startReportGenerationFromSelection(t, model)
	model = applyBatchCmd(t, model, cmd)

	if model.ActiveScreen() != "report_result" {
		t.Fatalf("expected report result screen, got %s", model.ActiveScreen())
	}

	var files = mustMarkdownFiles(t, reportIO.DocumentsDir)
	if len(files) != 1 {
		t.Fatalf("expected one saved Markdown file, got %#v", files)
	}
	var reportPath = files[0]
	var openerRequests = readOpenCommandRequests(t, openLogPath)
	if len(openerRequests) != 1 || openerRequests[0] != reportPath {
		t.Fatalf("expected one opener request for %q, got %#v", reportPath, openerRequests)
	}

	var rawReport, err = os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read saved report %q: %v", reportPath, err)
	}
	var reportText = string(rawReport)
	for _, expected := range []string{
		"| repeat-buy-1 | BUY | 3 | 0.3333333333333333 | 1 | 0 | USD | 1 | NOT APPLICABLE | 3 |",
		"| repeat-sell-1 | SELL | 1 | 1 | 1 | 0 | USD | 0.6666666666666667 | NOT APPLICABLE | 2 |",
		"| repeat-sell-1 | 1 | USD | 0.3333333333333333 | 1 | 0.6666666666666667 | NOT APPLICABLE |",
	} {
		if !strings.Contains(reportText, expected) {
			t.Fatalf("expected synced repeating-derivation report to contain %q, got %q", expected, reportText)
		}
	}
	assertNoCleartextReportInAppStorage(t, harness.BaseDir)
}

// roundedUnitPriceProtectedActivityCache builds one deterministic cache where a
// same-tier unit-price derivation requires repeating decimal rounding.
// Authored by: OpenCode
func roundedUnitPriceProtectedActivityCache(t *testing.T) syncmodel.ProtectedActivityCache {
	t.Helper()

	return syncmodel.ProtectedActivityCache{
		SyncedAt:             mustReportFixtureTime(t, "2026-05-20T15:04:05Z"),
		RetrievedCount:       2,
		ActivityCount:        2,
		AvailableReportYears: []int{2024},
		Activities: []syncmodel.ActivityRecord{
			roundedReportActivity(t, roundedReportActivityInput{
				SourceID:         "unit-buy-2024-001",
				OccurredAt:       "2024-01-01T10:00:00Z",
				ActivityType:     syncmodel.ActivityTypeBuy,
				AssetIdentityKey: "asset-unit-001",
				AssetSymbol:      "UNIT",
				AssetName:        "Unit Asset",
				Quantity:         "3",
				OrderCurrency:    "USD",
				OrderGrossValue:  "1",
				OrderFeeAmount:   "0",
			}),
			roundedReportActivity(t, roundedReportActivityInput{
				SourceID:         "unit-sell-2024-001",
				OccurredAt:       "2024-03-01T10:00:00Z",
				ActivityType:     syncmodel.ActivityTypeSell,
				AssetIdentityKey: "asset-unit-001",
				AssetSymbol:      "UNIT",
				AssetName:        "Unit Asset",
				Quantity:         "1",
				OrderCurrency:    "USD",
				OrderGrossValue:  "1",
				OrderFeeAmount:   "0",
				OrderUnitPrice:   "1",
			}),
		},
	}
}

// roundedReportActivityInput stores one compact rounded-division integration
// fixture before conversion into a normalized activity record.
// Authored by: OpenCode
type roundedReportActivityInput struct {
	SourceID         string
	OccurredAt       string
	ActivityType     syncmodel.ActivityType
	AssetIdentityKey string
	AssetSymbol      string
	AssetName        string
	Quantity         string
	OrderCurrency    string
	OrderUnitPrice   string
	OrderGrossValue  string
	OrderFeeAmount   string
}

// roundedReportActivity converts one compact rounded-division integration
// fixture into the normalized activity record used by the runtime flow.
// Authored by: OpenCode
func roundedReportActivity(t *testing.T, input roundedReportActivityInput) syncmodel.ActivityRecord {
	t.Helper()

	return syncmodel.ActivityRecord{
		SourceID:         input.SourceID,
		OccurredAt:       input.OccurredAt,
		ActivityType:     input.ActivityType,
		AssetIdentityKey: input.AssetIdentityKey,
		AssetSymbol:      input.AssetSymbol,
		AssetName:        input.AssetName,
		Quantity:         mustRoundedIntegrationDecimal(t, input.Quantity),
		OrderCurrency:    input.OrderCurrency,
		OrderUnitPrice:   roundedIntegrationDecimalPointer(t, input.OrderUnitPrice),
		OrderGrossValue:  roundedIntegrationDecimalPointer(t, input.OrderGrossValue),
		OrderFeeAmount:   roundedIntegrationDecimalPointer(t, input.OrderFeeAmount),
	}
}

// mustReportGenerationSyncConfig returns one custom-origin config for sync then
// runtime-backed report generation within the same base directory.
// Authored by: OpenCode
func mustReportGenerationSyncConfig(t *testing.T, origin string) configmodel.AppSetupConfig {
	t.Helper()

	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, origin, true, time.Now())
	if err != nil {
		t.Fatalf("new report-generation sync config: %v", err)
	}

	return config
}

// mustReportFixtureTime parses one RFC3339 fixture timestamp for integration
// caches.
// Authored by: OpenCode
func mustReportFixtureTime(t *testing.T, raw string) time.Time {
	t.Helper()

	var parsed, err = time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse report fixture time %q: %v", raw, err)
	}

	return parsed
}

// roundedIntegrationDecimalPointer parses one optional decimal fixture for
// rounded-division integration tests.
// Authored by: OpenCode
func roundedIntegrationDecimalPointer(t *testing.T, raw string) *apd.Decimal {
	t.Helper()

	if raw == "" {
		return nil
	}

	var value = mustRoundedIntegrationDecimal(t, raw)
	return &value
}

// mustRoundedIntegrationDecimal parses one decimal fixture for rounded-
// division integration tests.
// Authored by: OpenCode
func mustRoundedIntegrationDecimal(t *testing.T, raw string) apd.Decimal {
	t.Helper()

	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		t.Fatalf("parse rounded integration decimal %q: %v", raw, err)
	}

	return value
}
