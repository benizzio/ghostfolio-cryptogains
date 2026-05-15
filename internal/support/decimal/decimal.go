// Package decimal provides exact-decimal parsing and canonical string helpers
// for Ghostfolio transport and protected snapshot persistence.
// Authored by: OpenCode
package decimal

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// Service defines the exact-decimal helper boundary used by the sync and
// snapshot runtime.
//
// Example:
//
//	service := decimal.NewService()
//	_, canonical, err := service.ParseString("10.50")
//	if err != nil {
//		panic(err)
//	}
//	_ = canonical
//
// Authored by: OpenCode
type Service interface {
	ParseString(string) (apd.Decimal, string, error)
	ParseNumber(json.Number) (apd.Decimal, string, error)
	CanonicalString(apd.Decimal) (string, error)
	CanonicalStringPointer(*apd.Decimal) (string, error)
}

// exactService is the default exact-decimal helper implementation.
// Authored by: OpenCode
type exactService struct{}

// NewService creates the default exact-decimal helper service.
//
// Example:
//
//	service := decimal.NewService()
//	_, _, _ = service.ParseString("1")
//
// Authored by: OpenCode
func NewService() Service {
	return exactService{}
}

// ParseString parses one decimal string and returns both the canonical decimal
// value and its canonical persisted representation.
// Authored by: OpenCode
func (exactService) ParseString(raw string) (apd.Decimal, string, error) {
	return ParseString(raw)
}

// ParseNumber parses one JSON number and returns both the canonical decimal
// value and its canonical persisted representation.
// Authored by: OpenCode
func (exactService) ParseNumber(raw json.Number) (apd.Decimal, string, error) {
	return ParseNumber(raw)
}

// CanonicalString converts one exact decimal into its canonical persisted
// string form.
// Authored by: OpenCode
func (exactService) CanonicalString(value apd.Decimal) (string, error) {
	return CanonicalString(value)
}

// CanonicalStringPointer converts one optional exact decimal into its canonical
// persisted string form.
// Authored by: OpenCode
func (exactService) CanonicalStringPointer(value *apd.Decimal) (string, error) {
	return CanonicalStringPointer(value)
}

// ParseString parses one decimal string and returns both the canonical decimal
// value and its canonical persisted representation.
//
// Example:
//
//	decimalValue, canonical, err := decimal.ParseString("10.500")
//	if err != nil {
//		panic(err)
//	}
//	_, _ = decimalValue, canonical
//
// Authored by: OpenCode
func ParseString(raw string) (apd.Decimal, string, error) {
	var trimmed = strings.TrimSpace(raw)
	if trimmed == "" {
		return apd.Decimal{}, "", fmt.Errorf("decimal value is required")
	}

	var parsed apd.Decimal
	if _, _, err := parsed.SetString(trimmed); err != nil {
		return apd.Decimal{}, "", fmt.Errorf("parse decimal value: %w", err)
	}

	var canonicalValue, canonical, err = normalizeFiniteDecimal(&parsed)
	if err != nil {
		return apd.Decimal{}, "", err
	}

	return canonicalValue, canonical, nil
}

// ParseNumber parses one JSON number and returns both the canonical decimal
// value and its canonical persisted representation.
//
// Example:
//
//	decimalValue, canonical, err := decimal.ParseNumber(json.Number("10.500"))
//	if err != nil {
//		panic(err)
//	}
//	_, _ = decimalValue, canonical
//
// Authored by: OpenCode
func ParseNumber(raw json.Number) (apd.Decimal, string, error) {
	return ParseString(raw.String())
}

// CanonicalString converts one exact decimal into its canonical persisted
// string form.
//
// Example:
//
//	value, _, _ := decimal.ParseString("10.500")
//	canonical, err := decimal.CanonicalString(value)
//	if err != nil {
//		panic(err)
//	}
//	_ = canonical
//
// Authored by: OpenCode
func CanonicalString(value apd.Decimal) (string, error) {
	_, canonical, err := normalizeFiniteDecimal(&value)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

// CanonicalStringPointer converts one optional exact decimal into its canonical
// persisted string form.
//
// Example:
//
//	value, _, _ := decimal.ParseString("10.500")
//	canonical, err := decimal.CanonicalStringPointer(&value)
//	if err != nil {
//		panic(err)
//	}
//	_ = canonical
//
// Authored by: OpenCode
func CanonicalStringPointer(value *apd.Decimal) (string, error) {
	if value == nil {
		return "", nil
	}
	_, canonical, err := normalizeFiniteDecimal(value)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

// normalizeFiniteDecimal reduces one parsed decimal into its canonical finite
// persisted form.
// Authored by: OpenCode
func normalizeFiniteDecimal(value *apd.Decimal) (apd.Decimal, string, error) {
	if value == nil {
		return apd.Decimal{}, "", fmt.Errorf("decimal value is required")
	}
	if value.Form != apd.Finite {
		return apd.Decimal{}, "", fmt.Errorf("decimal value must be finite")
	}

	var reduced apd.Decimal
	reduced.Reduce(value)

	return reduced, reduced.Text('f'), nil
}
