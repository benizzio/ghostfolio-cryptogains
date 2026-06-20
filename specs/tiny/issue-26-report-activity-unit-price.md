# TinySpec: Report Activity Unit Price Column

**Branch**: 007-issue-26-report-activity-unit-price  
**Date**: 2026-06-20  
**Status**: done
**Complexity**: small

## What

Address GitHub issue #26 by adding `Unit Price` immediately after `Quantity` in the Markdown report's `Asset Detail > In-Year Activity` table. The report model already carries `AssetActivityRow.UnitPrice`, so this is a rendering and test-contract change only.

## Context

Existing numbered spec directories, including `specs/006-empirical-financial-tests`, are retained as historical completed feature artifacts. `.specify/feature.json` points only to the active tinyspec context for this work.

| File | Role |
|------|------|
| `internal/report/markdown/renderer.go` | Will be modified - add the column header, separator, row value, and unit-price render error context |
| `internal/report/markdown/renderer_internal_test.go` | Will be modified - cover renderer row output and invalid optional unit-price handling |
| `tests/unit/report_markdown_test.go` | Will be modified - update unit-level Markdown table header and expected rows |
| `tests/contract/markdown_report_contract_test.go` | Will be modified - update externally visible Markdown document contract |
| `internal/report/model/asset_activity_row.go` | Context - `AssetActivityRow.UnitPrice` already exists and validates as optional decimal |
| `internal/report/calculate/artifacts.go` | Context - calculated rows already copy `ActivityCalculationInput.UnitPrice` into `AssetActivityRow` |
| `internal/report/calculate/activity_input.go` | Context - selected activity inputs already derive or preserve unit price |

## Requirements

1. The Markdown `In-Year Activity` table MUST include `Unit Price` immediately after `Quantity`.
2. Each activity row MUST render `AssetActivityRow.UnitPrice` as a canonical exact decimal when present and as a blank cell when absent.
3. Invalid non-finite unit-price values MUST return a wrapped render error identifying the affected activity row and field.
4. Existing financial calculations, row ordering, currency behavior, redaction, and persistence MUST remain unchanged.

## Plan

1. Update `writeActivityBlock` in `internal/report/markdown/renderer.go` to canonicalize `row.UnitPrice` with `canonicalDecimalPointer` and insert it after quantity.
2. Update the Markdown table header and separator to include `Unit Price` in the same position.
3. Update unit and contract Markdown expectations and fixtures to verify populated, zero, and blank unit-price cells.
4. Add or adjust internal renderer error coverage for invalid optional unit price.

## Tasks

- [x] Add the `Unit Price` column to `internal/report/markdown/renderer.go`.
- [x] Add unit-price render error coverage in `internal/report/markdown/renderer_internal_test.go`.
- [x] Update `tests/unit/report_markdown_test.go` table header and row assertions.
- [x] Update `tests/contract/markdown_report_contract_test.go` visible Markdown contract assertions.
- [x] Run `go test ./internal/report/markdown ./tests/unit ./tests/contract`.

## Done When

- [x] All tasks checked off
- [x] Relevant tests pass
- [x] No lint or formatting errors
