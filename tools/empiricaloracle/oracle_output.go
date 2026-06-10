package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

const (
	oracleOutputFixtureVersion       = "1"
	oracleOutputNormalizationVersion = "1"
)

// oracleOutputNormalizationInput stores the raw external-oracle-derived values
// needed to build one normalized oracle fixture.
// Authored by: OpenCode
type oracleOutputNormalizationInput struct {
	FixtureVersion      string
	DatasetVersion      string
	CaseID              string
	Method              reportmodel.CostBasisMethod
	Year                int
	AssetIdentityKey    string
	Values              comparableOutputValuesInput
	Matches             []oracleMatchEvidenceInput
	UnsupportedSegments []unsupportedOracleSegmentInput
	Metadata            oracleGenerationMetadataInput
}

// comparableOutputValuesInput stores one raw comparable output block before the
// oracle normalizer canonicalizes decimal strings.
// Authored by: OpenCode
type comparableOutputValuesInput struct {
	RealizedGainOrLoss string `json:"realized_gain_or_loss"`
	AllocatedBasis     string `json:"allocated_basis"`
	ClosingQuantity    string `json:"closing_quantity"`
	ClosingBasis       string `json:"closing_basis"`
}

// oracleMatchEvidenceInput stores one raw comparable match-evidence row before
// canonicalization and stable sorting.
// Authored by: OpenCode
type oracleMatchEvidenceInput struct {
	DisposedSourceID    string                       `json:"disposed_source_id"`
	AcquisitionSourceID string                       `json:"acquisition_source_id,omitempty"`
	ScopeID             string                       `json:"scope_id,omitempty"`
	MatchedQuantity     string                       `json:"matched_quantity"`
	MatchedBasis        string                       `json:"matched_basis"`
	MatchedProceeds     string                       `json:"matched_proceeds,omitempty"`
	MatchedGainOrLoss   string                       `json:"matched_gain_or_loss,omitempty"`
	SupportLabel        fixture.EvidenceSupportLabel `json:"support_label,omitempty"`
	CompositionRuleID   string                       `json:"composition_rule_id,omitempty"`
}

// unsupportedOracleSegmentInput stores one raw unsupported segment before the
// normalizer copies it into the shared fixture model.
// Authored by: OpenCode
type unsupportedOracleSegmentInput struct {
	CaseID            string                      `json:"case_id"`
	Method            reportmodel.CostBasisMethod `json:"method"`
	ActivitySourceIDs []string                    `json:"activity_source_ids"`
	Reason            string                      `json:"reason"`
	ComparisonPolicy  fixture.ComparisonPolicy    `json:"comparison_policy"`
}

// oracleGenerationMetadataInput stores one raw generation metadata block before
// normalization applies canonical decimal and hash rules.
// Authored by: OpenCode
type oracleGenerationMetadataInput struct {
	RunID                   string
	OracleName              string
	SourceURL               string
	VersionOrCommit         string
	AdapterArguments        []string
	AdapterConstraints      []string
	DatasetInputHash        string
	ExternalOracleInputHash string
	DecimalPolicy           string
	CompositeRuleVersion    string
	FinancialTolerances     map[string]string
	ToleranceNotes          map[string]string
	GeneratedAt             string
}

// normalizeOracleOutput converts raw external-oracle-derived values into one
// validated normalized oracle fixture with a deterministic stable hash.
// Authored by: OpenCode
func normalizeOracleOutput(input oracleOutputNormalizationInput) (fixture.OracleOutput, error) {
	var output fixture.OracleOutput
	var err error

	output = fixture.OracleOutput{
		FixtureVersion:      defaultOracleFixtureVersion(input.FixtureVersion),
		DatasetVersion:      strings.TrimSpace(input.DatasetVersion),
		CaseID:              strings.TrimSpace(input.CaseID),
		Method:              input.Method,
		Year:                input.Year,
		AssetIdentityKey:    strings.TrimSpace(input.AssetIdentityKey),
		Matches:             make([]fixture.OracleMatchEvidence, 0, len(input.Matches)),
		UnsupportedSegments: make([]fixture.UnsupportedOracleSegment, 0, len(input.UnsupportedSegments)),
		Metadata: fixture.OracleGenerationRun{
			RunID:                   strings.TrimSpace(input.Metadata.RunID),
			OracleName:              strings.TrimSpace(input.Metadata.OracleName),
			SourceURL:               strings.TrimSpace(input.Metadata.SourceURL),
			VersionOrCommit:         strings.TrimSpace(input.Metadata.VersionOrCommit),
			AdapterArguments:        copyStringSlice(input.Metadata.AdapterArguments),
			AdapterConstraints:      copyStringSlice(input.Metadata.AdapterConstraints),
			DatasetInputHash:        strings.TrimSpace(input.Metadata.DatasetInputHash),
			ExternalOracleInputHash: strings.TrimSpace(input.Metadata.ExternalOracleInputHash),
			DecimalPolicy:           strings.TrimSpace(input.Metadata.DecimalPolicy),
			NormalizationVersion:    oracleOutputNormalizationVersion,
			CompositeRuleVersion:    strings.TrimSpace(input.Metadata.CompositeRuleVersion),
			FinancialTolerances:     copyStringMap(input.Metadata.FinancialTolerances),
			ToleranceNotes:          copyStringMap(input.Metadata.ToleranceNotes),
			GeneratedAt:             strings.TrimSpace(input.Metadata.GeneratedAt),
		},
	}

	output.Values, err = normalizeComparableOutputValues(input.Values)
	if err != nil {
		return fixture.OracleOutput{}, err
	}

	output.Matches, err = normalizeOracleMatches(input.Matches)
	if err != nil {
		return fixture.OracleOutput{}, err
	}

	output.UnsupportedSegments = normalizeUnsupportedOracleSegments(input.UnsupportedSegments)
	output.Metadata.FinancialTolerances, err = normalizeFinancialTolerances(input.Metadata.FinancialTolerances)
	if err != nil {
		return fixture.OracleOutput{}, err
	}

	var hash string
	hash, err = stableOracleOutputHash(output)
	if err != nil {
		return fixture.OracleOutput{}, err
	}
	output.Metadata.OracleOutputHash = hash

	var rawContent []byte
	rawContent, err = json.Marshal(output)
	if err != nil {
		return fixture.OracleOutput{}, fmt.Errorf("marshal normalized oracle output: %w", err)
	}
	if err := fixture.ValidateOracleOutput("generated oracle output", string(rawContent), output); err != nil {
		return fixture.OracleOutput{}, fmt.Errorf("validate normalized oracle output: %w", err)
	}

	return output, nil
}

// stablePrefixedSHA256Hash returns the stable `sha256:`-prefixed hash for one
// unchanged input byte slice.
// Authored by: OpenCode
func stablePrefixedSHA256Hash(content []byte) string {
	var digest = sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// stableOracleOutputHash returns the deterministic stable hash for one
// normalized oracle output fixture.
// Authored by: OpenCode
func stableOracleOutputHash(output fixture.OracleOutput) (string, error) {
	var hash, err = fixture.StableOracleOutputHash(output)
	if err != nil {
		return "", fmt.Errorf("stable oracle output hash: %w", err)
	}

	return hash, nil
}

// defaultOracleFixtureVersion returns the repository fixture version used when
// the caller leaves the raw input empty.
// Authored by: OpenCode
func defaultOracleFixtureVersion(raw string) string {
	var trimmed = strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}

	return oracleOutputFixtureVersion
}

// normalizeComparableOutputValues canonicalizes the raw top-level comparable
// values into the shared fixture shape.
// Authored by: OpenCode
func normalizeComparableOutputValues(input comparableOutputValuesInput) (fixture.ComparableOutputValues, error) {
	var values fixture.ComparableOutputValues
	var err error

	values.RealizedGainOrLoss, err = normalizeRequiredDecimalString(input.RealizedGainOrLoss)
	if err != nil {
		return fixture.ComparableOutputValues{}, fmt.Errorf("normalize realized_gain_or_loss: %w", err)
	}
	values.AllocatedBasis, err = normalizeRequiredDecimalString(input.AllocatedBasis)
	if err != nil {
		return fixture.ComparableOutputValues{}, fmt.Errorf("normalize allocated_basis: %w", err)
	}
	values.ClosingQuantity, err = normalizeRequiredDecimalString(input.ClosingQuantity)
	if err != nil {
		return fixture.ComparableOutputValues{}, fmt.Errorf("normalize closing_quantity: %w", err)
	}
	values.ClosingBasis, err = normalizeRequiredDecimalString(input.ClosingBasis)
	if err != nil {
		return fixture.ComparableOutputValues{}, fmt.Errorf("normalize closing_basis: %w", err)
	}

	return values, nil
}

// normalizeOracleMatches canonicalizes and stable-sorts raw match evidence.
// Authored by: OpenCode
func normalizeOracleMatches(matches []oracleMatchEvidenceInput) ([]fixture.OracleMatchEvidence, error) {
	var normalized = make([]fixture.OracleMatchEvidence, 0, len(matches))
	var index int

	for index = range matches {
		var match fixture.OracleMatchEvidence
		var err error

		match = fixture.OracleMatchEvidence{
			DisposedSourceID:    strings.TrimSpace(matches[index].DisposedSourceID),
			AcquisitionSourceID: strings.TrimSpace(matches[index].AcquisitionSourceID),
			ScopeID:             strings.TrimSpace(matches[index].ScopeID),
			SupportLabel:        matches[index].SupportLabel,
			CompositionRuleID:   strings.TrimSpace(matches[index].CompositionRuleID),
		}

		match.MatchedQuantity, err = normalizeRequiredDecimalString(matches[index].MatchedQuantity)
		if err != nil {
			return nil, fmt.Errorf("normalize match %d matched_quantity: %w", index, err)
		}
		match.MatchedBasis, err = normalizeRequiredDecimalString(matches[index].MatchedBasis)
		if err != nil {
			return nil, fmt.Errorf("normalize match %d matched_basis: %w", index, err)
		}
		match.MatchedProceeds, err = normalizeOptionalDecimalString(matches[index].MatchedProceeds)
		if err != nil {
			return nil, fmt.Errorf("normalize match %d matched_proceeds: %w", index, err)
		}
		match.MatchedGainOrLoss, err = normalizeOptionalDecimalString(matches[index].MatchedGainOrLoss)
		if err != nil {
			return nil, fmt.Errorf("normalize match %d matched_gain_or_loss: %w", index, err)
		}

		normalized = append(normalized, match)
	}

	sort.Slice(normalized, func(left int, right int) bool {
		return oracleMatchSortKey(normalized[left]) < oracleMatchSortKey(normalized[right])
	})

	return normalized, nil
}

// normalizeUnsupportedOracleSegments copies and stable-sorts unsupported
// segments for deterministic fixture output.
// Authored by: OpenCode
func normalizeUnsupportedOracleSegments(segments []unsupportedOracleSegmentInput) []fixture.UnsupportedOracleSegment {
	var normalized = make([]fixture.UnsupportedOracleSegment, 0, len(segments))
	var index int

	for index = range segments {
		var activitySourceIDs = copyStringSlice(segments[index].ActivitySourceIDs)
		sort.Strings(activitySourceIDs)

		normalized = append(normalized, fixture.UnsupportedOracleSegment{
			CaseID:            strings.TrimSpace(segments[index].CaseID),
			Method:            segments[index].Method,
			ActivitySourceIDs: activitySourceIDs,
			Reason:            strings.TrimSpace(segments[index].Reason),
			ComparisonPolicy:  segments[index].ComparisonPolicy,
		})
	}

	sort.Slice(normalized, func(left int, right int) bool {
		return unsupportedOracleSegmentSortKey(normalized[left]) < unsupportedOracleSegmentSortKey(normalized[right])
	})

	return normalized
}

// normalizeFinancialTolerances canonicalizes every persisted financial-tolerance
// value into the fixed-point form required by fixtures.
// Authored by: OpenCode
func normalizeFinancialTolerances(tolerances map[string]string) (map[string]string, error) {
	var normalized = copyStringMap(tolerances)

	for field, rawValue := range normalized {
		var canonicalValue, err = normalizeRequiredDecimalString(rawValue)
		if err != nil {
			return nil, fmt.Errorf("normalize financial_tolerances.%s: %w", field, err)
		}

		normalized[field] = canonicalValue
	}

	return normalized, nil
}

// normalizeRequiredDecimalString canonicalizes one required raw decimal string.
// Authored by: OpenCode
func normalizeRequiredDecimalString(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("decimal value is required")
	}

	var _, canonical, err = decimalsupport.ParseString(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse decimal value: %w", err)
	}

	return canonical, nil
}

// normalizeOptionalDecimalString canonicalizes one optional raw decimal string.
// Authored by: OpenCode
func normalizeOptionalDecimalString(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}

	return normalizeRequiredDecimalString(raw)
}

// oracleMatchSortKey returns the stable sort key for one normalized match row.
// Authored by: OpenCode
func oracleMatchSortKey(match fixture.OracleMatchEvidence) string {
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

// unsupportedOracleSegmentSortKey returns the stable sort key for one
// normalized unsupported segment.
// Authored by: OpenCode
func unsupportedOracleSegmentSortKey(segment fixture.UnsupportedOracleSegment) string {
	return strings.Join([]string{
		segment.CaseID,
		string(segment.Method),
		strings.Join(segment.ActivitySourceIDs, "\x01"),
		segment.Reason,
		string(segment.ComparisonPolicy),
	}, "\x00")
}

// copyStringSlice returns one non-nil copy of a string slice.
// Authored by: OpenCode
func copyStringSlice(values []string) []string {
	var copied = make([]string, len(values))
	copy(copied, values)
	return copied
}

// copyStringMap returns one non-nil copy of a string map.
// Authored by: OpenCode
func copyStringMap(values map[string]string) map[string]string {
	var copied = make(map[string]string, len(values))

	for key, value := range values {
		copied[key] = value
	}

	return copied
}
