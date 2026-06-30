// Package currency owns official exchange-rate provider integration for report
// base-currency conversion.
//
// This package is the boundary for provider HTTP clients, provider DTOs,
// provider selection, anticorruption mapping, canonical rate evidence, and the
// in-memory TUI-session rate cache. Code outside this package should request
// canonical evidence by source currency, report base currency, and activity date
// instead of handling provider payloads or provider URLs.
//
// Implementations in this package must use fixed official provider hosts, must
// not send Ghostfolio tokens to rate providers, must not persist exchange-rate
// evidence, and must keep financial and rate values as exact decimals.
//
// Authored by: OpenCode
package currency
