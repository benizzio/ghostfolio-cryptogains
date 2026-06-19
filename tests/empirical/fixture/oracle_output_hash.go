// Package fixture provides empirical fixture loading and validation helpers.
// Authored by: OpenCode
package fixture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

// StableOracleOutputHash returns the deterministic stable hash for one oracle
// output fixture.
//
// Example:
//
//	hash, err := fixture.StableOracleOutputHash(output)
//	if err != nil {
//		panic(err)
//	}
//	_ = hash
//
// StableOracleOutputHash hashes a canonical JSON representation and excludes
// `metadata.oracle_output_hash`, `metadata.run_id`, and `metadata.generated_at`
// so equivalent normalized fixtures keep the same stored hash across
// regenerations.
// Authored by: OpenCode
func StableOracleOutputHash(output OracleOutput) (string, error) {
	var canonicalOutput, err = canonicalOracleOutputForHash(output)
	if err != nil {
		return "", fmt.Errorf("canonicalize oracle output hash input: %w", err)
	}

	var payload []byte
	payload, err = json.Marshal(canonicalOutput)
	if err != nil {
		return "", fmt.Errorf("marshal oracle output hash input: %w", err)
	}

	var digest = sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

// canonicalOracleOutputForHash prepares one oracle-output fixture for stable
// hashing by canonicalizing decimal strings, sorting evidence slices, and
// clearing self-referential or ephemeral metadata fields.
// Authored by: OpenCode
func canonicalOracleOutputForHash(output OracleOutput) (OracleOutput, error) {
	var canonical = output
	var err error

	canonical.Values, err = canonicalComparableOutputValues(output.Values)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.Matches, err = canonicalOracleMatches(output.Matches)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.UnsupportedSegments, err = canonicalUnsupportedOracleSegments(output.UnsupportedSegments)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.Metadata, err = canonicalOracleGenerationRun(output.Metadata)
	if err != nil {
		return OracleOutput{}, err
	}

	canonical.Metadata.RunID = ""
	canonical.Metadata.GeneratedAt = ""
	canonical.Metadata.OracleOutputHash = ""

	return canonical, nil
}

// canonicalComparableOutputValues canonicalizes the comparable decimal-string values.
// Authored by: OpenCode
func canonicalComparableOutputValues(values ComparableOutputValues) (ComparableOutputValues, error) {
	var err error

	values.RealizedGainOrLoss, err = canonicalRequiredPersistedDecimal(values.RealizedGainOrLoss)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize realized_gain_or_loss: %w", err)
	}
	values.AllocatedBasis, err = canonicalRequiredPersistedDecimal(values.AllocatedBasis)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize allocated_basis: %w", err)
	}
	values.ClosingQuantity, err = canonicalRequiredPersistedDecimal(values.ClosingQuantity)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize closing_quantity: %w", err)
	}
	values.ClosingBasis, err = canonicalRequiredPersistedDecimal(values.ClosingBasis)
	if err != nil {
		return ComparableOutputValues{}, fmt.Errorf("canonicalize closing_basis: %w", err)
	}

	return values, nil
}

// canonicalOracleMatches canonicalizes and sorts comparable match evidence.
// Authored by: OpenCode
func canonicalOracleMatches(matches []OracleMatchEvidence) ([]OracleMatchEvidence, error) {
	var canonical = make([]OracleMatchEvidence, len(matches))
	copy(canonical, matches)

	var index int
	for index = range canonical {
		var err error
		canonical[index].MatchedQuantity, err = canonicalRequiredPersistedDecimal(canonical[index].MatchedQuantity)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_quantity: %w", index, err)
		}
		canonical[index].MatchedBasis, err = canonicalRequiredPersistedDecimal(canonical[index].MatchedBasis)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_basis: %w", index, err)
		}
		canonical[index].MatchedProceeds, err = canonicalOptionalPersistedDecimal(canonical[index].MatchedProceeds)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_proceeds: %w", index, err)
		}
		canonical[index].MatchedGainOrLoss, err = canonicalOptionalPersistedDecimal(canonical[index].MatchedGainOrLoss)
		if err != nil {
			return nil, fmt.Errorf("canonicalize match %d matched_gain_or_loss: %w", index, err)
		}
	}

	sort.Slice(canonical, func(left int, right int) bool {
		return oracleMatchSortKey(canonical[left]) < oracleMatchSortKey(canonical[right])
	})

	return canonical, nil
}

// canonicalUnsupportedOracleSegments canonicalizes and sorts unsupported segments.
// Authored by: OpenCode
func canonicalUnsupportedOracleSegments(segments []UnsupportedOracleSegment) ([]UnsupportedOracleSegment, error) {
	var canonical = make([]UnsupportedOracleSegment, len(segments))
	copy(canonical, segments)

	var index int
	for index = range canonical {
		canonical[index].ActivitySourceIDs = slices.Clone(canonical[index].ActivitySourceIDs)
		sort.Strings(canonical[index].ActivitySourceIDs)
	}

	sort.Slice(canonical, func(left int, right int) bool {
		return unsupportedOracleSegmentSortKey(canonical[left]) < unsupportedOracleSegmentSortKey(canonical[right])
	})

	return canonical, nil
}

// canonicalOracleGenerationRun canonicalizes the hash-relevant generation metadata.
// Authored by: OpenCode
func canonicalOracleGenerationRun(metadata OracleGenerationRun) (OracleGenerationRun, error) {
	metadata.AdapterArguments = slices.Clone(metadata.AdapterArguments)
	metadata.AdapterConstraints = slices.Clone(metadata.AdapterConstraints)
	metadata.FinancialTolerances = maps.Clone(metadata.FinancialTolerances)
	metadata.ToleranceNotes = maps.Clone(metadata.ToleranceNotes)

	for field, rawValue := range metadata.FinancialTolerances {
		var canonicalValue, err = canonicalRequiredPersistedDecimal(rawValue)
		if err != nil {
			return OracleGenerationRun{}, fmt.Errorf("canonicalize financial_tolerances.%s: %w", field, err)
		}

		metadata.FinancialTolerances[field] = canonicalValue
	}

	return metadata, nil
}

// canonicalRequiredPersistedDecimal canonicalizes one required persisted decimal string.
// Authored by: OpenCode
func canonicalRequiredPersistedDecimal(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("decimal value is required")
	}

	_, canonical, err := ParseDecimalString(raw)
	if err != nil {
		return "", err
	}

	return canonical, nil
}

// canonicalOptionalPersistedDecimal canonicalizes one optional persisted decimal string.
// Authored by: OpenCode
func canonicalOptionalPersistedDecimal(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}

	return canonicalRequiredPersistedDecimal(raw)
}

// oracleMatchSortKey returns the stable lexical sort key for one match-evidence row.
// Authored by: OpenCode
func oracleMatchSortKey(match OracleMatchEvidence) string {
	return strings.Join([]string{
		match.DisposedSourceID,
		match.AcquisitionSourceID,
		match.ScopeID,
		match.MatchedQuantity,
		match.MatchedBasis,
		match.MatchedProceeds,
		match.MatchedGainOrLoss,
		string(match.SupportLabel),
		match.CompositionRuleID,
	}, "\x00")
}

// unsupportedOracleSegmentSortKey returns the stable lexical sort key for one
// unsupported segment.
// Authored by: OpenCode
func unsupportedOracleSegmentSortKey(segment UnsupportedOracleSegment) string {
	return strings.Join([]string{
		segment.CaseID,
		string(segment.Method),
		strings.Join(segment.ActivitySourceIDs, "\x01"),
		segment.Reason,
		string(segment.ComparisonPolicy),
	}, "\x00")
}
