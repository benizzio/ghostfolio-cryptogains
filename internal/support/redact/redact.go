// Package redact provides secret-safe text helpers for application-owned
// diagnostics and transient messages.
// Authored by: OpenCode
package redact

import (
	"regexp"
	"strings"
)

const replacement = "[REDACTED]"

var obviousSecretPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{pattern: regexp.MustCompile(`(?i)\b(token|jwt|payload)\s*[:=]\s*[^\s]+`), replacement: `$1=[REDACTED]`},
	{pattern: regexp.MustCompile(`(?i)\b(token|jwt|payload)\s+[^\s]+`), replacement: `$1 [REDACTED]`},
	{pattern: regexp.MustCompile(`(?i)\bBearer\s+[^\s]+`), replacement: `Bearer [REDACTED]`},
}

// Text removes known secret values from text that may be shown to users or
// written into diagnostics.
//
// Example:
//
//	safe := redact.Text("token=abc123", "abc123")
//	_ = safe
//
// Authored by: OpenCode
func Text(input string, secrets ...string) string {
	var redacted = input
	for _, secret := range secrets {
		if strings.TrimSpace(secret) == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, replacement)
	}
	for _, secretPattern := range obviousSecretPatterns {
		redacted = secretPattern.pattern.ReplaceAllString(redacted, secretPattern.replacement)
	}
	return redacted
}

// ErrorText converts an error into a redacted string representation.
//
// Example:
//
//	safe := redact.ErrorText(err, "abc123")
//	_ = safe
//
// Authored by: OpenCode
func ErrorText(err error, secrets ...string) string {
	if err == nil {
		return ""
	}
	return Text(err.Error(), secrets...)
}
