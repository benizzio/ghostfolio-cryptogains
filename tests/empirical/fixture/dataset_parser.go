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
