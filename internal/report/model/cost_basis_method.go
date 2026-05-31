// Package model defines runtime report models shared across calculation,
// rendering, output, and runtime orchestration packages.
// Authored by: OpenCode
package model

import "strings"

// CostBasisMethod identifies one supported cost-basis method for a full report
// run.
// Authored by: OpenCode
type CostBasisMethod string

const (
	// CostBasisMethodFIFO consumes the oldest open acquisitions first.
	CostBasisMethodFIFO CostBasisMethod = "fifo"

	// CostBasisMethodLIFO consumes the newest open acquisitions first.
	CostBasisMethodLIFO CostBasisMethod = "lifo"

	// CostBasisMethodHIFO consumes the highest-unit-cost open acquisitions first.
	CostBasisMethodHIFO CostBasisMethod = "hifo"

	// CostBasisMethodAverageCost maintains one moving weighted-average pool per
	// asset or applicable scope.
	CostBasisMethodAverageCost CostBasisMethod = "average_cost"

	// CostBasisMethodScopeLocalHybrid applies scope-local exact matching when it
	// remains defensible and otherwise falls back to scope-local average cost.
	CostBasisMethodScopeLocalHybrid CostBasisMethod = "scope_local_hybrid"
)

var supportedCostBasisMethods = []CostBasisMethod{
	CostBasisMethodFIFO,
	CostBasisMethodLIFO,
	CostBasisMethodHIFO,
	CostBasisMethodAverageCost,
	CostBasisMethodScopeLocalHybrid,
}

// SupportedCostBasisMethods returns the exact supported report cost-basis
// methods in the stable selection order used across the application.
//
// Example:
//
//	methods := model.SupportedCostBasisMethods()
//	_ = methods[0]
//
// Authored by: OpenCode
func SupportedCostBasisMethods() []CostBasisMethod {
	var methods = make([]CostBasisMethod, len(supportedCostBasisMethods))
	copy(methods, supportedCostBasisMethods)
	return methods
}

// Label returns the exact user-visible label for one supported report
// cost-basis method.
//
// Example:
//
//	label := model.CostBasisMethodFIFO.Label()
//	_ = label
//
// Authored by: OpenCode
func (method CostBasisMethod) Label() string {
	switch method {
	case CostBasisMethodFIFO:
		return "FIFO"
	case CostBasisMethodLIFO:
		return "LIFO"
	case CostBasisMethodHIFO:
		return "HIFO"
	case CostBasisMethodAverageCost:
		return "Average Cost Basis"
	case CostBasisMethodScopeLocalHybrid:
		return "Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order"
	default:
		return strings.TrimSpace(string(method))
	}
}

// FilenameSlug returns the stable lowercase filename slug for one report
// cost-basis method.
//
// Example:
//
//	slug := model.CostBasisMethodAverageCost.FilenameSlug()
//	_ = slug
//
// Authored by: OpenCode
func (method CostBasisMethod) FilenameSlug() string {
	switch method {
	case CostBasisMethodFIFO:
		return "fifo"
	case CostBasisMethodLIFO:
		return "lifo"
	case CostBasisMethodHIFO:
		return "hifo"
	case CostBasisMethodAverageCost:
		return "average-cost"
	case CostBasisMethodScopeLocalHybrid:
		return "scope-local-hybrid"
	default:
		return sanitizeCostBasisMethodSlug(strings.TrimSpace(string(method)))
	}
}

// Explanation returns the plain-language explanation shown while the user
// highlights one report cost-basis method in the TUI.
//
// Example:
//
//	explanation := model.CostBasisMethodHIFO.Explanation()
//	_ = explanation
//
// Authored by: OpenCode
func (method CostBasisMethod) Explanation() string {
	switch method {
	case CostBasisMethodFIFO:
		return "FIFO uses the oldest open acquisitions first."
	case CostBasisMethodLIFO:
		return "LIFO uses the newest open acquisitions first."
	case CostBasisMethodHIFO:
		return "HIFO uses the highest-unit-cost open acquisitions first."
	case CostBasisMethodAverageCost:
		return "Average Cost Basis uses one moving weighted-average pool."
	case CostBasisMethodScopeLocalHybrid:
		return "Scope-local exact matching stays narrow when defensible and otherwise falls back to scope-local average cost until the scope reaches zero."
	default:
		return "Select one supported method."
	}
}

// sanitizeCostBasisMethodSlug normalizes one unsupported raw method value into a
// filename-safe fallback slug.
// Authored by: OpenCode
func sanitizeCostBasisMethodSlug(raw string) string {
	if raw == "" {
		return "method"
	}

	var builder strings.Builder
	builder.Grow(len(raw))
	var lastDash = false

	for _, current := range strings.ToLower(raw) {
		if current >= 'a' && current <= 'z' || current >= '0' && current <= '9' {
			builder.WriteRune(current)
			lastDash = false
			continue
		}
		if lastDash {
			continue
		}
		builder.WriteByte('-')
		lastDash = true
	}

	var slug = strings.Trim(builder.String(), "-")
	if slug == "" {
		return "method"
	}

	return slug
}
