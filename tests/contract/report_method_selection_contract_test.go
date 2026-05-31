// Package contract verifies rendered workflow and Ghostfolio-boundary contracts
// for the sync-and-storage slice.
// Authored by: OpenCode
package contract

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/component"
	"github.com/benizzio/ghostfolio-cryptogains/internal/tui/screen"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var wrappedHyphenPattern = regexp.MustCompile(`-\s+`)

// TestReportMethodSelectionContract verifies exact method labels, stable
// filename slugs, and highlighted explanation text for the report-selection
// workflow.
// Authored by: OpenCode
func TestReportMethodSelectionContract(t *testing.T) {
	t.Parallel()

	var methods = reportmodel.SupportedCostBasisMethods()
	var expected = []struct {
		method      reportmodel.CostBasisMethod
		label       string
		slug        string
		explanation string
	}{
		{method: reportmodel.CostBasisMethodFIFO, label: "FIFO", slug: "fifo", explanation: "FIFO uses the oldest open acquisitions first."},
		{method: reportmodel.CostBasisMethodLIFO, label: "LIFO", slug: "lifo", explanation: "LIFO uses the newest open acquisitions first."},
		{method: reportmodel.CostBasisMethodHIFO, label: "HIFO", slug: "hifo", explanation: "HIFO uses the highest-unit-cost open acquisitions first."},
		{method: reportmodel.CostBasisMethodAverageCost, label: "Average Cost Basis", slug: "average-cost", explanation: "Average Cost Basis uses one moving weighted-average pool."},
		{method: reportmodel.CostBasisMethodScopeLocalHybrid, label: "Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order", slug: "scope-local-hybrid", explanation: "Scope-local exact matching stays narrow when defensible and otherwise falls back to scope-local average cost until the scope reaches zero."},
	}

	if len(methods) != len(expected) {
		t.Fatalf("unexpected supported method count: got %d want %d", len(methods), len(expected))
	}

	var methodItems = make([]component.MenuItem, 0, len(methods))
	for index, item := range expected {
		if methods[index] != item.method {
			t.Fatalf("unexpected supported method order at index %d: got %q want %q", index, methods[index], item.method)
		}
		if item.method.Label() != item.label {
			t.Fatalf("unexpected method label for %q: got %q want %q", item.method, item.method.Label(), item.label)
		}
		if item.method.FilenameSlug() != item.slug {
			t.Fatalf("unexpected method slug for %q: got %q want %q", item.method, item.method.FilenameSlug(), item.slug)
		}
		if item.method.Explanation() != item.explanation {
			t.Fatalf("unexpected method explanation for %q: got %q want %q", item.method, item.method.Explanation(), item.explanation)
		}
		methodItems = append(methodItems, component.MenuItem{Label: item.label, Enabled: true})
	}

	for index, item := range expected {
		var selection = screen.ReportSelectionScreenView(screen.ReportSelectionScreenParams{
			Theme:             component.DefaultTheme(),
			Width:             100,
			Height:            32,
			AvailableYears:    []int{2024, 2025},
			SelectedYearIndex: 0,
			MethodItems:       methodItems,
			SelectedMethod:    index,
			MethodExplanation: methods[index].Explanation(),
			MenuItems:         []component.MenuItem{{Label: "Generate Report", Enabled: true}, {Label: "Back", Enabled: true}},
			SelectedAction:    0,
		})
		var normalizedSelection = normalizeRenderedContractText(selection)
		var normalizedExplanation = normalizeRenderedContractText(item.explanation)

		assertContains(t, normalizedSelection, "Method Explanation")
		assertContains(t, normalizedSelection, normalizedExplanation)

		for otherIndex, other := range expected {
			if otherIndex == index {
				continue
			}
			assertNotContains(t, normalizedSelection, fmt.Sprintf("Method Explanation %s", other.explanation))
		}
	}
}

// normalizeRenderedContractText removes terminal styling and layout-only
// whitespace so contract assertions can match the visible text content.
// Authored by: OpenCode
func normalizeRenderedContractText(content string) string {
	var normalized = ansiEscapePattern.ReplaceAllString(content, "")
	normalized = strings.NewReplacer(
		"│", " ",
		"╭", " ",
		"╮", " ",
		"╰", " ",
		"╯", " ",
		"─", " ",
	).Replace(normalized)
	normalized = wrappedHyphenPattern.ReplaceAllString(normalized, "-")
	return strings.Join(strings.Fields(normalized), " ")
}
