# Contract: hledger Oracle Output

## Scope

This contract defines generated hledger input files, normalized oracle golden fixtures, fixture metadata, unsupported-case handling, and hledger vendoring expectations.

## Locations

Generated hledger journals:

```text
testdata/empirical/hledger/
```

Normalized golden fixtures:

```text
testdata/empirical/golden/
```

hledger vendoring materials:

```text
third_party/hledger/
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
    "hledger_version": "1.52.1",
    "command_arguments": ["-f", "testdata/empirical/hledger/fifo.journal", "print"],
    "decimal_policy": "scale=16,rounding=half_up",
    "dataset_input_hash": "sha256:...",
    "hledger_input_hash": "sha256:...",
    "normalization_version": "1",
    "financial_tolerances": {
      "realized_gain_or_loss": "0.0000000000000001",
      "allocated_basis": "0.0000000000000001",
      "closing_basis": "0.0000000000000001"
    },
    "oracle_output_hash": "sha256:..."
  }
}
```

## Required Metadata

Every golden fixture must include:

- hledger version
- exact command arguments
- selected decimal policy
- documented financial tolerances
- dataset input hash
- generated hledger input hash
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
- If hledger cannot be configured or normalized to the production policy for every valid case, empirical tests must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` before project calculation runs and fixtures must record the hledger-aligned policy used.
- Residual financial differences after decimal-policy alignment may use documented tight per-field tolerances. Quantity tolerance is always zero.
- Tolerances must be small enough to catch material drift and systematic method differences.
- Floating-point JSON numbers are invalid for financial fields.

## Unsupported Segment Rules

If hledger cannot represent a dataset segment without changing the financial meaning, the fixture must include an unsupported segment:

```json
{
  "case_id": "case-zero-reduction-unrepresentable",
  "method": "fifo",
  "activity_source_ids": ["emp-act-000090"],
  "reason": "hledger syntax cannot represent this zero-priced holding reduction without producing gain/loss for this case",
  "comparison_policy": "skip_external_oracle"
}
```

Rules:

- `reason` is required.
- Unsupported segments must not fabricate expected values.
- Unsupported segments must not be silently omitted from method coverage reporting.
- Project-owned composition rules may compare scope-local hybrid lifecycle state only when documented by the fixture and test failure output.

## hledger Invocation Rules

- Empirical tests read golden fixtures by default.
- hledger generation is allowed only when a required fixture is absent or when an explicit regeneration command is used.
- The command must be the repository-vendored hledger executable or a wrapper that resolves only that vendored tool.
- The command must not use a developer's default `LEDGER_FILE` or hledger config.
- The command must pass explicit file arguments.
- The command must record version output before normalization.
- Missing, non-executable, or unsupported hledger must fail fixture generation with an actionable setup error.

## Vendoring Contract

`third_party/hledger/` must include:

- GPL-3.0-or-later license text
- upstream source URL
- selected hledger version
- checksum for vendored source or artifact
- source or complete corresponding source for any executable artifact
- platform support notes
- regeneration instructions
- statement that runtime application code must not link, import, or execute hledger

Binary-only vendoring is invalid.

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
