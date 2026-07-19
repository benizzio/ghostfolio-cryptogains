// Package testutil contains deterministic fixtures for report presentation
// acceptance tests.
// Authored by: OpenCode
package testutil

// ReportPresentationFormat identifies one supported report-rendering attempt.
// The public fixture uses ReportPresentationFormatMarkdown for the main and
// Annex Markdown documents and ReportPresentationFormatPDF for the combined
// PDF document; the internal cross-format value is reserved for parity keys.
// Authored by: OpenCode
type ReportPresentationFormat string

const (
	// ReportPresentationLegalWarningText is the exact standalone warning used
	// by both main-report formats.
	// Authored by: OpenCode
	ReportPresentationLegalWarningText = "The data in this report does not follow any legally required rules for any country's tax returns and is for reference only."
	// ReportPresentationFormatMarkdown identifies the Markdown bundle attempt.
	// Authored by: OpenCode
	ReportPresentationFormatMarkdown ReportPresentationFormat = "markdown"
	// ReportPresentationFormatPDF identifies the combined PDF attempt.
	// Authored by: OpenCode
	ReportPresentationFormatPDF ReportPresentationFormat = "pdf"
	// reportPresentationFormatCrossFormat identifies a parity comparison.
	// Authored by: OpenCode
	reportPresentationFormatCrossFormat ReportPresentationFormat = "cross-format"
)

// ReportPresentationDocumentRole identifies the logical report section used by
// a semantic occurrence key. Main and Annex identify the separate Markdown
// documents, while Combined identifies the single PDF document; the internal
// model role is used only for cross-format integrity evidence.
// Authored by: OpenCode
type ReportPresentationDocumentRole string

const (
	// ReportPresentationDocumentRoleMain identifies the main report section.
	// Authored by: OpenCode
	ReportPresentationDocumentRoleMain ReportPresentationDocumentRole = "main"
	// ReportPresentationDocumentRoleAnnex identifies the Annex 1 section.
	// Authored by: OpenCode
	ReportPresentationDocumentRoleAnnex ReportPresentationDocumentRole = "annex"
	// ReportPresentationDocumentRoleCombined identifies the combined PDF document.
	// Authored by: OpenCode
	ReportPresentationDocumentRoleCombined ReportPresentationDocumentRole = "combined"
	// reportPresentationDocumentRoleModel identifies a model comparison.
	// Authored by: OpenCode
	reportPresentationDocumentRoleModel ReportPresentationDocumentRole = "model"
)

// ReportPresentationCaseKind identifies one closed acceptance-case family.
// Each value selects the fixture controls and semantic occurrence population
// expected for that family, so callers should compare the typed value rather
// than infer behavior from its string representation.
// Authored by: OpenCode
type ReportPresentationCaseKind string

const (
	// ReportPresentationCaseKindWarning identifies the wrapped-warning case.
	// Authored by: OpenCode
	ReportPresentationCaseKindWarning ReportPresentationCaseKind = "warning"
	// ReportPresentationCaseKindFinancial identifies a matrix financial case.
	// Authored by: OpenCode
	ReportPresentationCaseKindFinancial ReportPresentationCaseKind = "financial"
	// ReportPresentationCaseKindQuantity identifies a quantity case.
	// Authored by: OpenCode
	ReportPresentationCaseKindQuantity ReportPresentationCaseKind = "quantity"
	// ReportPresentationCaseKindRate identifies a normalized-rate case.
	// Authored by: OpenCode
	ReportPresentationCaseKindRate ReportPresentationCaseKind = "rate"
	// ReportPresentationCaseKindBoolean identifies a structured-boolean case.
	// Authored by: OpenCode
	ReportPresentationCaseKindBoolean ReportPresentationCaseKind = "boolean"
	// ReportPresentationCaseKindCurrency identifies an audit-currency case.
	// Authored by: OpenCode
	ReportPresentationCaseKindCurrency ReportPresentationCaseKind = "currency"
	// ReportPresentationCaseKindConverted identifies a conversion-sequence case.
	// Authored by: OpenCode
	ReportPresentationCaseKindConverted ReportPresentationCaseKind = "converted"
)

// ReportPresentationPopulation identifies an acceptance denominator. Counters
// are keyed by this type and are derived directly from occurrence keys, which
// keeps the denominator mapping explicit and rejects unrecognized populations
// at the contract layer.
// Authored by: OpenCode
type ReportPresentationPopulation string

const (
	// ReportPresentationPopulationWarning identifies warning occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationWarning ReportPresentationPopulation = "W"
	// ReportPresentationPopulationVisibleFinancial identifies present financial fields.
	// Authored by: OpenCode
	ReportPresentationPopulationVisibleFinancial ReportPresentationPopulation = "V"
	// ReportPresentationPopulationModelIntegrity identifies model comparisons.
	// Authored by: OpenCode
	ReportPresentationPopulationModelIntegrity ReportPresentationPopulation = "M"
	// ReportPresentationPopulationQuantity identifies quantity occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationQuantity ReportPresentationPopulation = "Q"
	// ReportPresentationPopulationBoolean identifies structured boolean occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationBoolean ReportPresentationPopulation = "B"
	// ReportPresentationPopulationClassifiedCurrency identifies classified currency controls.
	// Authored by: OpenCode
	ReportPresentationPopulationClassifiedCurrency ReportPresentationPopulation = "Z"
	// ReportPresentationPopulationUnclassified identifies unclassified currency controls.
	// Authored by: OpenCode
	ReportPresentationPopulationUnclassified ReportPresentationPopulation = "N"
	// ReportPresentationPopulationConversionRow identifies conversion-row occurrences.
	// Authored by: OpenCode
	ReportPresentationPopulationConversionRow ReportPresentationPopulation = "C"
	// ReportPresentationPopulationParity identifies cross-format parity items.
	// Authored by: OpenCode
	ReportPresentationPopulationParity ReportPresentationPopulation = "P"
	// ReportPresentationPopulationConvertedEntry identifies included conversion entries.
	// Authored by: OpenCode
	ReportPresentationPopulationConvertedEntry ReportPresentationPopulation = "E"
)

// ReportPresentationFinancialField describes one semantic financial field in a
// matrix row, including its amount kind and ordinal within a repeated group.
// Name identifies the visible field, AmountKind identifies the financial
// presentation policy, AmountOrdinal preserves the expected occurrence order
// for repeated values, and Nullable marks fields whose visible value can be
// blank while the containing row remains applicable.
// Authored by: OpenCode
type ReportPresentationFinancialField struct {
	Name          string
	AmountKind    string
	AmountOrdinal int
	Nullable      bool
}

// ReportPresentationFormatAttempt describes one format attempt for an
// acceptance case. Format identifies the renderer under test and
// DocumentRoles identifies the logical documents it must produce. The closed
// fixture invariant is two attempts per case: Markdown with Main and Annex,
// and PDF with Combined.
// Authored by: OpenCode
type ReportPresentationFormatAttempt struct {
	Format        ReportPresentationFormat
	DocumentRoles []ReportPresentationDocumentRole
}

// ReportPresentationOccurrenceKey identifies one semantic occurrence without
// relying on substring counts in generated document text. Its typed population,
// format, document role, section, source identity, field, amount kind, and
// ordinal locate the expected value or parity assertion. Empty dimensions are
// intentional for controls that do not represent a field or repeated amount.
// Authored by: OpenCode
type ReportPresentationOccurrenceKey struct {
	Population          ReportPresentationPopulation
	CaseID              string
	Format              ReportPresentationFormat
	DocumentRole        ReportPresentationDocumentRole
	Section             string
	AssetIdentity       string
	SourceOrRowIdentity string
	FieldName           string
	AmountKind          string
	AmountOrdinal       int
}

// ReportPresentationAcceptanceCase stores one closed case, its exact source
// control, both format attempts, and all semantic occurrence keys expected from
// those attempts. ExactValue describes the synthetic model input and
// ExpectedVisibleValue or ExpectedText describes the renderer result; Absent,
// Omitted, and the zero-priced classification distinguish applicability
// controls from displayed zero values. The fixture builder preserves case and
// occurrence order and supplies fresh slices for each generated manifest.
// Authored by: OpenCode
type ReportPresentationAcceptanceCase struct {
	ID                              string
	Kind                            ReportPresentationCaseKind
	Section                         string
	FinancialFieldClass             string
	VectorCase                      string
	ExactValue                      string
	ExpectedVisibleValue            string
	ExpectedText                    string
	Absent                          bool
	Omitted                         bool
	BooleanValue                    bool
	HasBooleanValue                 bool
	IsZeroPricedHoldingReduction    bool
	HasZeroPricedClassification     bool
	PreFormatActivityCurrency       string
	VisibleOriginalActivityCurrency string
	FinancialFields                 []ReportPresentationFinancialField
	OmittedFieldNames               []string
	ConvertedAmountKinds            []string
	Attempts                        []ReportPresentationFormatAttempt
	OccurrenceKeys                  []ReportPresentationOccurrenceKey
}

// ReportPresentationAcceptanceCounters reports the derived denominators for
// the closed acceptance manifest. CaseCount equals the number of cases, and
// Populations counts occurrences directly by typed population from those cases;
// it is not a second hand-maintained classification table.
// Authored by: OpenCode
type ReportPresentationAcceptanceCounters struct {
	CaseCount   int
	Populations map[ReportPresentationPopulation]int
}

// ReportPresentationAcceptanceManifest contains the closed-shape case set and
// its semantic population counters. Cases remain in contract order and
// Counters is derived from the same occurrence keys. The public fixture
// constructor returns fresh case and slice storage on each call, so callers may
// inspect or annotate one manifest without changing another test's fixture.
// Authored by: OpenCode
type ReportPresentationAcceptanceManifest struct {
	Cases    []ReportPresentationAcceptanceCase
	Counters ReportPresentationAcceptanceCounters
}
