// Package fixture contains empirical case, case-field, and case list parsing for
// the constrained empirical dataset YAML parser.
//
// Authored by: OpenCode
package fixture

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// parseCases parses the top-level empirical case list.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCases(parentLine datasetYAMLLine, rawValue string) ([]EmpiricalCase, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		parser.index++
		return []EmpiricalCase{}, nil
	}
	if trimmedValue != "" {
		return nil, parser.newError(parentLine, "", "", "cases", "expected block list or []")
	}

	parser.index++
	return parser.parseCaseItems(parentLine)
}

// parseCaseItems parses all block case items under a parent list field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseItems(parentLine datasetYAMLLine) ([]EmpiricalCase, error) {
	var cases = make([]EmpiricalCase, 0)

	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
			return nil, parser.newError(line, "", "", "cases", "expected case list item")
		}

		var caseRecord EmpiricalCase
		var err = parser.parseCase(line, &caseRecord)
		if err != nil {
			return nil, err
		}
		cases = append(cases, caseRecord)
	}

	return cases, nil
}

// parseCase parses one empirical case item.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCase(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	var err = parser.parseCaseFirstField(startLine, caseRecord)
	if err != nil {
		return err
	}

	parser.index++
	return parser.parseCaseFields(startLine, caseRecord)
}

// parseCaseFirstField decodes the case field embedded in the list marker.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseFirstField(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	var firstField = strings.TrimSpace(strings.TrimPrefix(startLine.Text, "- "))
	var field, rawValue, ok = splitYAMLField(firstField)
	if !ok {
		return parser.newError(startLine, "", "", "cases", "expected case field")
	}

	return parser.applyCaseField(startLine, caseRecord, field, rawValue)
}

// parseCaseFields parses all indented fields that belong to one case.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseFields(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	for parser.index < len(parser.lines) {
		var line = parser.lines[parser.index]
		if line.Indent < startLine.Indent+2 {
			break
		}

		var err = parser.parseCaseFieldLine(startLine, line, caseRecord)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseCaseFieldLine decodes one nested case field line.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseFieldLine(startLine datasetYAMLLine, line datasetYAMLLine, caseRecord *EmpiricalCase) error {
	if line.Indent != startLine.Indent+2 {
		return parser.newError(line, "case_id", caseRecord.CaseID, "cases", "unexpected nested indentation")
	}

	var field, rawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return parser.newError(line, "case_id", caseRecord.CaseID, "cases", "expected case field")
	}

	switch field {
	case "methods":
		return parser.parseCaseMethodsField(line, caseRecord, field, rawValue)
	case "asset_identity_keys":
		return parser.parseCaseStringListField(line, caseRecord, field, rawValue, &caseRecord.AssetIdentityKeys)
	case "activity_source_ids":
		return parser.parseCaseStringListField(line, caseRecord, field, rawValue, &caseRecord.ActivitySourceIDs)
	case "coverage_tags":
		return parser.parseCaseStringListField(line, caseRecord, field, rawValue, &caseRecord.CoverageTags)
	default:
		return parser.parseCaseScalarField(line, caseRecord, field, rawValue)
	}
}

// parseCaseMethodsField decodes one case methods list field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseMethodsField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var values, err = parser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
	if err != nil {
		return err
	}

	caseRecord.Methods = caseCostBasisMethodsFromStrings(values)
	return nil
}

// caseCostBasisMethodsFromStrings converts raw case method labels to model values.
// Authored by: OpenCode
func caseCostBasisMethodsFromStrings(values []string) []reportmodel.CostBasisMethod {
	var methods = make([]reportmodel.CostBasisMethod, len(values))
	var index int

	for index = range values {
		methods[index] = reportmodel.CostBasisMethod(values[index])
	}

	return methods
}

// parseCaseStringListField decodes one case string-list field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseStringListField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string, destination *[]string) error {
	var values, err = parser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
	if err != nil {
		return err
	}

	*destination = values
	return nil
}

// parseCaseScalarField decodes one scalar case field and advances the cursor.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseCaseScalarField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var err = parser.applyCaseField(line, caseRecord, field, rawValue)
	if err != nil {
		return err
	}

	parser.index++
	return nil
}

// applyCaseField decodes one scalar empirical case field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) applyCaseField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var err error

	switch field {
	case "case_id":
		caseRecord.CaseID, err = parseYAMLScalarText(rawValue)
	case "description":
		caseRecord.Description, err = parseYAMLScalarText(rawValue)
	case "year":
		caseRecord.Year, err = parseYAMLInteger(rawValue)
		if err != nil {
			return parser.newError(line, "case_id", caseRecord.CaseID, field, err.Error())
		}
		return nil
	case "oracle_support":
		var value string
		value, err = parseYAMLScalarText(rawValue)
		caseRecord.OracleSupport = OracleSupport(value)
	case "unsupported_reason":
		caseRecord.UnsupportedReason, err = parseYAMLScalarText(rawValue)
	default:
		return parser.newError(line, "case_id", caseRecord.CaseID, field, "unknown case field")
	}

	if err != nil {
		return parser.newError(line, "case_id", caseRecord.CaseID, field, err.Error())
	}

	return nil
}
