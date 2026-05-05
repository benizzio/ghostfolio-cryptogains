# Implementation Plan: Ghostfolio Gains Reporting

**Branch**: `[001-ghostfolio-gains-reporting]` | **Date**: 2026-05-02 | **Spec**: `/specs/001-ghostfolio-gains-reporting/spec.md`
**Input**: Feature specification from `/specs/001-ghostfolio-gains-reporting/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build an installed cross-platform Go terminal application that defaults setup to the Ghostfolio cloud origin `https://ghostfol.io`, allows a self-hosted origin override, opens a short-lived authenticated Ghostfolio sync session for a specific registered local user, stores successful per-user setup and activity history only in local token-derived encrypted storage, and generates yearly PDF capital gains and losses reports from normalized activity history. The baseline architecture uses a Bubble Tea TUI, exact decimal domain math with no cross-currency conversion in this feature slice, stdlib HTTP integration against Ghostfolio's observed `api/v1` endpoints, support for only `BUY` and `SELL` source activities, zero-priced `SELL` handling as a non-taxable holding reduction, mandatory non-zero pricing for all `BUY` records, and source-scope normalization that derives `applicable_scope` from reliable wallet or account data when available, while keeping the report pipeline separate from storage, transport, and presentation concerns.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: `charm.land/bubbletea/v2`, selected `charm.land/bubbles/v2` components, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/signintech/gopdf`, Go standard library (`net/http`, `encoding/json`, `crypto/aes`, `crypto/cipher`, `os`, `path/filepath`)
**Storage**: Local-only encrypted per-user snapshot files in the OS application data directory; Argon2id key derivation from the runtime Ghostfolio token; AES-256-GCM protected payload with an authenticated cleartext header; atomic rewrite on update  
**Testing**: `go test` with table-driven unit tests and `httptest.Server` integration suites; statement coverage from `go test -coverprofile`; branch and file coverage gate via `github.com/Fabianexe/gocoverageplus` until an in-repo verifier replaces it  
**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access and PDF file output  
**Project Type**: Single-module Go TUI application  
**Performance Goals**: Unlock cached data in under 2 seconds on supported hardware; complete sync normalization and persistence for 10,000 activities without freezing the UI; generate a yearly PDF report for 10,000 activities spanning 5 years in under 2 minutes  
**Constraints**: Ghostfolio token and JWT are runtime-only; no recoverable token trace on disk; non-HTTPS production origins are rejected with a blocking error and only explicitly permitted local-development origins may use HTTP; financial domain logic uses arbitrary-precision decimals only and baseline calculations intentionally skip currency conversion by treating source base-currency amounts as price-equivalent; Ghostfolio source activity support is limited to `BUY` and `SELL`; normalized `BUY` records must have non-zero unit price; zero-priced `SELL` records are treated as non-taxable holding reductions and require explanatory comments; persisted data stays local and is replaced atomically after confirmed server mismatch; no CGO-required runtime dependency in the baseline distribution
**Scale/Scope**: Multiple encrypted local profiles per machine, each unlocked by its own Ghostfolio token; up to 10,000 stored activities per profile; one report type; five supported cost basis methods with the scope-local hybrid method narrowing to reliable source scope when available and broadening to asset-level scope when it is not; default sync target is the Ghostfolio cloud origin with optional self-hosted replacement; Ghostfolio `api/v1` integration treated as runtime-validated rather than fully stable

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Persistence is justified only for encrypted setup and activity-cache reuse. Ghostfolio credentials remain runtime-only, per-user snapshots stay local-only and unlock solely from the user-entered token via Argon2id, the storage design follows the OWASP Cryptographic Storage Cheat Sheet, non-HTTPS production origins are rejected with a blocking error, and the most recent published OWASP Top 10 review scope covers cryptographic failures, authentication failures, insecure transport/configuration, outdated components, logging leakage, and data-integrity tampering.
- [x] Precision: Domain math uses `apd/v3` arbitrary-precision decimals, JSON numbers are parsed without floating-point domain storage, canonical decimal strings preserve source scale at rest, baseline reporting intentionally performs no currency conversion and treats source base-currency amounts as price-equivalent inputs, and one documented output-rounding policy is applied only at the report boundary.
- [x] Testing: Integration-first tests drive setup, unlock, sync, normalization, mismatch replacement, and PDF generation via mocked Ghostfolio responses; unit tests isolate complex basis calculators, normalization rules, and crypto envelope code; statement and branch/file coverage are explicit release gates.
- [x] Dependencies: Every planned third-party library is justified against the standard library or a custom implementation and is researched in `research.md` for maintenance, community acceptance, security posture, release freshness, and recent activity.
- [x] External APIs: Ghostfolio integration is necessary, and the observed `api/v1` auth and activities endpoints, bearer-JWT model, pagination behavior, default cloud origin `https://ghostfol.io`, live cloud health/auth verification, account-scoped activity data used as source grouping data to derive `applicable_scope` when reliable, 400/401/403 failures, redaction risk, and host-origin security implications are documented.
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

## Source Activity Interpretation Rules

### Supported Source Activity Types

Only these normalized Ghostfolio activity types are valid input for this feature slice:

```text
BUY
SELL
```

Any other source activity type makes the ledger unsupported and sync must fail before persistence.

### BUY Rules

- A `BUY` record is always treated as an acquisition.
- A normalized `BUY` record must have `unit_price > 0`.
- If Ghostfolio models the receiving side of a blockchain transfer as a `BUY`, that record must already contain the intended non-zero acquisition price used for basis.
- Free-text comments on `BUY` records are explanatory only and must not be used to infer basis linkage to any other activity.

### SELL Rules

- A `SELL` record with `unit_price > 0` is treated as a disposal under the selected cost basis method.
- A `SELL` record with `unit_price = 0` is treated as a non-taxable holding reduction representing a blockchain fee or transfer-out movement.
- A zero-priced `SELL` must include an explanatory comment in the normalized record.
- Free-text comments on `SELL` records are explanatory only and must not be used to infer basis linkage to any receiving acquisition.

## Cost Basis Calculation Rules

### Shared Cost Basis Math

These formulas apply to all methods unless a method definition narrows them.

Acquisition:

```text
acquisition_basis = gross_value + acquisition_fee
unit_cost = acquisition_basis / acquired_quantity
```

Disposal:

```text
net_proceeds = gross_value - disposal_fee
gain_or_loss = net_proceeds - allocated_basis
```

If one disposal is matched across multiple fragments, proceeds are allocated pro rata by matched quantity:

```text
proceeds_per_unit = net_proceeds / disposed_quantity
matched_proceeds_i = proceeds_per_unit * matched_quantity_i
matched_gain_or_loss_i = matched_proceeds_i - matched_basis_i
```

### Deterministic Ordering Rules

- Normalize same-asset history in this stable order before any basis calculation:

```text
occurred_at asc
then source_id asc
```

- If same-asset events still cannot be deterministically ordered after applying that rule, the ledger is non-defensible and sync must fail.
- For method definitions below, `acquired_at` means the acquisition event timestamp from normalized history.

### FIFO

Maintain open lots per asset. On each disposal, consume the oldest open lots first.

Lot matching order:

```text
acquired_at asc
then source_id asc
```

Partial lot consumption:

```text
lot_unit_cost = lot_basis / lot_quantity
matched_basis = lot_unit_cost * matched_quantity
remaining_quantity' = remaining_quantity - matched_quantity
remaining_basis' = remaining_basis - matched_basis
```

### LIFO

Maintain open lots per asset. On each disposal, consume the newest open lots first.

Lot matching order:

```text
acquired_at desc
then source_id desc
```

Use the same partial-lot formulas as FIFO.

### HIFO

Maintain open lots per asset. On each disposal, consume the open lots with the highest unit cost first.

Lot matching order:

```text
unit_cost desc
then acquired_at asc
then source_id asc
```

Use the same partial-lot formulas as FIFO.

### Average Cost Basis

Maintain one moving weighted-average pool per asset using all activity for that asset up to each disposal, regardless of scope.

Pool state:

```text
pool_quantity
pool_basis
average_unit_cost = pool_basis / pool_quantity
```

On acquisition:

```text
pool_quantity' = pool_quantity + acquired_quantity
pool_basis' = pool_basis + acquisition_basis
average_unit_cost' = pool_basis' / pool_quantity'
```

On disposal:

```text
allocated_basis = disposed_quantity * average_unit_cost
pool_quantity' = pool_quantity - disposed_quantity
pool_basis' = pool_basis - allocated_basis
gain_or_loss = net_proceeds - allocated_basis
```

Clarifications:

- Average Cost Basis is global per asset, not per wallet or account.
- A disposal uses the average unit cost immediately before that disposal.
- A disposal reduces pool quantity and pool basis proportionally.
- If `pool_quantity` becomes zero, the next acquisition starts a new pool.

### Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order

This is a hybrid method. It does not degrade to FIFO.

Applicable scope:

```text
applicable_scope = reliable wallet or account scope for the asset when that scope data is available and defensible
otherwise applicable_scope = the asset as a whole
```

Exact unit identification is possible only when the normalized ledger can unambiguously identify the outgoing units within the current `(asset, applicable_scope)` partition.

Valid sources of exact identification are limited to:

- explicit disposal-to-acquisition linkage in normalized data
- other normalized source evidence that identifies the exact outgoing units without ambiguity

For each disposal:

1. Determine `applicable_scope`.
2. Partition holdings by `(asset, applicable_scope)`.
3. If the outgoing units are exactly identifiable within that partition, allocate basis from those exact units.
4. Otherwise, use the partition-local average-cost fallback and assign deemed-disposal order to the oldest remaining quantities in that same partition.

Exact identification basis:

```text
allocated_basis = sum(matched_quantity_i * matched_unit_cost_i)
matched_unit_cost_i = matched_basis_i / matched_quantity_i
```

Fallback average-cost basis within the current partition:

```text
partition_quantity = sum(remaining_quantity_i)
partition_basis = sum(remaining_basis_i)
average_unit_cost = partition_basis / partition_quantity
allocated_basis = disposed_quantity * average_unit_cost
partition_quantity' = partition_quantity - disposed_quantity
partition_basis' = partition_basis - allocated_basis
gain_or_loss = net_proceeds - allocated_basis
```

Fallback deemed-disposal order within the same partition:

```text
occurred_at asc
then source_id asc
```

### Hybrid Runtime Modeling Rule

Within each `(asset, applicable_scope)` partition, maintain two independent runtime structures while the hybrid method is active:

1. A valuation state used for basis math.
2. A provenance state used for exact identification or oldest-acquired deemed-disposal ordering.

Recommended minimum model:

```text
valuation_pool_quantity
valuation_pool_basis
provenance_queue[] = {quantity, occurred_at, source_id, carried_provenance}
exact_lots[] = {quantity, basis, occurred_at, source_id, carried_provenance}
pooled_until_zero = boolean
```

Normative behavior:

- While exact-lot basis remains defensible and `pooled_until_zero == false`, exact unit matching may be used when exact outgoing units are known.
- If the method falls back to partition-local average cost for any disposal in an open partition, set `pooled_until_zero = true` for that partition.
- While `pooled_until_zero == true`, all further disposals in that partition must use the partition-local average-cost valuation until the partition quantity reaches zero.
- During `pooled_until_zero == true`, provenance removal still follows oldest-acquired deemed-disposal order.
- When partition quantity reaches zero, clear all partition state. Later acquisitions start a new clean partition state.

### Zero-Priced SELL Holding-Reduction Rule

A `SELL` record with unit price `0` is not a capital gain or loss realization event. It still reduces holdings under the currently selected cost basis method.

Method-specific basis removal:

```text
allocated_basis = method_specific_basis_for_removed_quantity
remaining_quantity' = remaining_quantity - reduced_quantity
remaining_basis' = remaining_basis - allocated_basis
realized_gain_or_loss = 0
```

Normative behavior:

- The quantity reduction follows the same ordering, partitioning, and basis-allocation rules that the active cost basis method would use for a taxable disposal.
- The removed basis leaves the remaining holdings.
- The event does not create taxable proceeds.
- The event does not create a gain or loss line item in the report output.

### Priced BUY Acquisition Rule

A `BUY` record always affects cost basis directly using its explicit non-zero unit price.

```text
acquisition_basis = gross_value + acquisition_fee
unit_cost = acquisition_basis / acquired_quantity
```

Normative behavior:

- A transfer-in or similar receiving movement must arrive as a priced `BUY` if it is intended to adjust holdings and basis.
- The acquisition basis for that `BUY` comes from the priced activity data itself.
- Comments may explain why the acquisition occurred, but comments do not participate in basis linkage or matching.

### Invalid History Conditions

Reject the normalized ledger as non-defensible if any of the following occurs:

- any normalized activity type is not `BUY` or `SELL`
- a `BUY` record has unit price `0`
- a zero-priced `SELL` lacks an explanatory comment
- disposed quantity exceeds currently available quantity in the relevant partition
- required ordering is ambiguous after deterministic tie-breaking
- exact unit identification is claimed but cannot be substantiated by normalized data

## Complexity Tracking

No constitution violations require justification for this plan.
