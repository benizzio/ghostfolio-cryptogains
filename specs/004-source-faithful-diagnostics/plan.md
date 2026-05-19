# Implementation Plan: Source-faithful synced-data diagnostics

**Branch**: `[004-source-faithful-diagnostics]`  
**Date**: 2026-05-19  
**Spec**: `/specs/004-source-faithful-diagnostics/spec.md`
**Process**: Summarized version; not part of a full Spec Kit process.

## Summary

Make a focused diagnostic-model change so the `records` section mirrors the offending activity data instead of including resolved or derived current-slice money views. Keep the existing diagnostic report flow and redaction path, but apply redaction to the remaining source financial fields only.

## Technical Plan

1. Update `internal/sync/model/diagnostic_context.go`:
   - remove top-level selected financial fields from `DiagnosticRecord`
   - remove `ResolveActivityAmounts()` use when converting normalized records
   - keep explicit source-tier amount fields and currency identifiers
2. Update `internal/ghostfolio/mapper/activity_mapper.go`:
   - stop populating preferred or derived top-level financial diagnostic fields
   - remove mapper-only preferred and derived diagnostic helpers
3. Update `internal/app/runtime/diagnostic_report.go`:
   - keep redaction for quantity and remaining source-tier financial fields
4. Update focused tests:
   - assert diagnostic records preserve source-tier amounts
   - assert no generic preferred or derived fields are expected
   - assert production redaction still removes remaining financial values
5. Validate with targeted package tests, then the full existing test suite.

## Validation

- `go test ./internal/sync/model ./internal/ghostfolio/mapper ./internal/app/runtime ./tests/integration`
- `make test`
