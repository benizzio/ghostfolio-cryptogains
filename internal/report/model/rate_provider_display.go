// Package model defines display labels for report-owned rate provider evidence.
// Authored by: OpenCode
package model

// RateAuthorityDisplayLabel returns the report-facing label for a retained rate
// authority value.
// Authored by: OpenCode
func RateAuthorityDisplayLabel(authority RateAuthority) string {
	switch authority {
	case RateAuthorityEuropeanCentralBank:
		return "European Central Bank"
	case RateAuthorityFederalReserve:
		return "Federal Reserve"
	default:
		return string(authority)
	}
}

// RateProviderDisplayLabel returns the report-facing label for retained provider
// evidence.
// Authored by: OpenCode
func RateProviderDisplayLabel(providerID RateProviderID) string {
	switch providerID {
	case RateProviderIDECBEXR:
		return "ECB Data Portal EXR"
	case RateProviderIDFederalReserveH10:
		return "Federal Reserve Board H.10/Data Download Program"
	default:
		return string(providerID)
	}
}

// RateProviderUnavailableDateRule returns the report-facing prior-observation
// disclosure for retained provider evidence.
// Authored by: OpenCode
func RateProviderUnavailableDateRule(providerID RateProviderID) string {
	switch providerID {
	case RateProviderIDECBEXR:
		return "most recent previous available ECB observation"
	case RateProviderIDFederalReserveH10:
		return "most recent previous available H.10 observation"
	default:
		return "most recent previous available official observation"
	}
}
