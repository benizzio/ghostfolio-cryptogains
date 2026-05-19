// Package model defines the protected snapshot data structures shared across
// snapshot packages.
// Authored by: OpenCode
package model

// SupportsEnvelopeFormatVersion reports whether the supplied cleartext envelope
// format version is supported by this application build.
// Authored by: OpenCode
func SupportsEnvelopeFormatVersion(formatVersion int) bool {
	return formatVersion == EnvelopeFormatVersion
}

// SupportsStoredDataVersion reports whether the supplied protected stored-data
// version markers are supported by this application build.
// Authored by: OpenCode
func SupportsStoredDataVersion(version StoredDataVersion) bool {
	return version.EnvelopeFormatVersion == EnvelopeFormatVersion &&
		version.PayloadSchemaVersion == PayloadSchemaVersion &&
		version.ActivityModelVersion == ActivityModelVersion
}
