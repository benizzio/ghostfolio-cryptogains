package runtimeflow

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/flow"
	"github.com/benizzio/ghostfolio-cryptogains/tests/testutil"
)

// UnlockSyncReportsContext unlocks the supplied model into the Sync and Reports menu.
// Authored by: OpenCode
func UnlockSyncReportsContext(t *testing.T, model *flow.Model, token string) *flow.Model {
	t.Helper()
	var updated tea.Model
	var cmd tea.Cmd
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = AssertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_unlock" {
		t.Fatalf("expected sync reports unlock screen, got %s", model.ActiveScreen())
	}
	// The visible token input accepts text through its dedicated input model.
	for _, character := range token {
		updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Text: string(character), Code: character}))
		_ = testutil.RunCmd(cmd)
		model = AssertFlowModel(t, updated)
	}
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	_ = testutil.RunCmd(cmd)
	model = AssertFlowModel(t, updated)
	updated, cmd = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	_ = testutil.RunCmd(cmd)
	model = AssertFlowModel(t, updated)
	if model.ActiveScreen() != "sync_reports_menu" {
		t.Fatalf("expected sync reports menu after unlock, got %s", model.ActiveScreen())
	}
	return model
}

// SelectReportBaseCurrency moves focus to the report base-currency list and
// selects the requested currency. For example, call it after SelectReportYear
// and before StartReportGeneration.
// Authored by: OpenCode
func SelectReportBaseCurrency(t *testing.T, model *flow.Model, reportBaseCurrency reportmodel.ReportBaseCurrency) *flow.Model {
	t.Helper()

	for focusStep := 0; focusStep < 2; focusStep++ {
		var updated tea.Model
		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		model = AssertFlowModel(t, updated)
	}

	var marker = "> " + reportBaseCurrency.Label()
	for attempt := 0; attempt < len(reportmodel.SupportedReportBaseCurrencies())+1; attempt++ {
		var content = NormalizeRenderedText(model.View().Content)
		if strings.Contains(content, "Report Base Currency") && strings.Contains(content, marker) {
			return model
		}

		var updated tea.Model
		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = AssertFlowModel(t, updated)
	}

	t.Fatalf("expected report base currency %q to be selected, got %q", reportBaseCurrency.Label(), model.View().Content)
	return model
}

// SelectReportOutputFormat moves focus to the output-format list and selects the
// requested supported format. For example, call it after SelectReportBaseCurrency
// and before StartReportGeneration when a journey must exercise PDF instead of
// the default Markdown output.
// Authored by: OpenCode
func SelectReportOutputFormat(t *testing.T, model *flow.Model, outputFormat reportmodel.ReportOutputFormat) *flow.Model {
	t.Helper()

	var supportedFormats = reportmodel.SupportedReportOutputFormats()
	var targetFound bool
	for _, supportedFormat := range supportedFormats {
		if supportedFormat == outputFormat {
			targetFound = true
			break
		}
	}
	if !targetFound {
		t.Fatalf("unsupported report output format %q", outputFormat)
	}

	var updated tea.Model
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = AssertFlowModel(t, updated)
	var marker = "> " + outputFormat.Label()
	for attempt := 0; attempt < len(supportedFormats); attempt++ {
		var content = NormalizeRenderedText(model.View().Content)
		if strings.Contains(content, "Output Format") && strings.Contains(content, marker) {
			return model
		}

		updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = AssertFlowModel(t, updated)
	}

	t.Fatalf("expected report output format %q to be selected, got %q", outputFormat, NormalizeRenderedText(model.View().Content))
	return model
}

// OpenReportSelection opens report selection from an unlocked context.
// Authored by: OpenCode
func OpenReportSelection(t *testing.T, model *flow.Model) *flow.Model {
	t.Helper()
	var updated tea.Model
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = AssertFlowModel(t, updated)
	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	model = AssertFlowModel(t, updated)
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
		if strings.Contains(NormalizeRenderedText(model.View().Content), marker) {
			return model
		}
		var updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
		model = AssertFlowModel(t, updated)
	}
	t.Fatalf("expected report year %d to be selected, got %q", year, NormalizeRenderedText(model.View().Content))
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
		model = AssertFlowModel(t, updated)
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
			model = AssertFlowModel(t, updated)
		}
	}
	return model
}

// AssertFlowModel returns updated as a flow model or fails the test. For example,
// call AssertFlowModel(t, updated) after Model.Update to continue a test flow.
// Authored by: OpenCode
func AssertFlowModel(t *testing.T, updated tea.Model) *flow.Model {
	t.Helper()
	var model, ok = updated.(*flow.Model)
	if !ok {
		t.Fatalf("expected updated model to be *flow.Model, got %T", updated)
	}
	return model
}
