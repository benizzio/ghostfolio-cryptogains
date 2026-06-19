// Package fixture contains empirical case, case-field, and case list parsing for
// the constrained empirical dataset YAML parser.
//
// Authored by: OpenCode
package fixture

import (
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// datasetYAMLCaseParser owns empirical case parsing while sharing the parent
// parser cursor.
// Authored by: OpenCode
type datasetYAMLCaseParser struct {
	cursor *datasetYAMLParser
}

// parseCases parses the top-level empirical case list.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCases(parentLine datasetYAMLLine, rawValue string) ([]EmpiricalCase, error) {
	var trimmedValue = strings.TrimSpace(rawValue)
	if trimmedValue == "[]" {
		caseParser.cursor.index++
		return []EmpiricalCase{}, nil
	}
	if trimmedValue != "" {
		return nil, caseParser.cursor.newError(parentLine, "", "", "cases", "expected block list or []")
	}

	caseParser.cursor.index++
	return caseParser.parseCaseItems(parentLine)
}

// parseCaseItems parses all block case items under a parent list field.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCaseItems(parentLine datasetYAMLLine) ([]EmpiricalCase, error) {
	var cases = make([]EmpiricalCase, 0)

	for caseParser.cursor.index < len(caseParser.cursor.lines) {
		var line = caseParser.cursor.lines[caseParser.cursor.index]
		if line.Indent < parentLine.Indent+2 {
			break
		}
		if line.Indent != parentLine.Indent+2 || !strings.HasPrefix(line.Text, "- ") {
			return nil, caseParser.cursor.newError(line, "", "", "cases", "expected case list item")
		}

		var caseRecord EmpiricalCase
		var err = caseParser.parseCase(line, &caseRecord)
		if err != nil {
			return nil, err
		}
		cases = append(cases, caseRecord)
	}

	return cases, nil
}

// parseCase parses one empirical case item.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCase(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	var err = caseParser.parseCaseFirstField(startLine, caseRecord)
	if err != nil {
		return err
	}

	caseParser.cursor.index++
	return caseParser.parseCaseFields(startLine, caseRecord)
}

// parseCaseFirstField decodes the case field embedded in the list marker.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCaseFirstField(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	var firstField = strings.TrimSpace(strings.TrimPrefix(startLine.Text, "- "))
	var field, rawValue, ok = splitYAMLField(firstField)
	if !ok {
		return caseParser.cursor.newError(startLine, "", "", "cases", "expected case field")
	}

	return caseParser.applyCaseField(startLine, caseRecord, field, rawValue)
}

// parseCaseFields parses all indented fields that belong to one case.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCaseFields(startLine datasetYAMLLine, caseRecord *EmpiricalCase) error {
	for caseParser.cursor.index < len(caseParser.cursor.lines) {
		var line = caseParser.cursor.lines[caseParser.cursor.index]
		if line.Indent < startLine.Indent+2 {
			break
		}

		var err = caseParser.parseCaseFieldLine(startLine, line, caseRecord)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseCaseFieldLine decodes one nested case field line.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCaseFieldLine(startLine datasetYAMLLine, line datasetYAMLLine, caseRecord *EmpiricalCase) error {
	if line.Indent != startLine.Indent+2 {
		return caseParser.cursor.newError(line, "case_id", caseRecord.CaseID, "cases", "unexpected nested indentation")
	}

	var field, rawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return caseParser.cursor.newError(line, "case_id", caseRecord.CaseID, "cases", "expected case field")
	}

	switch field {
	case "methods":
		return caseParser.parseCaseMethodsField(line, caseRecord, field, rawValue)
	case "asset_identity_keys":
		return caseParser.parseCaseStringListField(line, caseRecord, field, rawValue, &caseRecord.AssetIdentityKeys)
	case "activity_source_ids":
		return caseParser.parseCaseStringListField(line, caseRecord, field, rawValue, &caseRecord.ActivitySourceIDs)
	case "coverage_tags":
		return caseParser.parseCaseStringListField(line, caseRecord, field, rawValue, &caseRecord.CoverageTags)
	default:
		return caseParser.parseCaseScalarField(line, caseRecord, field, rawValue)
	}
}

// parseCaseMethodsField decodes one case methods list field.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCaseMethodsField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var listParser = datasetYAMLListParser{cursor: caseParser.cursor}
	var values, err = listParser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
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
func (caseParser *datasetYAMLCaseParser) parseCaseStringListField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string, destination *[]string) error {
	var listParser = datasetYAMLListParser{cursor: caseParser.cursor}
	var values, err = listParser.parseStringList(line, rawValue, field, "case_id", caseRecord.CaseID, false)
	if err != nil {
		return err
	}

	*destination = values
	return nil
}

// parseCaseScalarField decodes one scalar case field and advances the cursor.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) parseCaseScalarField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var err = caseParser.applyCaseField(line, caseRecord, field, rawValue)
	if err != nil {
		return err
	}

	caseParser.cursor.index++
	return nil
}

// applyCaseField decodes one scalar empirical case field.
// Authored by: OpenCode
func (caseParser *datasetYAMLCaseParser) applyCaseField(line datasetYAMLLine, caseRecord *EmpiricalCase, field string, rawValue string) error {
	var err error

	switch field {
	case "case_id":
		caseRecord.CaseID, err = parseYAMLScalarText(rawValue)
	case "description":
		caseRecord.Description, err = parseYAMLScalarText(rawValue)
	case "year":
		caseRecord.Year, err = parseYAMLInteger(rawValue)
		if err != nil {
			return caseParser.cursor.newError(line, "case_id", caseRecord.CaseID, field, err.Error())
		}
		return nil
	case "oracle_support":
		var value string
		value, err = parseYAMLScalarText(rawValue)
		caseRecord.OracleSupport = OracleSupport(value)
	case "unsupported_reason":
		caseRecord.UnsupportedReason, err = parseYAMLScalarText(rawValue)
	default:
		return caseParser.cursor.newError(line, "case_id", caseRecord.CaseID, field, "unknown case field")
	}

	if err != nil {
		return caseParser.cursor.newError(line, "case_id", caseRecord.CaseID, field, err.Error())
	}

	return nil
}
