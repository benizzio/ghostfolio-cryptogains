// Package fixture provides empirical dataset fixtures, parsers, validators, and
// comparison helpers for integration tests.
// Authored by: OpenCode
package fixture

import (
	"fmt"
	"os"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
)

// LoadEmpiricalDataset reads one persisted empirical dataset file and parses it
// through the constrained project-owned YAML parser.
//
// Example:
//
//	dataset, rawContent, err := fixture.LoadEmpiricalDataset("testdata/empirical/financial-dataset.yaml")
//	if err != nil {
//		panic(err)
//	}
//	_, _ = dataset, rawContent
//
// Authored by: OpenCode
func LoadEmpiricalDataset(path string) (EmpiricalDataset, string, error) {
	var rawContent, err = os.ReadFile(path)
	if err != nil {
		return EmpiricalDataset{}, "", fmt.Errorf("read empirical dataset %s: %w", path, err)
	}

	var dataset EmpiricalDataset
	dataset, err = ParseEmpiricalDataset(path, rawContent)
	if err != nil {
		return EmpiricalDataset{}, "", err
	}

	return dataset, string(rawContent), nil
}

// ParseEmpiricalDataset parses one constrained project-owned YAML dataset fixture
// into the shared empirical dataset models.
//
// Example:
//
//	dataset, err := fixture.ParseEmpiricalDataset(path, rawContent)
//	if err != nil {
//		panic(err)
//	}
//	_ = dataset
//
// ParseEmpiricalDataset accepts only the repository-owned schema used by the
// empirical test suite. It rejects decimal fields that are not quoted strings.
// Authored by: OpenCode
func ParseEmpiricalDataset(path string, content []byte) (EmpiricalDataset, error) {
	var parser = newDatasetYAMLParser(path, string(content))
	return parser.parse()
}

// parse decodes the full constrained empirical dataset document.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parse() (EmpiricalDataset, error) {
	var dataset EmpiricalDataset

	for parser.index < len(parser.lines) {
		var err = parser.parseTopLevelLine(&dataset)
		if err != nil {
			return EmpiricalDataset{}, err
		}
	}

	return dataset, nil
}

// parseTopLevelLine decodes the field at the parser cursor into the dataset.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseTopLevelLine(dataset *EmpiricalDataset) error {
	var line = parser.lines[parser.index]
	var field, rawValue, err = parser.readTopLevelField(line)
	if err != nil {
		return err
	}

	return parser.applyTopLevelField(line, dataset, field, rawValue)
}

// readTopLevelField verifies and splits one top-level YAML field line.
// Authored by: OpenCode
func (parser *datasetYAMLParser) readTopLevelField(line datasetYAMLLine) (string, string, error) {
	if line.Indent != 0 {
		return "", "", parser.newError(line, "", "", "", "expected top-level field")
	}

	var field, rawValue, ok = splitYAMLField(line.Text)
	if !ok {
		return "", "", parser.newError(line, "", "", "", "expected YAML field")
	}

	return field, rawValue, nil
}

// applyTopLevelField dispatches one top-level dataset field to its parser.
// Authored by: OpenCode
func (parser *datasetYAMLParser) applyTopLevelField(line datasetYAMLLine, dataset *EmpiricalDataset, field string, rawValue string) error {
	switch field {
	case "dataset_version":
		var value, err = parser.parseDatasetScalarField(line, field, rawValue)
		dataset.DatasetVersion = value
		return err
	case "description":
		var value, err = parser.parseDatasetScalarField(line, field, rawValue)
		dataset.Description = value
		return err
	case "currency":
		var value, err = parser.parseDatasetScalarField(line, field, rawValue)
		dataset.Currency = value
		return err
	case "supported_years":
		var values, err = parser.parseIntegerList(line, rawValue, field, "", "")
		dataset.SupportedYears = values
		return err
	case "supported_methods":
		var values, err = parser.parseSupportedMethods(line, rawValue, field)
		dataset.SupportedMethods = values
		return err
	case "coverage_tags":
		var values, err = parser.parseStringList(line, rawValue, field, "", "", false)
		dataset.CoverageTags = values
		return err
	case "activities":
		var values, err = parser.parseActivities(line, rawValue)
		dataset.Activities = values
		return err
	case "cases":
		var values, err = parser.parseCases(line, rawValue)
		dataset.Cases = values
		return err
	default:
		return parser.newError(line, "", "", field, "unknown top-level field")
	}
}

// parseDatasetScalarField decodes one top-level scalar dataset field.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseDatasetScalarField(line datasetYAMLLine, field string, rawValue string) (string, error) {
	var value, err = parseYAMLScalarText(rawValue)
	if err != nil {
		return "", parser.newError(line, "", "", field, err.Error())
	}

	parser.index++
	return value, nil
}

// parseSupportedMethods decodes the dataset-level cost basis method list.
// Authored by: OpenCode
func (parser *datasetYAMLParser) parseSupportedMethods(line datasetYAMLLine, rawValue string, field string) ([]reportmodel.CostBasisMethod, error) {
	var values, err = parser.parseStringList(line, rawValue, field, "", "", false)
	if err != nil {
		return nil, err
	}

	return costBasisMethodsFromStrings(values), nil
}

// costBasisMethodsFromStrings converts raw YAML method labels to model values.
// Authored by: OpenCode
func costBasisMethodsFromStrings(values []string) []reportmodel.CostBasisMethod {
	var methods = make([]reportmodel.CostBasisMethod, len(values))
	var index int

	for index = range values {
		methods[index] = reportmodel.CostBasisMethod(values[index])
	}

	return methods
}
