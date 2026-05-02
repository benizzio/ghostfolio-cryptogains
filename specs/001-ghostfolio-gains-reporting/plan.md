# Implementation Plan: Ghostfolio Gains Reporting

**Branch**: `[001-ghostfolio-gains-reporting]` | **Date**: 2026-05-02 | **Spec**: `/specs/001-ghostfolio-gains-reporting/spec.md`
**Input**: Feature specification from `/specs/001-ghostfolio-gains-reporting/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build an installed cross-platform Go terminal application that opens a short-lived authenticated Ghostfolio sync session for a specific registered local user, stores successful per-user setup and activity history only in local token-derived encrypted storage, and generates yearly PDF capital gains and losses reports from normalized activity history. The baseline architecture uses a Bubble Tea TUI, exact decimal domain math, stdlib HTTP integration against Ghostfolio's observed `api/v1` endpoints, and a deterministic report pipeline kept separate from storage, transport, and presentation concerns.

## Technical Context

**Language/Version**: Go 1.24  
**Primary Dependencies**: `github.com/charmbracelet/bubbletea/v2`, selected `github.com/charmbracelet/bubbles` components, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/signintech/gopdf`, Go standard library (`net/http`, `encoding/json`, `crypto/aes`, `crypto/cipher`, `os`, `path/filepath`)  
**Storage**: Local-only encrypted per-user snapshot files in the OS application data directory; Argon2id key derivation from the runtime Ghostfolio token; AES-256-GCM protected payload with an authenticated cleartext header; atomic rewrite on update  
**Testing**: `go test` with table-driven unit tests and `httptest.Server` integration suites; statement coverage from `go test -coverprofile`; branch and file coverage gate via `github.com/Fabianexe/gocoverageplus` until an in-repo verifier replaces it  
**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access and PDF file output  
**Project Type**: Single-module Go TUI application  
**Performance Goals**: Unlock cached data in under 2 seconds on supported hardware; complete sync normalization and persistence for 10,000 activities without freezing the UI; generate a yearly PDF report for 10,000 activities spanning 5 years in under 2 minutes  
**Constraints**: Ghostfolio token and JWT are runtime-only; no recoverable token trace on disk; HTTPS is required by default except explicitly allowed local-development origins; financial domain logic uses arbitrary-precision decimals only; persisted data stays local and is replaced atomically after confirmed server mismatch; no CGO-required runtime dependency in the baseline distribution  
**Scale/Scope**: Multiple encrypted local profiles per machine, each unlocked by its own Ghostfolio token; up to 10,000 stored activities per profile; one report type; five supported cost basis methods; Ghostfolio `api/v1` integration treated as runtime-validated rather than fully stable

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Persistence is justified only for encrypted setup and activity-cache reuse. Ghostfolio credentials remain runtime-only, per-user snapshots stay local-only and unlock solely from the user-entered token via Argon2id, and the OWASP Top 10:2025 review scope covers cryptographic failures, authentication failures, insecure transport/configuration, outdated components, logging leakage, and data-integrity tampering.
- [x] Precision: Domain math uses `apd/v3` arbitrary-precision decimals, JSON numbers are parsed without floating-point domain storage, canonical decimal strings preserve source scale at rest, and one documented output-rounding policy is applied only at the report boundary.
- [x] Testing: Integration-first tests drive setup, unlock, sync, normalization, mismatch replacement, and PDF generation via mocked Ghostfolio responses; unit tests isolate complex basis calculators, normalization rules, and crypto envelope code; statement and branch/file coverage are explicit release gates.
- [x] Dependencies: Every planned third-party library is justified against the standard library or a custom implementation and is researched in `research.md` for maintenance, community acceptance, security posture, release freshness, and recent activity.
- [x] External APIs: Ghostfolio integration is necessary, and the observed `api/v1` auth and activities endpoints, bearer-JWT model, pagination behavior, 400/401/403 failures, redaction risk, and host-origin security implications are documented.
- [x] Architecture: The design uses a single Go module with isolated domain, storage, Ghostfolio client, report rendering, and Bubble Tea presentation layers so tax logic remains independent from TUI and filesystem code.

## Project Structure

### Documentation (this feature)

```text
specs/001-ghostfolio-gains-reporting/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── ghostfolio-sync.md
│   └── tui-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── ghostfolio-cryptogains/
    └── main.go

internal/
├── app/
│   ├── bootstrap/
│   └── runtime/
├── tui/
│   ├── component/
│   ├── flow/
│   └── screen/
├── ghostfolio/
│   ├── client/
│   ├── dto/
│   └── mapper/
├── storage/
│   ├── envelope/
│   ├── profile/
│   └── snapshot/
├── report/
│   ├── builder/
│   └── pdf/
├── domain/
│   ├── activity/
│   ├── basis/
│   ├── ledger/
│   ├── report/
│   └── user/
└── support/
    ├── clock/
    ├── decimal/
    └── redact/

tests/
├── contract/
├── fixtures/
└── integration/
```

**Structure Decision**: Use a single Go module rooted at the repository root. Bubble Tea screens live under `internal/tui`, Ghostfolio HTTP and encrypted persistence remain in infrastructure packages, and all financial rules live under `internal/domain` so calculation logic is testable without filesystem or terminal dependencies.

## Complexity Tracking

No constitution violations require justification for this plan.
