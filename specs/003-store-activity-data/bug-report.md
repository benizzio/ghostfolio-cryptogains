# BUG REPORT

To be used in `/speckit.bugfix.report` prompt. Only unmarked bugs should be read to be verified. Mark bugs here (done) after verified.

- [X] 1 - Message for sync error is insufficient to guide user and engineer troubleshooting
  - Solution: error reporting logs.
    - In prod mode: ask the user if they want to generate a detailed error report. When the error is related to synced data, add the record(s) that caused the error, redacting any financial value information.
    - In dev mode: generate the detailed error report automatically (without asking) and add ALL information about the record(s) that caused the error. Information like the security token should never be included
    - In both modes, show a message on the UI informing that the error report was generated and where to find it.
- [X] 2 - Dev mode in dev environment:
  - currently there is no Makefile command to run the application in dev mode. Add it.
- [X] 3 - Ghostfolio data fragility:
  - after some investigation, I found out that the `date` field from the contract for Ghostfolio's `/api/v1/activities` cannot be trusted for precision. Since Ghostfolio UI for activity management only allows selection of date (without time) on its form, time values can be arbitrary.
  - since the time part of the value cannot be trusted, the application should ignore (consider time as `00:00:00`) when processing deterministic same-asset ordering with `occurred_at`.
  - for deterministic same-asset ordering with internal data when records have the same `occurred_at`, the second tiebreaker filed should be `activity_type` (with `BUY` taking precedence over `SELL`), and only after that use `source_id` as a tiebreaker.
