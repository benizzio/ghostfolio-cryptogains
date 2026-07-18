package integration

import "github.com/benizzio/ghostfolio-cryptogains/tests/testutil/runtimeflow"

// Runtimeflow bindings keep integration scenarios focused on their assertions
// while using the authoritative shared runtime-backed test boundary.
// Authored by: OpenCode
var (
	newRuntimeBackedFlowHarness            = runtimeflow.NewRuntimeBackedFlowHarness
	seedProtectedSnapshot                  = runtimeflow.SeedProtectedSnapshot
	unlockSyncReportsContext               = runtimeflow.UnlockSyncReportsContext
	openReportSelectionFromContext         = runtimeflow.OpenReportSelection
	selectReportYear                       = runtimeflow.SelectReportYear
	selectReportBaseCurrency               = runtimeflow.SelectReportBaseCurrency
	startReportGenerationFromSelection     = runtimeflow.StartReportGeneration
	applyBatchCmd                          = runtimeflow.ApplyBatchCmd
	assertFlowModel                        = runtimeflow.AssertFlowModel
	normalizeRenderedText                  = runtimeflow.NormalizeRenderedText
	mustMarkdownFiles                      = runtimeflow.MarkdownFiles
	installOpenCommandRecorder             = runtimeflow.InstallOpenCommandRecorder
	readOpenCommandRequests                = runtimeflow.ReadOpenCommandRequests
	assertNoCleartextReportInAppStorage    = runtimeflow.AssertNoCleartextReportInAppStorage
	mustCloudSetupConfig                   = runtimeflow.MustCloudSetupConfig
	roundedUnitPriceProtectedActivityCache = runtimeflow.RoundedUnitPriceProtectedActivityCache
	mustIntegrationReportRequestForFormat  = runtimeflow.MustIntegrationReportRequestForFormat
	mustPDFFiles                           = runtimeflow.PDFFiles
)
