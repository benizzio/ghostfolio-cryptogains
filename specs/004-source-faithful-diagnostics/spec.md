# Feature Specification: Source-faithful synced-data diagnostics

**Branch**: `[004-source-faithful-diagnostics]`  
**Date**: 2026-05-19  
**Input**: Issue "Simplify synced-data diagnostic records to mirror the actual activity record"

## Summary

Synced-data diagnostic reports should keep each offending record close to the activity record or upstream activity entry that caused the failure. The record payload must not add default top-level resolved or preferred financial values when those values are selected or derived by current validation rules instead of being source fields.

## Requirements

- Remove default diagnostic-record fields for selected top-level `unit_price`, `gross_value`, and `fee_amount`, including matching currency fields.
- Keep source financial fields that are present on the normalized activity record or upstream activity entry, including `order_*`, `asset_profile_*`, `base_*`, quantity, and currency identifiers.
- Stop using preferred or derived diagnostic amount helpers for generic offending-record context.
- Keep production diagnostic reports redacting remaining source financial values.
- Preserve diagnostic usefulness by keeping failure stage, failure detail, identity, activity, asset, source-scope, comment, data source, and source currency context.

## Out of Scope

- Changing validation amount-resolution behavior.
- Adding a separate explanatory diagnostic-data model unless a specific failure requires it.
- Changing diagnostic report storage, prompting, or generation flow.
