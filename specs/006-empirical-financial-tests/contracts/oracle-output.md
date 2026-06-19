# Contract: External Oracle Output

**Bugfix**: 2026-06-10 — [BUG-001] Superseded the hledger-only output contract with rotki-backed pure-method fixtures and Scope-Local Hybrid composite-oracle fixture rules.

## Scope

This contract defines generated external-oracle input files, normalized oracle golden fixtures, fixture metadata, unsupported-case handling, rotki-backed pure-method oracle expectations, and Scope-Local Hybrid composite-oracle expectations.

## Locations

Generated external-oracle inputs:

```text
.cache/empiricaloracle/oracle-inputs/
```

Normalized golden fixtures:

```text
testdata/empirical/golden/
```

External oracle materials:

```text
third_party/rotki/
```

Oracle helper code:

```text
tools/empiricaloracle/
```

## Oracle Fixture Shape

```json
{
  "fixture_version": "1",
  "dataset_version": "1",
  "case_id": "case-fifo-alpha-2024",
  "method": "fifo",
  "year": 2024,
  "asset_identity_key": "asset-alpha",
  "values": {
    "realized_gain_or_loss": "19.6666666666666667",
    "allocated_basis": "17.3333333333333333",
    "closing_quantity": "8",
    "closing_basis": "94"
  },
  "matches": [
    {
      "disposed_source_id": "emp-act-000053",
      "acquisition_source_id": "emp-act-000004",
      "matched_quantity": "1",
      "matched_basis": "3.3333333333333333",
      "matched_proceeds": "21",
      "matched_gain_or_loss": "17.6666666666666667",
      "support_label": "rotki_backed"
    }
  ],
  "unsupported_segments": [],
  "metadata": {
    "oracle_name": "rotki",
    "source_url": "https://github.com/rotki/rotki/archive/refs/tags/v1.43.1.tar.gz",
    "source_checksum": "sha256:8434b653104f8d5b0638e98d88a5ef256fac7720cc459eb33b729e2848900e3b",
    "version_or_commit": "a2e00be49a0ea36e7563a5d235cfa6a7c91edbfb",
    "adapter_arguments": [
      "--source-root",
      ".cache/empiricaloracle/rotki-source/rotki-1.43.1",
      "--input",
      ".cache/empiricaloracle/oracle-inputs/fifo/case-fifo-alpha-2024.json",
      "--rotki-method",
      "fifo",
      "--method",
      "fifo"
    ],
    "adapter_constraints": [
      "Verified pinned rotki source archive execution from an untracked project-local cache",
      "Zero-priced holding reductions are excluded from external-oracle fixture generation"
    ],
    "dataset_input_hash": "sha256:6287f862061b1f51d795694ab768b1927ff4422cc94b10a7ef053b45ef581042",
    "external_oracle_input_hash": "sha256:b53828dfc811cbb906acf379084a14b5ce30862453970205ed19c517692dd40f",
    "decimal_policy": "scale=16,rounding=half_up",
    "normalization_version": "1",
    "financial_tolerances": {
      "realized_gain_or_loss": "0",
      "allocated_basis": "0",
      "closing_basis": "0"
    },
    "tolerance_notes": {},
    "oracle_output_hash": "sha256:dc2c394c49a645bcefbf2770e1ae874bcaa7c1910089eb25b39dc0d833a14c51"
  }
}
```

Scope-Local Hybrid composite fixtures use the same top-level shape with:

```json
{
  "metadata": {
    "oracle_name": "scope_local_hybrid_composite",
    "adapter_arguments": [
      "--source-root",
      ".cache/empiricaloracle/rotki-source/rotki-1.43.1",
      "--input",
      ".cache/empiricaloracle/oracle-inputs/scope-local-hybrid/case-scope-local-reliable-epsilon-2024.json",
      "--rotki-method",
      "average_cost",
      "--method",
      "scope_local_hybrid"
    ],
    "composite_rule_version": "scope_local_hybrid_composite_v1"
  }
}
```

## Current Committed Fixture Set

- `fifo/case-fifo-alpha-2024.json`
- `lifo/case-lifo-beta-2024.json`
- `hifo/case-hifo-gamma-2024.json`
- `average-cost/case-average-cost-delta-2024.json`
- `average-cost/case-average-cost-reset-delta-2024.json`
- `average-cost/case-post-year-ignore-delta-2024.json`
- `scope-local-hybrid/case-scope-local-broadening-gamma-2024--asset-delta.json`
- `scope-local-hybrid/case-scope-local-broadening-gamma-2024--asset-gamma.json`
- `scope-local-hybrid/case-scope-local-reliable-epsilon-2024.json`
- `scope-local-hybrid/case-scope-local-reset-epsilon-2024.json`

## Required Metadata

Every golden fixture must include:

- external oracle name
- external oracle source URL
- external oracle source checksum
- pinned version or commit
- exact adapter or command arguments
- adapter constraints
- selected decimal policy
- documented financial tolerances
- tolerance notes for every non-zero financial tolerance
- dataset input hash
- generated external-oracle input hash
- normalized oracle output hash
- normalization version
- dataset version
- fixture version
- method
- year
- case ID
- asset identity key

## Decimal Rules

- All decimal values are JSON strings.
- Quantities are canonical decimal strings and compare exactly.
- Financial values are normalized to the selected decimal policy before comparison.
- The default selected policy is the project's production 16-decimal round-half-up policy.
- Accepted `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` values use the form `scale=<digits>,rounding=half_up`.
- The required accepted value is `scale=16,rounding=half_up`, matching the production default.
- `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` is an application-run-scoped override, not a test-only switch.
- Additional external-oracle-aligned accepted values may be added only when the selected external oracle cannot align with the production default, and each added value must be documented with the oracle name, pinned version or commit, and reason. Practical custom scale values should stay at or below 64 for safety.
- If the selected external oracle cannot be configured or normalized to the production policy for every valid case, the relevant empirical run must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` before project calculation runs and fixtures must record the external-oracle-aligned policy used.
- Residual financial differences after decimal-policy alignment may use documented per-field tolerances. Quantity tolerance is always zero.
- Non-zero financial tolerances must not exceed one unit at the selected decimal-policy scale. For the production 16-decimal policy, the maximum is `0.0000000000000001`.
- Every non-zero financial tolerance must include a tolerance note explaining why exact equality is not achievable for that external-oracle-derived value.
- Floating-point JSON numbers are invalid for financial fields.

## Comparability Rules

- A field is comparable only when the fixture contains a normalized expected value for the same case, method, year, asset, and source-row segment.
- Full-liquidation effects and method-specific lot or pool evidence are comparable only when the fixture records the evidence source IDs and expected values.
- Current committed fixtures use `rotki_backed` match evidence only.
- Scope-Local Hybrid (`scope_local_hybrid`) project-owned lifecycle assertions are currently represented by `unsupported_segments` with `comparison_policy: project_composition_only` rather than by committed `project_composition_rule` match rows.
- The schema still permits `project_composition_rule` match evidence. If added later, it must include a stable `composition_rule_id` and the source-row segment it covers.
- Unsupported fields must be reported as skipped with the unsupported reason and must not be counted as matched external-oracle assertions.
- Supported empirical fixture groups must not be skipped before project calculation and oracle comparison. Unsupported field-level segments may be skipped only when fixture metadata records an explicit reason.

## Unsupported Segment Rules

If the selected external oracle cannot represent a dataset segment without changing the financial meaning, the fixture must include an unsupported segment:

```json
{
  "case_id": "case-average-cost-delta-2024",
  "method": "average_cost",
  "activity_source_ids": ["emp-act-000085"],
  "reason": "Average-cost pool provenance remains outside the verified rotki aggregate oracle boundary",
  "comparison_policy": "skip_external_oracle"
}
```

Current committed Scope-Local Hybrid fixtures also use:

```json
{
  "case_id": "case-scope-local-reliable-epsilon-2024",
  "method": "scope_local_hybrid",
  "activity_source_ids": ["emp-act-000041", "emp-act-000042", "emp-act-000091", "emp-act-000092", "emp-act-000094"],
  "reason": "Hybrid lifecycle composition remains project-owned outside the repository-controlled composite scope slice",
  "comparison_policy": "project_composition_only"
}
```

Rules:

- `reason` is required.
- Unsupported segments must not fabricate expected values.
- Unsupported segments must not be silently omitted from method coverage reporting.
- Project-owned composition rules may compare Scope-Local Hybrid (`scope_local_hybrid`) lifecycle state only when documented by the fixture and test failure output.
- Zero-priced holding reductions are excluded from empirical external-oracle fixture scope after BUG-001 and must not remain as supported external-oracle fixture groups.

## External Oracle Invocation Rules

- Empirical tests read golden fixtures by default.
- External oracle generation is allowed only when a required fixture is absent or when an explicit regeneration command is used.
- Fixture-backed test runs use `go test ./tests/empirical -count=1 -v` and must not require `.cache/empiricaloracle/rotki-source/` while committed fixtures are present.
- Missing-fixture generation from the empirical tests is opt-in through `GHOSTFOLIO_CRYPTOGAINS_GENERATE_MISSING_FIXTURES=true`, which runs `go run ./tools/empiricaloracle`.
- Explicit full regeneration uses `go run ./tools/empiricaloracle --regenerate`.
- The command or adapter must resolve only repository-controlled external oracle boundaries and pinned source or artifact metadata.
- The command or adapter must not use a developer's default local accounting configuration, a global `rotki` executable, committed raw rotki payloads, or unpinned system installation.
- The command must pass explicit file arguments.
- The command or adapter must record oracle name, source URL, pinned version or commit, adapter constraints, and arguments before normalization.
- Missing, non-executable, or unsupported external oracle boundaries must fail fixture generation with an actionable setup error.

## External Oracle Provenance Contract

`third_party/rotki/` must include:

- applicable license text
- upstream source URL
- selected version or commit
- checksum for the pinned rotki source archive
- source provenance and corresponding source where the applicable license requires it
- supported source-cache, provenance, or adapter artifact paths
- platform support notes
- regeneration instructions
- statement that runtime application code must not link, import, or execute external-oracle tooling, oracle adapters, or composite oracle helpers

Superseded materials:

- Committed raw rotki outputs, hand-authored rotki datasets, and `third_party/rotki/source/` are invalid oracle evidence.

Binary-only vendoring is invalid unless the applicable license and source-distribution obligations are satisfied and documented.

## Security Contract

Oracle inputs and outputs must not contain:

- real tokens or JWTs
- real user financial data
- raw protected snapshot data
- app configuration paths
- generated Markdown reports
- TUI output
- copied upstream fixture content

Fixture validation must scan persisted oracle materials for secret-like strings before empirical tests accept them.
