# Contract: ~~hledger Oracle Output~~ External Oracle Output

**Bugfix**: 2026-06-10 — [BUG-001] Superseded the hledger-only output contract with rotki-backed pure-method fixtures and Scope-Local Hybrid composite-oracle fixture rules.

## Scope

This contract defines generated external-oracle input files, normalized oracle golden fixtures, fixture metadata, unsupported-case handling, rotki-backed pure-method oracle expectations, Scope-Local Hybrid composite-oracle expectations, and retained hledger material expectations when applicable.

## Locations

Generated external-oracle inputs:

```text
testdata/empirical/rotki/
testdata/empirical/hledger/       # retained only when auxiliary or historical hledger inputs remain relevant
```

Normalized golden fixtures:

```text
testdata/empirical/golden/
```

External oracle materials:

```text
third_party/rotki/
third_party/hledger/       # retained only when auxiliary or historical hledger materials remain relevant
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
  "case_id": "case-fifo-basic-2024",
  "method": "fifo",
  "year": 2024,
  "asset_identity_key": "asset-alpha",
  "values": {
    "realized_gain_or_loss": "5",
    "allocated_basis": "10",
    "closing_quantity": "0",
    "closing_basis": "0"
  },
  "matches": [
    {
      "disposed_source_id": "emp-act-000010",
      "acquisition_source_id": "emp-act-000001",
      "matched_quantity": "1",
      "matched_basis": "10",
      "matched_proceeds": "15",
      "matched_gain_or_loss": "5"
    }
  ],
  "unsupported_segments": [],
  "metadata": {
    "oracle_name": "rotki",
    "source_url": "https://github.com/rotki/rotki",
    "version_or_commit": "<pinned-version-or-commit>",
    "adapter_arguments": ["--method", "fifo", "--input", "testdata/empirical/rotki/fifo.json"],
    "adapter_constraints": ["zero-priced reductions excluded from external-oracle fixture generation"],
    "decimal_policy": "scale=16,rounding=half_up",
    "dataset_input_hash": "sha256:...",
    "external_oracle_input_hash": "sha256:...",
    "normalization_version": "1",
    "composite_rule_version": null,
    "financial_tolerances": {
      "realized_gain_or_loss": "0.0000000000000001",
      "allocated_basis": "0.0000000000000001",
      "closing_basis": "0.0000000000000001"
    },
    "tolerance_notes": {
      "realized_gain_or_loss": "One-unit residual from external-oracle output scale after decimal-policy alignment for this fixture"
    },
    "oracle_output_hash": "sha256:..."
  }
}
```

## Required Metadata

Every golden fixture must include:

- external oracle name
- external oracle source URL
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
- Additional external-oracle-aligned accepted values may be added only when the selected external oracle cannot align with the production default, and each added value must be documented with the oracle name, pinned version or commit, and reason.
- If the selected external oracle cannot be configured or normalized to the production policy for every valid case, empirical tests must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` before project calculation runs and fixtures must record the external-oracle-aligned policy used.
- Residual financial differences after decimal-policy alignment may use documented per-field tolerances. Quantity tolerance is always zero.
- Non-zero financial tolerances must not exceed one unit at the selected decimal-policy scale. For the production 16-decimal policy, the maximum is `0.0000000000000001`.
- Every non-zero financial tolerance must include a tolerance note explaining why exact equality is not achievable for that external-oracle-derived value.
- Floating-point JSON numbers are invalid for financial fields.

## Comparability Rules

- A field is comparable only when the fixture contains a normalized expected value for the same case, method, year, asset, and source-row segment.
- Full-liquidation effects and method-specific lot or pool evidence are comparable only when the fixture records the evidence source IDs and expected values.
- Scope-Local Hybrid (`scope_local_hybrid`) assertions must be labeled `rotki_backed` or `project_composition_rule`.
- A `project_composition_rule` assertion must include a stable rule ID and the source-row segment it covers.
- Unsupported fields must be reported as skipped with the unsupported reason and must not be counted as matched external-oracle assertions.
- Supported empirical fixture groups must not be skipped before project calculation and oracle comparison. Unsupported field-level segments may be skipped only when fixture metadata records an explicit reason.

## Unsupported Segment Rules

If the selected external oracle cannot represent a dataset segment without changing the financial meaning, the fixture must include an unsupported segment:

```json
{
  "case_id": "case-selected-oracle-unrepresentable",
  "method": "fifo",
  "activity_source_ids": ["emp-act-000090"],
  "reason": "selected external oracle cannot represent this field-level segment without changing financial meaning",
  "comparison_policy": "skip_external_oracle"
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
- The command or adapter must resolve only repository-controlled external oracle boundaries and pinned source or artifact metadata.
- The command or adapter must not use a developer's default local accounting configuration, external user data, or unpinned system installation.
- The command must pass explicit file arguments.
- The command or adapter must record oracle name, source URL, pinned version or commit, adapter constraints, and arguments before normalization.
- Missing, non-executable, or unsupported external oracle boundaries must fail fixture generation with an actionable setup error.

## External Oracle Provenance Contract

`third_party/rotki/` and any retained `third_party/hledger/` materials must include:

- applicable license text
- upstream source URL
- selected version or commit
- checksum for vendored source or source artifact
- checksum for each supported executable or adapter artifact
- source provenance and corresponding source where the applicable license requires it
- supported executable, source, or adapter artifact paths
- platform support notes
- regeneration instructions
- statement that runtime application code must not link, import, or execute hledger, rotki, oracle adapters, or composite oracle helpers

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
