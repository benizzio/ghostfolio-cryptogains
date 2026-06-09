package main

import (
	"encoding/json"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestOracleNormalizeOracleOutputCanonicalizesAndHashesDeterministically verifies
// the normalizer emits canonical fixture values, stable ordering, and a stable
// stored hash.
// Authored by: OpenCode
func TestOracleNormalizeOracleOutputCanonicalizesAndHashesDeterministically(t *testing.T) {
	t.Parallel()

	var input = oracleOutputNormalizationInput{
		DatasetVersion:   "1",
		CaseID:           "case-fifo-basic-2024",
		Method:           reportmodel.CostBasisMethodFIFO,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: comparableOutputValuesInput{
			RealizedGainOrLoss: "005.0000",
			AllocatedBasis:     "010.0000",
			ClosingQuantity:    ".5000",
			ClosingBasis:       "6.0000",
		},
		Matches: []oracleMatchEvidenceInput{
			{
				DisposedSourceID:    "emp-act-000020",
				AcquisitionSourceID: "emp-act-000002",
				MatchedQuantity:     "0.2500",
				MatchedBasis:        "3.0000",
				MatchedProceeds:     "4.5000",
				MatchedGainOrLoss:   "1.5000",
			},
			{
				DisposedSourceID:    "emp-act-000010",
				AcquisitionSourceID: "emp-act-000001",
				MatchedQuantity:     "0.7500",
				MatchedBasis:        "7.0000",
				MatchedProceeds:     "10.5000",
				MatchedGainOrLoss:   "3.5000",
			},
		},
		Metadata: oracleGenerationMetadataInput{
			RunID:            "run-001",
			HledgerVersion:   "1.99.2",
			CommandArguments: []string{"-f", "testdata/empirical/hledger/fifo.journal", "print"},
			DatasetInputHash: stablePrefixedSHA256Hash([]byte("dataset")),
			HledgerInputHash: stablePrefixedSHA256Hash([]byte("hledger")),
			DecimalPolicy:    "scale=16,rounding=half_up",
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0.0000000000000001",
				"allocated_basis":       "0",
				"closing_basis":         "0.0000000000000000",
			},
			ToleranceNotes: map[string]string{
				"realized_gain_or_loss": "Synthetic residual note",
			},
			GeneratedAt: "2026-06-08T12:00:00Z",
		},
	}

	var output, err = normalizeOracleOutput(input)
	if err != nil {
		t.Fatalf("normalize oracle output: %v", err)
	}

	if output.FixtureVersion != "1" || output.Metadata.NormalizationVersion != "1" || output.Metadata.HledgerVersion != "1.99.2" {
		t.Fatalf("unexpected metadata versions: %+v", output.Metadata)
	}
	if output.Values.RealizedGainOrLoss != "5" || output.Values.AllocatedBasis != "10" || output.Values.ClosingQuantity != "0.5" || output.Values.ClosingBasis != "6" {
		t.Fatalf("unexpected canonical values: %+v", output.Values)
	}
	if output.Metadata.FinancialTolerances["closing_basis"] != "0" {
		t.Fatalf("expected tolerance canonicalization, got %q", output.Metadata.FinancialTolerances["closing_basis"])
	}
	if len(output.Matches) != 2 || output.Matches[0].DisposedSourceID != "emp-act-000010" || output.Matches[1].DisposedSourceID != "emp-act-000020" {
		t.Fatalf("expected stable match ordering, got %+v", output.Matches)
	}

	var recomputedHash, hashErr = stableOracleOutputHash(output)
	if hashErr != nil {
		t.Fatalf("recompute stable hash: %v", hashErr)
	}
	if output.Metadata.OracleOutputHash != recomputedHash {
		t.Fatalf("unexpected stored hash: got %q want %q", output.Metadata.OracleOutputHash, recomputedHash)
	}

	var repeatedOutput, repeatedErr = normalizeOracleOutput(input)
	if repeatedErr != nil {
		t.Fatalf("normalize repeated oracle output: %v", repeatedErr)
	}
	if repeatedOutput.Metadata.OracleOutputHash != output.Metadata.OracleOutputHash {
		t.Fatalf("expected stable hash across unchanged inputs: first %q second %q", output.Metadata.OracleOutputHash, repeatedOutput.Metadata.OracleOutputHash)
	}
}

// TestOracleStableOracleOutputHashExcludesSelfReferentialMetadata verifies the
// stable hash ignores the stored hash field and regeneration-only metadata.
// Authored by: OpenCode
func TestOracleStableOracleOutputHashExcludesSelfReferentialMetadata(t *testing.T) {
	t.Parallel()

	var output = fixture.OracleOutput{
		FixtureVersion:   "1",
		DatasetVersion:   "1",
		CaseID:           "case-fifo-basic-2024",
		Method:           reportmodel.CostBasisMethodFIFO,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: fixture.ComparableOutputValues{
			RealizedGainOrLoss: "5",
			AllocatedBasis:     "10",
			ClosingQuantity:    "0.5",
			ClosingBasis:       "6",
		},
		Matches: []fixture.OracleMatchEvidence{{
			DisposedSourceID:    "emp-act-000010",
			AcquisitionSourceID: "emp-act-000001",
			MatchedQuantity:     "1",
			MatchedBasis:        "10",
		}},
		UnsupportedSegments: []fixture.UnsupportedOracleSegment{},
		Metadata: fixture.OracleGenerationRun{
			RunID:                "run-001",
			HledgerVersion:       "1.99.2",
			CommandArguments:     []string{"-f", "testdata/empirical/hledger/fifo.journal", "print"},
			DatasetInputHash:     stablePrefixedSHA256Hash([]byte("dataset")),
			HledgerInputHash:     stablePrefixedSHA256Hash([]byte("hledger")),
			DecimalPolicy:        "scale=16,rounding=half_up",
			NormalizationVersion: "1",
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0",
				"allocated_basis":       "0",
				"closing_basis":         "0",
			},
			ToleranceNotes:   map[string]string{},
			OracleOutputHash: stablePrefixedSHA256Hash([]byte("placeholder")),
			GeneratedAt:      "2026-06-08T12:00:00Z",
		},
	}

	var firstHash, err = stableOracleOutputHash(output)
	if err != nil {
		t.Fatalf("compute first stable hash: %v", err)
	}

	output.Metadata.OracleOutputHash = stablePrefixedSHA256Hash([]byte("changed"))
	output.Metadata.RunID = "run-999"
	output.Metadata.GeneratedAt = "2030-01-01T00:00:00Z"

	var secondHash, secondErr = stableOracleOutputHash(output)
	if secondErr != nil {
		t.Fatalf("compute second stable hash: %v", secondErr)
	}
	if firstHash != secondHash {
		t.Fatalf("expected self-referential metadata exclusion: first %q second %q", firstHash, secondHash)
	}
}

// TestOracleNormalizeOracleOutputRejectsInvalidNormalizedFixture verifies the
// generator-side normalizer fails when the resulting fixture violates the shared
// contract.
// Authored by: OpenCode
func TestOracleNormalizeOracleOutputRejectsInvalidNormalizedFixture(t *testing.T) {
	t.Parallel()

	var input = oracleOutputNormalizationInput{
		DatasetVersion:   "1",
		CaseID:           "case-scope-local-hybrid-2024",
		Method:           reportmodel.CostBasisMethodScopeLocalHybrid,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: comparableOutputValuesInput{
			RealizedGainOrLoss: "5",
			AllocatedBasis:     "10",
			ClosingQuantity:    "0.5",
			ClosingBasis:       "6",
		},
		Matches: []oracleMatchEvidenceInput{{
			DisposedSourceID:  "emp-act-000010",
			MatchedQuantity:   "1",
			MatchedBasis:      "10",
			MatchedGainOrLoss: "5",
		}},
		Metadata: oracleGenerationMetadataInput{
			HledgerVersion:   "1.99.2",
			CommandArguments: []string{"-f", "testdata/empirical/hledger/scope-local-hybrid.journal", "print"},
			DatasetInputHash: stablePrefixedSHA256Hash([]byte("dataset")),
			HledgerInputHash: stablePrefixedSHA256Hash([]byte("hledger")),
			DecimalPolicy:    "scale=16,rounding=half_up",
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0.0000000000000001",
				"allocated_basis":       "0",
				"closing_basis":         "0",
			},
			ToleranceNotes: map[string]string{
				"realized_gain_or_loss": "Synthetic residual note",
			},
		},
	}

	_, err := normalizeOracleOutput(input)
	if err == nil {
		t.Fatal("expected invalid scope-local-hybrid normalization failure, got nil")
	}
	if !strings.Contains(err.Error(), "support_label") {
		t.Fatalf("expected support_label validation error, got %v", err)
	}
}

// TestOracleStablePrefixedSHA256HashMatchesKnownDigest verifies the low-level
// hash helper uses deterministic sha256 output with the required prefix.
// Authored by: OpenCode
func TestOracleStablePrefixedSHA256HashMatchesKnownDigest(t *testing.T) {
	t.Parallel()

	var hash = stablePrefixedSHA256Hash([]byte("oracle-output"))
	if hash != "sha256:ef6cd929ba396754f003b2c94c0a54901ed96ad6ca0591cfc940d290d5411046" {
		t.Fatalf("unexpected stable sha256 hash: got %q", hash)
	}
}

// TestOracleNormalizeOracleOutputProducesStrictJSONShape verifies the
// normalized fixture keeps string-based decimals and the shared schema shape.
// Authored by: OpenCode
func TestOracleNormalizeOracleOutputProducesStrictJSONShape(t *testing.T) {
	t.Parallel()

	var input = oracleOutputNormalizationInput{
		DatasetVersion:   "1",
		CaseID:           "case-fifo-json-shape-2024",
		Method:           reportmodel.CostBasisMethodFIFO,
		Year:             2024,
		AssetIdentityKey: "asset-alpha",
		Values: comparableOutputValuesInput{
			RealizedGainOrLoss: "5",
			AllocatedBasis:     "10",
			ClosingQuantity:    "0.5",
			ClosingBasis:       "6",
		},
		Metadata: oracleGenerationMetadataInput{
			HledgerVersion:   "1.99.2",
			CommandArguments: []string{"-f", "testdata/empirical/hledger/fifo.journal", "print"},
			DatasetInputHash: stablePrefixedSHA256Hash([]byte("dataset")),
			HledgerInputHash: stablePrefixedSHA256Hash([]byte("hledger")),
			DecimalPolicy:    "scale=16,rounding=half_up",
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0",
				"allocated_basis":       "0",
				"closing_basis":         "0",
			},
			ToleranceNotes: map[string]string{},
		},
	}

	var output, err = normalizeOracleOutput(input)
	if err != nil {
		t.Fatalf("normalize oracle output: %v", err)
	}

	var rawContent, marshalErr = json.Marshal(output)
	if marshalErr != nil {
		t.Fatalf("marshal normalized output: %v", marshalErr)
	}
	if strings.Contains(string(rawContent), `"allocated_basis":10`) || strings.Contains(string(rawContent), `"realized_gain_or_loss":5`) {
		t.Fatalf("expected string-only decimal JSON fields, got %s", string(rawContent))
	}
}
