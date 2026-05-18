package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/benizzio/ghostfolio-cryptogains/internal/app/bootstrap"
	"github.com/benizzio/ghostfolio-cryptogains/internal/app/runtime"
	configmodel "github.com/benizzio/ghostfolio-cryptogains/internal/config/model"
	ghostfolioclient "github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/client"
	snapshotstore "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/store"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncnormalize "github.com/benizzio/ghostfolio-cryptogains/internal/sync/normalize"
	syncvalidate "github.com/benizzio/ghostfolio-cryptogains/internal/sync/validate"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

func TestSyncDiagnosticReportFlowPromptsInProductionAndWritesOnExplicitChoice(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newGhostfolioStorageTLSServer(t, []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"unsupported-1","date":"2024-01-02T10:00:00Z","type":"TRANSFER","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}]`,
	}})
	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, false, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	service := runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		baseDir,
		false,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
	model := flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, false, service))

	model = openSyncEntry(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInputFromSyncEntry(t, model)
	model, cmd := startSyncAttempt(t, model)
	model = applySyncBatch(t, model, cmd)

	content := model.View().Content
	if !strings.Contains(content, "Generate Diagnostic Report") {
		t.Fatalf("expected production diagnostic-report prompt, got %q", content)
	}
	if strings.Contains(content, ".diagnostic.json") {
		t.Fatalf("expected no written path before explicit choice, got %q", content)
	}

	updated, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = assertFlowModel(t, updated)
	if !strings.Contains(model.View().Content, "Generating diagnostic report...") {
		t.Fatalf("expected visible diagnostic-report busy status, got %q", model.View().Content)
	}
	updated, _ = model.Update(testutil.RunCmd(cmd))
	model = assertFlowModel(t, updated)
	content = model.View().Content
	if !strings.Contains(content, ".diagnostic.json") {
		t.Fatalf("expected generated-report path disclosure after explicit choice, got %q", content)
	}

	diagnosticFiles := mustDiagnosticFiles(t, baseDir)
	if len(diagnosticFiles) != 1 {
		t.Fatalf("expected one generated diagnostic report, got %d", len(diagnosticFiles))
	}
	reportBytes, err := os.ReadFile(diagnosticFiles[0])
	if err != nil {
		t.Fatalf("read diagnostic report: %v", err)
	}
	reportText := string(reportBytes)
	if strings.Contains(reportText, "abc123") || strings.Contains(reportText, "jwt") {
		t.Fatalf("expected report to stay secret-safe, got %q", reportText)
	}
	records := mustDiagnosticReportRecords(t, reportBytes)
	if len(records) == 0 {
		t.Fatalf("expected production diagnostic report records")
	}
	if records[0].Quantity == "1" || records[0].UnitPrice == "100" || records[0].GrossValue == "100" {
		t.Fatalf("expected production report to redact financial values, got %q", reportText)
	}
}

func TestSyncDiagnosticReportFlowGeneratesAutomaticallyInExplicitDevelopmentMode(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	server := newGhostfolioStorageServer(t, []storagePageFixture{{
		Count:          1,
		ActivitiesJSON: `[{"id":"unsupported-1","date":"2024-01-02T10:00:00Z","type":"TRANSFER","quantity":1,"valueInBaseCurrency":100,"unitPriceInAssetProfileCurrency":100,"SymbolProfile":{"symbol":"BTC","name":"Bitcoin","currency":"USD"}}]`,
	}})
	config, err := configmodel.NewSetupConfig(configmodel.ServerModeCustomOrigin, server.URL, true, time.Now())
	if err != nil {
		t.Fatalf("new setup config: %v", err)
	}
	service := runtime.NewSyncService(
		ghostfolioclient.New(server.Client()),
		time.Second,
		baseDir,
		true,
		decimalsupport.NewService(),
		syncnormalize.NewNormalizer(),
		syncvalidate.NewValidator(),
		snapshotstore.NewEncryptedStore(baseDir, nil),
	)
	model := flow.NewModel(newFlowDependencies(t, bootstrap.StartupState{ActiveConfig: &config}, true, service))

	model = openSyncEntry(t, model)
	model = typeToken(t, model, "abc123")
	model = blurTokenInputFromSyncEntry(t, model)
	model, cmd := startSyncAttempt(t, model)
	model = applySyncBatch(t, model, cmd)

	content := model.View().Content
	if strings.Contains(content, "Generate Diagnostic Report") {
		t.Fatalf("expected explicit development mode to skip the prompt, got %q", content)
	}
	if !strings.Contains(content, ".diagnostic.json") {
		t.Fatalf("expected explicit development mode to disclose written report path, got %q", content)
	}

	diagnosticFiles := mustDiagnosticFiles(t, baseDir)
	if len(diagnosticFiles) != 1 {
		t.Fatalf("expected one generated diagnostic report, got %d", len(diagnosticFiles))
	}
	reportBytes, err := os.ReadFile(diagnosticFiles[0])
	if err != nil {
		t.Fatalf("read diagnostic report: %v", err)
	}
	reportText := string(reportBytes)
	if strings.Contains(reportText, "abc123") || strings.Contains(reportText, "jwt") {
		t.Fatalf("expected development report to stay secret-safe, got %q", reportText)
	}
	records := mustDiagnosticReportRecords(t, reportBytes)
	if len(records) == 0 {
		t.Fatalf("expected development diagnostic report records")
	}
	if records[0].Quantity != "1" || records[0].UnitPrice != "100" || records[0].GrossValue != "100" {
		t.Fatalf("expected development report to retain allowed financial context, got %q", reportText)
	}
}

func TestSyncDiagnosticReportFlowTLSServerHandlesEmptyPagesFixture(t *testing.T) {
	t.Parallel()

	var server = newGhostfolioStorageTLSServer(t, nil)
	var response, err = server.Client().Get(server.URL + "/api/v1/activities?sortColumn=date&sortDirection=asc")
	if err != nil {
		t.Fatalf("get empty activity page: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected empty fixture request to succeed, got %d", response.StatusCode)
	}

	var responseBody []byte
	responseBody, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read empty activity page: %v", err)
	}
	if string(responseBody) != `{"activities":[],"count":0}` {
		t.Fatalf("expected explicit empty activity response, got %q", string(responseBody))
	}
}

func newGhostfolioStorageTLSServer(t *testing.T, pages []storagePageFixture) *httptest.Server {
	t.Helper()

	var requestCount int
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/anonymous":
			_, _ = writer.Write([]byte(`{"authToken":"jwt"}`))
		case "/api/v1/user":
			_, _ = writer.Write([]byte(`{"settings":{"baseCurrency":"USD"}}`))
		case "/api/v1/activities":
			if request.URL.Query().Get("sortColumn") != "date" || request.URL.Query().Get("sortDirection") != "asc" {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if len(pages) == 0 {
				_, _ = writer.Write([]byte(`{"activities":[],"count":0}`))
				return
			}
			if requestCount >= len(pages) {
				requestCount = len(pages) - 1
			}
			page := pages[requestCount]
			requestCount++
			_, _ = writer.Write([]byte(fmt.Sprintf(`{"activities":%s,"count":%d}`, page.ActivitiesJSON, page.Count)))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func mustDiagnosticFiles(t *testing.T, baseDir string) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(baseDir, "ghostfolio-cryptogains", "diagnostics"))
	if err != nil {
		t.Fatalf("read diagnostics directory: %v", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, filepath.Join(baseDir, "ghostfolio-cryptogains", "diagnostics", entry.Name()))
	}
	return paths
}

type diagnosticReportRecord struct {
	Quantity              string `json:"quantity"`
	UnitPrice             string `json:"unit_price"`
	GrossValue            string `json:"gross_value"`
	OrderUnitPrice        string `json:"order_unit_price"`
	OrderGrossValue       string `json:"order_gross_value"`
	AssetProfileUnitPrice string `json:"asset_profile_unit_price"`
	BaseGrossValue        string `json:"base_gross_value"`
}

type diagnosticReportPayload struct {
	Records []diagnosticReportRecord `json:"records"`
}

func mustDiagnosticReportRecords(t *testing.T, reportBytes []byte) []diagnosticReportRecord {
	t.Helper()

	var payload diagnosticReportPayload
	if err := json.Unmarshal(reportBytes, &payload); err == nil && len(payload.Records) > 0 {
		return payload.Records
	}

	var nested struct {
		Context diagnosticReportPayload `json:"context"`
	}
	if err := json.Unmarshal(reportBytes, &nested); err != nil {
		t.Fatalf("unmarshal diagnostic report: %v", err)
	}
	return nested.Context.Records
}
