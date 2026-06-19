// Package fixture provides deterministic synthetic-content validation helpers for persisted empirical fixtures.
//
// Authored by: OpenCode
package fixture

import (
	"fmt"
	"regexp"
	"strings"

	supporttext "github.com/benizzio/ghostfolio-cryptogains/internal/support/text"
)

const (
	violationBearerToken     = "bearer_token"
	violationCopiedFixture   = "copied_fixture_text"
	violationJWTLikeValue    = "jwt_like_value"
	violationRealNameLike    = "real_name_like_value"
	violationTokenLikeValue  = "token_like_value"
	defaultSyntheticLocation = "fixture content"
)

var bearerTokenPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{12,}\b`)
var copiedFixtureHeaderPattern = regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}\s+(?:[*!]\s+)?(?:"[^"]+"|[^:]+)$`)
var copiedFixturePostingPattern = regexp.MustCompile(`^(?:Assets|Liabilities|Equity|Income|Expenses):[A-Za-z0-9:_-]+(?:\s|$)`)
var beancountDirectivePattern = regexp.MustCompile(`^(?:option|plugin|include|open|close|balance|commodity|pushtag|poptag)\s+`)
var jwtLikePattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}\b`)
var opaqueTokenValuePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~+/=-]{11,}$`)
var opaqueTokenContentPattern = regexp.MustCompile(`\b[A-Za-z0-9][A-Za-z0-9._~+/=]{15,}\b`)
var realNamePattern = regexp.MustCompile(`^[A-Z][a-z]+(?:['-][A-Z][a-z]+)?(?:\s+[A-Z][a-z]+(?:['-][A-Z][a-z]+)?){1,2}$`)
var realNameContentPattern = regexp.MustCompile(`\b[A-Z][a-z]+(?:['-][A-Z][a-z]+)?(?:\s+[A-Z][a-z]+(?:['-][A-Z][a-z]+)?){1,2}\b`)

var realNameFields = map[string]struct{}{
	"account_name": {},
	"display_name": {},
	"full_name":    {},
	"holder_name":  {},
	"name":         {},
	"owner_name":   {},
	"user_name":    {},
	"wallet_name":  {},
}

var syntheticMarkers = []string{
	"account",
	"dataset",
	"demo",
	"dummy",
	"example",
	"fixture",
	"masked",
	"mock",
	"oracle",
	"placeholder",
	"portfolio",
	"redacted",
	"sample",
	"scope",
	"synthetic",
	"test",
	"user",
	"wallet",
}

var tokenFields = map[string]struct{}{
	"access_token":  {},
	"api_token":     {},
	"auth_token":    {},
	"bearer_token":  {},
	"id_token":      {},
	"refresh_token": {},
	"token":         {},
}

var genericFreeTextFields = map[string]struct{}{
	"comment":     {},
	"description": {},
	"details":     {},
	"label":       {},
	"memo":        {},
	"message":     {},
	"name":        {},
	"note":        {},
	"reason":      {},
	"summary":     {},
	"text":        {},
	"title":       {},
}

// SyntheticContentIssue describes one synthetic-only validation failure without exposing the matched text.
//
// Authored by: OpenCode
type SyntheticContentIssue struct {
	Field    string
	Kind     string
	Line     int
	Location string
	Message  string
}

// Error formats the issue for validation output without echoing the matched secret-like or copied content.
//
// Example:
//
//	message := issue.Error()
//	if message != "" {
//		fmt.Println(message)
//	}
//
// Authored by: OpenCode
func (issue SyntheticContentIssue) Error() string {
	if issue.Field != "" {
		return fmt.Sprintf("%s:%d field %s: %s: %s", issue.Location, issue.Line, issue.Field, issue.Kind, issue.Message)
	}

	return fmt.Sprintf("%s:%d: %s: %s", issue.Location, issue.Line, issue.Kind, issue.Message)
}

// SyntheticContentError groups deterministic synthetic-only validation failures for one persisted fixture input.
//
// Authored by: OpenCode
type SyntheticContentError struct {
	Issues   []SyntheticContentIssue
	Location string
}

// Error formats the grouped validation failures for callers that need one actionable error value.
//
// Example:
//
//	err := fixture.SyntheticContentError{Location: path, Issues: issues}
//	if err.Error() != "" {
//		fmt.Println(err.Error())
//	}
//
// Authored by: OpenCode
func (issueError SyntheticContentError) Error() string {
	var builder strings.Builder
	var location = normalizeLocation(issueError.Location)

	if len(issueError.Issues) == 0 {
		return fmt.Sprintf("%s failed synthetic-only content validation", location)
	}

	builder.WriteString(fmt.Sprintf("%s failed synthetic-only content validation with %d issue(s):", location, len(issueError.Issues)))

	for _, issue := range issueError.Issues {
		builder.WriteString("\n- ")
		builder.WriteString(issue.Error())
	}

	return builder.String()
}

// ScanSyntheticOnlyContent scans persisted fixture text for deterministic synthetic-only validation failures.
//
// Example:
//
//	issues := fixture.ScanSyntheticOnlyContent("testdata/empirical/financial-dataset.yaml", content)
//	if len(issues) != 0 {
//		return fixture.SyntheticContentError{Location: "testdata/empirical/financial-dataset.yaml", Issues: issues}
//	}
//
// Authored by: OpenCode
func ScanSyntheticOnlyContent(location string, content string) []SyntheticContentIssue {
	var issues []SyntheticContentIssue
	var normalizedLocation = normalizeLocation(location)
	var lines = strings.Split(content, "\n")

	for index, line := range lines {
		var lineNumber = index + 1
		var trimmedLine = strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		var fieldName, fieldValue, hasField = parseStructuredField(trimmedLine)

		if looksLikeBearerToken(trimmedLine) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, fieldName, violationBearerToken, "remove bearer token syntax and replace it with reviewed synthetic explanatory text"))
		}

		if looksLikeJWT(trimmedLine) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, fieldName, violationJWTLikeValue, "remove the JWT-like value or replace it with reviewed synthetic placeholder text"))
		}

		if hasField && looksLikeTokenField(fieldName, fieldValue) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, fieldName, violationTokenLikeValue, "remove the token-like field value or replace it with reviewed synthetic placeholder text"))
		}
		if hasField && looksLikeGenericFreeTextToken(fieldName, fieldValue) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, fieldName, violationTokenLikeValue, "remove the token-like free-text content or replace it with reviewed synthetic placeholder text"))
		}

		if hasField && looksLikeRealNameField(fieldName, fieldValue) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, fieldName, violationRealNameLike, "use a clearly synthetic label instead of a real-person name"))
		}
		if hasField && looksLikeGenericFreeTextRealName(fieldName, fieldValue) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, fieldName, violationRealNameLike, "remove the real-person name from free-text content or replace it with a clearly synthetic placeholder"))
		}

		if looksLikeCopiedFixtureLine(trimmedLine) {
			issues = append(issues, newSyntheticContentIssue(normalizedLocation, lineNumber, "", violationCopiedFixture, "rewrite copied ledger-style text using the project-owned dataset or oracle schema"))
		}
	}

	return issues
}

// ValidateSyntheticOnlyContent converts synthetic-only scan results into one actionable error for callers.
//
// Example:
//
//	err := fixture.ValidateSyntheticOnlyContent("testdata/empirical/golden/fifo.json", content)
//	if err != nil {
//		return err
//	}
//
// Authored by: OpenCode
func ValidateSyntheticOnlyContent(location string, content string) error {
	var issues = ScanSyntheticOnlyContent(location, content)

	if len(issues) == 0 {
		return nil
	}

	return SyntheticContentError{Location: normalizeLocation(location), Issues: issues}
}

// newSyntheticContentIssue builds one deterministic issue result.
//
// Authored by: OpenCode
func newSyntheticContentIssue(location string, line int, field string, kind string, message string) SyntheticContentIssue {
	return SyntheticContentIssue{
		Field:    field,
		Kind:     kind,
		Line:     line,
		Location: location,
		Message:  message,
	}
}

// normalizeLocation gives empty caller locations a stable label.
//
// Authored by: OpenCode
func normalizeLocation(location string) string {
	var trimmedLocation = strings.TrimSpace(location)

	if trimmedLocation == "" {
		return defaultSyntheticLocation
	}

	return trimmedLocation
}

// parseStructuredField extracts a simple YAML or JSON-style field name and value from one line.
//
// Authored by: OpenCode
func parseStructuredField(line string) (string, string, bool) {
	var field string
	var ok bool
	var trimmedLine = strings.TrimSpace(line)

	trimmedLine = strings.TrimPrefix(trimmedLine, "- ")
	field, trimmedLine, ok = strings.Cut(trimmedLine, ":")

	if !ok {
		field, trimmedLine, ok = strings.Cut(trimmedLine, "=")
	}

	if !ok {
		return "", "", false
	}

	var normalizedField = normalizeFieldName(field)
	var normalizedValue = normalizeFieldValue(trimmedLine)

	if normalizedField == "" || normalizedValue == "" {
		return "", "", false
	}

	return normalizedField, normalizedValue, true
}

// normalizeFieldName canonicalizes a field name for rule matching.
//
// Authored by: OpenCode
func normalizeFieldName(field string) string {
	var normalizedField = strings.TrimSpace(field)

	normalizedField = strings.Trim(normalizedField, `"'`)
	normalizedField = strings.ToLower(normalizedField)
	normalizedField = strings.ReplaceAll(normalizedField, "-", "_")
	normalizedField = strings.ReplaceAll(normalizedField, " ", "_")

	return normalizedField
}

// normalizeFieldValue trims common quoting and trailing punctuation from a field value.
//
// Authored by: OpenCode
func normalizeFieldValue(value string) string {
	var normalizedValue = strings.TrimSpace(value)

	normalizedValue = strings.TrimSuffix(normalizedValue, ",")
	normalizedValue = strings.TrimSpace(normalizedValue)
	normalizedValue = strings.Trim(normalizedValue, `"'`)

	return strings.TrimSpace(normalizedValue)
}

// looksLikeBearerToken detects bearer-style authorization syntax.
//
// Authored by: OpenCode
func looksLikeBearerToken(line string) bool {
	return bearerTokenPattern.MatchString(line)
}

// looksLikeCopiedFixtureLine detects upstream ledger or beancount fixture syntax pasted into persisted fixtures.
//
// Authored by: OpenCode
func looksLikeCopiedFixtureLine(line string) bool {
	return copiedFixtureHeaderPattern.MatchString(line) || copiedFixturePostingPattern.MatchString(line) || beancountDirectivePattern.MatchString(line)
}

// looksLikeJWT detects JWT-shaped values without exposing them.
//
// Authored by: OpenCode
func looksLikeJWT(line string) bool {
	return jwtLikePattern.MatchString(line)
}

// looksLikeRealNameField detects real-person names in name-like fields while allowing clearly synthetic placeholders.
//
// Authored by: OpenCode
func looksLikeRealNameField(field string, value string) bool {
	if _, ok := realNameFields[field]; !ok {
		return false
	}

	if containsSyntheticMarker(value) {
		return false
	}

	return realNamePattern.MatchString(value)
}

// looksLikeTokenField detects token-labelled fields with opaque token-like values.
//
// Authored by: OpenCode
func looksLikeTokenField(field string, value string) bool {
	if _, ok := tokenFields[field]; !ok {
		return false
	}

	if containsSyntheticMarker(value) {
		return false
	}

	return opaqueTokenValuePattern.MatchString(value)
}

// looksLikeGenericFreeTextToken detects opaque token-like content embedded in generic prose fields.
//
// Authored by: OpenCode
func looksLikeGenericFreeTextToken(field string, value string) bool {
	if _, ok := tokenFields[field]; ok {
		return false
	}
	if !isGenericFreeTextField(field) {
		return false
	}

	var matches = opaqueTokenContentPattern.FindAllString(value, -1)
	var match string
	for _, match = range matches {
		if containsSyntheticMarker(match) {
			continue
		}
		if supporttext.ContainsASCIILetter(match) && supporttext.ContainsASCIIDigit(match) {
			return true
		}
	}

	return false
}

// looksLikeGenericFreeTextRealName detects real-person names embedded in generic prose fields.
//
// Authored by: OpenCode
func looksLikeGenericFreeTextRealName(field string, value string) bool {
	if _, ok := realNameFields[field]; ok {
		return false
	}
	if !isGenericFreeTextField(field) {
		return false
	}

	var matches = realNameContentPattern.FindAllString(value, -1)
	var match string
	for _, match = range matches {
		if containsSyntheticMarker(match) {
			continue
		}

		return true
	}

	return false
}

// isGenericFreeTextField reports whether a field conventionally stores free-form prose.
//
// Authored by: OpenCode
func isGenericFreeTextField(field string) bool {
	if _, ok := genericFreeTextFields[field]; ok {
		return true
	}

	return strings.HasSuffix(field, "_description") ||
		strings.HasSuffix(field, "_details") ||
		strings.HasSuffix(field, "_label") ||
		strings.HasSuffix(field, "_message") ||
		strings.HasSuffix(field, "_note") ||
		strings.HasSuffix(field, "_reason") ||
		strings.HasSuffix(field, "_summary") ||
		strings.HasSuffix(field, "_text") ||
		strings.HasSuffix(field, "_title")
}

// containsSyntheticMarker allows clearly synthetic placeholder text to pass name and token heuristics.
//
// Authored by: OpenCode
func containsSyntheticMarker(value string) bool {
	var lowerValue = strings.ToLower(value)

	for _, marker := range syntheticMarkers {
		if strings.Contains(lowerValue, marker) {
			return true
		}
	}

	return false
}
