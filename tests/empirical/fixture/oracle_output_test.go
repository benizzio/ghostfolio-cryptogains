package fixture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// TestOracleLoadOracleOutputsLoadsValidatedFixtures verifies the fixture loader
// accepts valid oracle JSON files, validates them, and returns them in stable
// path order.
// Authored by: OpenCode
func TestOracleLoadOracleOutputsLoadsValidatedFixtures(t *testing.T) {
	t.Parallel()

	var rootPath = t.TempDir()
	var scopeLocalHybrid = newValidOracleOutputFixture(t, reportmodel.CostBasisMethodScopeLocalHybrid, "case-scope-local-hybrid-2024", 2024, "asset-alpha", true)
	var fifo = newValidOracleOutputFixture(t, reportmodel.CostBasisMethodFIFO, "case-fifo-2023", 2023, "asset-beta", false)

	writeOracleOutputFixtureFile(t, filepath.Join(rootPath, "a", "scope-local-hybrid.json"), scopeLocalHybrid)
	writeOracleOutputFixtureFile(t, filepath.Join(rootPath, "b", "fifo.json"), fifo)

	var outputs, err = LoadOracleOutputs(rootPath)
	if err != nil {
		t.Fatalf("load oracle outputs: %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("unexpected output count: got %d want %d", len(outputs), 2)
	}

	if outputs[0].CaseID != scopeLocalHybrid.CaseID || outputs[0].Method != reportmodel.CostBasisMethodScopeLocalHybrid || outputs[0].Year != 2024 {
		t.Fatalf("unexpected first fixture: %+v", outputs[0])
	}
	if outputs[0].Metadata.HledgerVersion != "1.99.2" {
		t.Fatalf("unexpected hledger version: got %q want %q", outputs[0].Metadata.HledgerVersion, "1.99.2")
	}
	if outputs[0].Values.RealizedGainOrLoss != "5" || outputs[0].Values.AllocatedBasis != "10" || outputs[0].Values.ClosingQuantity != "0.5" || outputs[0].Values.ClosingBasis != "6" {
		t.Fatalf("unexpected canonical comparable values: %+v", outputs[0].Values)
	}
	if len(outputs[0].Matches) != 2 {
		t.Fatalf("unexpected match count: got %d want %d", len(outputs[0].Matches), 2)
	}
	if outputs[0].Matches[0].SupportLabel != EvidenceSupportLabelHledgerBacked || outputs[0].Matches[1].SupportLabel != EvidenceSupportLabelProjectCompositionRule {
		t.Fatalf("unexpected support labels: %+v", outputs[0].Matches)
	}
	if len(outputs[0].UnsupportedSegments) != 1 || outputs[0].UnsupportedSegments[0].ComparisonPolicy != ComparisonPolicySkipExternalOracle {
		t.Fatalf("unexpected unsupported segments: %+v", outputs[0].UnsupportedSegments)
	}

	if outputs[1].CaseID != fifo.CaseID || outputs[1].Method != reportmodel.CostBasisMethodFIFO || outputs[1].Year != 2023 || outputs[1].AssetIdentityKey != "asset-beta" {
		t.Fatalf("unexpected second fixture: %+v", outputs[1])
	}
}

// TestOracleParseOracleOutputRejectsFloatJSONNumberForFinancialField verifies
// persisted financial fields must remain string-based decimals.
// Authored by: OpenCode
func TestOracleParseOracleOutputRejectsFloatJSONNumberForFinancialField(t *testing.T) {
	t.Parallel()

	var output = newValidOracleOutputFixture(t, reportmodel.CostBasisMethodFIFO, "case-float-json-number-2024", 2024, "asset-alpha", false)
	var rawContent = marshalOracleOutputFixture(t, output)
	rawContent = strings.Replace(rawContent, `"allocated_basis": "10"`, `"allocated_basis": 10`, 1)

	_, err := ParseOracleOutput("testdata/empirical/golden/float-number.json", []byte(rawContent))
	if err == nil {
		t.Fatal("expected float-style JSON number rejection, got nil")
	}
	assertOracleOutputErrorContainsAll(t, err, "allocated_basis", "cannot unmarshal number")
}

// TestOracleParseOracleOutputRejectsMissingRequiredMetadata verifies missing
// metadata fields are rejected during fixture parsing.
// Authored by: OpenCode
func TestOracleParseOracleOutputRejectsMissingRequiredMetadata(t *testing.T) {
	t.Parallel()

	var output = newValidOracleOutputFixture(t, reportmodel.CostBasisMethodFIFO, "case-missing-metadata-2024", 2024, "asset-alpha", false)
	var payload = marshalOracleOutputFixtureToMap(t, output)
	var metadata = requireOracleOutputNestedMap(t, payload, "metadata")
	delete(metadata, "hledger_version")

	var rawContent = marshalOracleOutputMap(t, payload)
	_, err := ParseOracleOutput("testdata/empirical/golden/missing-metadata.json", []byte(rawContent))
	if err == nil {
		t.Fatal("expected missing metadata rejection, got nil")
	}
	assertOracleOutputErrorContainsAll(t, err, "metadata.hledger_version", "required_field")
}

// TestOracleValidateOracleOutputRejectsNonCanonicalDecimalAndToleranceIssues
// verifies canonical decimal strings and tolerance contracts are enforced.
// Authored by: OpenCode
func TestOracleValidateOracleOutputRejectsNonCanonicalDecimalAndToleranceIssues(t *testing.T) {
	t.Parallel()

	var output = newValidOracleOutputFixture(t, reportmodel.CostBasisMethodFIFO, "case-invalid-decimal-2024", 2024, "asset-alpha", false)
	output.Values.AllocatedBasis = "010.0"
	output.Metadata.FinancialTolerances["realized_gain_or_loss"] = "0.0000000000000002"
	output.Metadata.ToleranceNotes["realized_gain_or_loss"] = ""

	var rawContent = marshalOracleOutputFixture(t, output)
	var err = ValidateOracleOutput("testdata/empirical/golden/invalid-decimal.json", rawContent, output)
	if err == nil {
		t.Fatal("expected invalid oracle output rejection, got nil")
	}
	assertOracleOutputErrorContainsAll(t, err,
		"values.allocated_basis",
		"canonical fixed-point representation",
		"metadata.financial_tolerances.realized_gain_or_loss",
		"exceeds the maximum",
		"metadata.tolerance_notes.realized_gain_or_loss",
		"non-zero financial tolerance requires a tolerance note",
	)
}

// TestOracleValidateOracleOutputRejectsStoredHashMismatch verifies the stored
// oracle_output_hash must match the recomputed stable hash.
// Authored by: OpenCode
func TestOracleValidateOracleOutputRejectsStoredHashMismatch(t *testing.T) {
	t.Parallel()

	var output = newValidOracleOutputFixture(t, reportmodel.CostBasisMethodFIFO, "case-hash-mismatch-2024", 2024, "asset-alpha", false)
	output.Metadata.OracleOutputHash = stableOracleFixtureHashText("different-hash")

	var rawContent = marshalOracleOutputFixture(t, output)
	var err = ValidateOracleOutput("testdata/empirical/golden/hash-mismatch.json", rawContent, output)
	if err == nil {
		t.Fatal("expected stored hash mismatch rejection, got nil")
	}
	assertOracleOutputErrorContainsAll(t, err, "metadata.oracle_output_hash", "stored hash", "recomputed hash")
}

// newValidOracleOutputFixture builds one fully valid synthetic oracle fixture.
// Authored by: OpenCode
func newValidOracleOutputFixture(t *testing.T, method reportmodel.CostBasisMethod, caseID string, year int, assetIdentityKey string, includeUnsupported bool) OracleOutput {
	t.Helper()

	var output = OracleOutput{
		FixtureVersion:   "1",
		DatasetVersion:   "1",
		CaseID:           caseID,
		Method:           method,
		Year:             year,
		AssetIdentityKey: assetIdentityKey,
		Values: ComparableOutputValues{
			RealizedGainOrLoss: "5",
			AllocatedBasis:     "10",
			ClosingQuantity:    "0.5",
			ClosingBasis:       "6",
		},
		Matches: []OracleMatchEvidence{
			{
				DisposedSourceID:    "emp-act-000010",
				AcquisitionSourceID: "emp-act-000001",
				MatchedQuantity:     "1",
				MatchedBasis:        "10",
				MatchedProceeds:     "15",
				MatchedGainOrLoss:   "5",
				SupportLabel:        defaultOracleOutputSupportLabel(method, false),
			},
		},
		UnsupportedSegments: []UnsupportedOracleSegment{},
		Metadata: OracleGenerationRun{
			HledgerVersion:       "1.99.2",
			CommandArguments:     []string{"-f", "testdata/empirical/hledger/" + method.FilenameSlug() + ".journal", "print"},
			DatasetInputHash:     stableOracleFixtureHashText(caseID + ":dataset"),
			HledgerInputHash:     stableOracleFixtureHashText(caseID + ":hledger"),
			DecimalPolicy:        "scale=16,rounding=half_up",
			NormalizationVersion: "1",
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0.0000000000000001",
				"allocated_basis":       "0",
				"closing_basis":         "0",
			},
			ToleranceNotes: map[string]string{
				"realized_gain_or_loss": "One-unit residual after decimal-policy alignment for synthetic fixture",
			},
		},
	}

	if method == reportmodel.CostBasisMethodScopeLocalHybrid {
		output.Matches = append(output.Matches, OracleMatchEvidence{
			DisposedSourceID:  "emp-act-000020",
			ScopeID:           "scope-synthetic-001",
			MatchedQuantity:   "0.5",
			MatchedBasis:      "6",
			MatchedGainOrLoss: "0",
			SupportLabel:      EvidenceSupportLabelProjectCompositionRule,
			CompositionRuleID: "scope_local_hybrid_fallback_pool_v1",
		})
	}

	if includeUnsupported {
		output.UnsupportedSegments = []UnsupportedOracleSegment{
			{
				CaseID:            caseID,
				Method:            method,
				ActivitySourceIDs: []string{"emp-act-000090", "emp-act-000099"},
				Reason:            "Synthetic zero-priced reduction stays outside faithful hledger representation for this case",
				ComparisonPolicy:  ComparisonPolicySkipExternalOracle,
			},
		}
	}

	var hash, err = StableOracleOutputHash(output)
	if err != nil {
		t.Fatalf("compute oracle fixture hash: %v", err)
	}
	output.Metadata.OracleOutputHash = hash

	return output
}

// defaultOracleOutputSupportLabel returns the support label required for one
// test fixture match row.
// Authored by: OpenCode
func defaultOracleOutputSupportLabel(method reportmodel.CostBasisMethod, compositionRule bool) EvidenceSupportLabel {
	if compositionRule {
		return EvidenceSupportLabelProjectCompositionRule
	}
	if method == reportmodel.CostBasisMethodScopeLocalHybrid {
		return EvidenceSupportLabelHledgerBacked
	}

	return ""
}

// writeOracleOutputFixtureFile persists one synthetic oracle fixture to a temp path.
// Authored by: OpenCode
func writeOracleOutputFixtureFile(t *testing.T, path string, output OracleOutput) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(marshalOracleOutputFixture(t, output)), 0o644); err != nil {
		t.Fatalf("write oracle fixture %s: %v", path, err)
	}
}

// marshalOracleOutputFixture serializes one oracle fixture into deterministic JSON text.
// Authored by: OpenCode
func marshalOracleOutputFixture(t *testing.T, output OracleOutput) string {
	t.Helper()

	var rawContent, err = json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("marshal oracle fixture: %v", err)
	}

	return string(rawContent)
}

// marshalOracleOutputFixtureToMap converts one oracle fixture into a generic map.
// Authored by: OpenCode
func marshalOracleOutputFixtureToMap(t *testing.T, output OracleOutput) map[string]any {
	t.Helper()

	var payload map[string]any
	var rawContent = marshalOracleOutputFixture(t, output)
	if err := json.Unmarshal([]byte(rawContent), &payload); err != nil {
		t.Fatalf("unmarshal oracle fixture into map: %v", err)
	}

	return payload
}

// requireOracleOutputNestedMap returns one nested JSON object from the generic payload.
// Authored by: OpenCode
func requireOracleOutputNestedMap(t *testing.T, payload map[string]any, field string) map[string]any {
	t.Helper()

	var rawValue, ok = payload[field]
	if !ok {
		t.Fatalf("missing nested map %q", field)
	}

	var nested, nestedOK = rawValue.(map[string]any)
	if !nestedOK {
		t.Fatalf("nested field %q is not a map: %T", field, rawValue)
	}

	return nested
}

// marshalOracleOutputMap serializes one generic oracle payload map to JSON text.
// Authored by: OpenCode
func marshalOracleOutputMap(t *testing.T, payload map[string]any) string {
	t.Helper()

	var rawContent, err = json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal oracle payload map: %v", err)
	}

	return string(rawContent)
}

// assertOracleOutputErrorContainsAll verifies every expected fragment exists in one error message.
// Authored by: OpenCode
func assertOracleOutputErrorContainsAll(t *testing.T, err error, fragments ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var message = err.Error()
	var index int
	for index = range fragments {
		if strings.Contains(message, fragments[index]) {
			continue
		}

		t.Fatalf("expected error %q to contain %q", message, fragments[index])
	}
}

// stableOracleFixtureHashText returns one deterministic `sha256:`-prefixed hash for test metadata.
// Authored by: OpenCode
func stableOracleFixtureHashText(raw string) string {
	var digest = sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(digest[:])
}
