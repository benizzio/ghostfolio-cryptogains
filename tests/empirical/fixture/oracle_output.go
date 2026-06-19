// Package fixture provides empirical fixture loading and validation helpers.
// Authored by: OpenCode
package fixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadOracleOutputs reads and validates every oracle-output JSON fixture below
// one repository-controlled root directory.
//
// Example:
//
//	outputs, err := fixture.LoadOracleOutputs("testdata/empirical/golden")
//	if err != nil {
//		panic(err)
//	}
//	_ = outputs
//
// LoadOracleOutputs walks the root recursively, loads every `.json` file in
// stable path order, validates synthetic-only content, validates canonical
// decimal strings, and verifies the stored stable hash.
// Authored by: OpenCode
func LoadOracleOutputs(rootPath string) ([]OracleOutput, error) {
	var paths, err = collectOracleOutputPaths(rootPath)
	if err != nil {
		return nil, err
	}

	var outputs = make([]OracleOutput, 0, len(paths))
	var index int

	for index = range paths {
		var output OracleOutput
		output, _, err = LoadOracleOutput(paths[index])
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// LoadOracleOutput reads, parses, and validates one persisted oracle-output
// fixture file.
//
// Example:
//
//	output, rawContent, err := fixture.LoadOracleOutput("testdata/empirical/golden/fifo.json")
//	if err != nil {
//		panic(err)
//	}
//	_, _ = output, rawContent
//
// Authored by: OpenCode
func LoadOracleOutput(path string) (OracleOutput, string, error) {
	var rawContent, err = os.ReadFile(path)
	if err != nil {
		return OracleOutput{}, "", fmt.Errorf("read oracle output %s: %w", path, err)
	}

	var output OracleOutput
	output, err = ParseOracleOutput(path, rawContent)
	if err != nil {
		return OracleOutput{}, "", err
	}

	return output, string(rawContent), nil
}

// ParseOracleOutput parses one raw oracle-output JSON payload into the shared
// oracle fixture model and applies the persisted-fixture validation contract.
//
// Example:
//
//	output, err := fixture.ParseOracleOutput(path, rawContent)
//	if err != nil {
//		panic(err)
//	}
//	_ = output
//
// ParseOracleOutput rejects unknown JSON fields, trailing JSON content,
// float-style JSON numbers in string-decimal fields, and any persisted decimal
// text that is not already canonical.
// Authored by: OpenCode
func ParseOracleOutput(path string, content []byte) (OracleOutput, error) {
	if err := ValidateSyntheticOnlyContent(path, string(content)); err != nil {
		return OracleOutput{}, err
	}

	var decoder = json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var output OracleOutput
	if err := decoder.Decode(&output); err != nil {
		return OracleOutput{}, fmt.Errorf("decode oracle output JSON %s: %w", path, err)
	}

	var trailing struct{}
	var err = decoder.Decode(&trailing)
	if err == nil {
		return OracleOutput{}, fmt.Errorf("decode oracle output JSON %s: unexpected trailing JSON content", path)
	}
	if err != io.EOF {
		return OracleOutput{}, fmt.Errorf("decode oracle output JSON %s: %w", path, err)
	}

	err = validateOracleOutputStructure(path, output)
	if err != nil {
		return OracleOutput{}, err
	}

	return output, nil
}

// ValidateOracleOutput applies the oracle-output fixture contract to one
// already-parsed fixture value and its persisted raw JSON content.
//
// Example:
//
//	err := fixture.ValidateOracleOutput(path, rawContent, output)
//	if err != nil {
//		panic(err)
//	}
//
// ValidateOracleOutput enforces required metadata, canonical decimal strings,
// tolerance metadata, supported methods, match evidence rules, unsupported
// segment rules, and stable-hash verification.
// Authored by: OpenCode
func ValidateOracleOutput(path string, rawContent string, output OracleOutput) error {
	if err := ValidateSyntheticOnlyContent(path, rawContent); err != nil {
		return err
	}

	return validateOracleOutputStructure(path, output)
}
