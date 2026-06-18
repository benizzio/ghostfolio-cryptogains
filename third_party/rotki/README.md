# Repository-Controlled rotki Boundary

This directory records the pinned rotki provenance used by empirical fixture
regeneration. It does not vendor a rotki source checkout, built artifact, or
executable. Normal empirical tests use committed golden fixtures and do not
execute rotki.

## Purpose

- rotki is the pure-method external oracle boundary for `fifo`, `lifo`, `hifo`,
  and `average_cost` aggregate comparisons.
- `scope_local_hybrid` remains a composite oracle and is not documented here as a
  native rotki method.
- Explicit golden-fixture regeneration must download or reuse the verified
  pinned rotki source under `.cache/empiricaloracle/rotki-source/`, which is an
  untracked project-local cache path.
- Regeneration must execute only through the project-owned adapter boundary in
  `tools/empiricaloracle`.
- Regeneration must not use committed raw rotki captures, hand-authored rotki
  datasets, developer-global installations, or vendored rotki source as oracle
  evidence.

## Upstream Pin And Verifiable Sources

- Upstream repository: `https://github.com/rotki/rotki`
- Pinned release tag: `v1.43.1`
- Resolved commit: `a2e00be49a0ea36e7563a5d235cfa6a7c91edbfb`
- Signed tag object: `b35a3c934eedf23b1387ff564b6386fb2ce3f201`
- Source archive URL: `https://github.com/rotki/rotki/archive/refs/tags/v1.43.1.tar.gz`
- Source archive SHA-256: `8434b653104f8d5b0638e98d88a5ef256fac7720cc459eb33b729e2848900e3b`
- License source URL: `https://raw.githubusercontent.com/rotki/rotki/v1.43.1/LICENSE.md`
- License source SHA-256: `eb6f58a98d8bdb6d3c8fee3817543589f3cd0921d14748fa0630edff2d4c08b0`

## Included Materials

- `LICENSE.md`: exact AGPLv3 license text copied from the pinned upstream license URL.
- `README.md`: pinned provenance, adapter constraints, and regeneration boundary policy.

## Platform Support And Boundary Scope

- The upstream README states rotki supports Windows, macOS, and Linux.
- This repository currently vendors no rotki executable, wheel, package lock, or
  local source tree under `third_party/rotki/`.
- Empirical tests must use committed golden fixtures by default and must not rely
  on a developer-local rotki installation.
- Explicit regeneration may only use the verified untracked source cache path
  documented above and must fail when provenance, checksum, commit or tag, or
  adapter constraints do not match.
- Regeneration must not read committed raw rotki payloads, hand-authored adapter
  inputs, or any vendored `third_party/rotki/source/` checkout.
- Normal fixture-backed empirical tests must not download rotki, require the
  untracked source cache, or invoke oracle generation while committed golden
  fixtures are present.

## Authentication Model

- The archive download uses an unauthenticated HTTPS `GET` request to the pinned
  GitHub archive URL in `tools/empiricaloracle/rotki_source.go`.
- Remote tag verification uses unauthenticated `git ls-remote --tags` against
  the public upstream repository and checks both the signed tag object and the
  peeled commit identity.
- No GitHub token, application credential, SSH key, cookie, or developer-local
  rotki login is required or read by the regeneration boundary.

## Expected External Failure Modes

- GitHub or network outage during archive download or `git ls-remote` tag
  verification.
- HTTP status other than `200 OK` for the pinned source archive.
- Missing local `git`, `python3`, or `python` executable during explicit
  regeneration.
- Archive checksum, signed tag object, peeled commit, manifest, source root, or
  adapter path mismatch.
- Corrupt archive, invalid archive paths, or extraction failure under the
  untracked `.cache/empiricaloracle/rotki-source/` directory.

## Security Implications

- Normal empirical tests consume committed golden fixtures and must not contact
  GitHub or execute rotki source.
- Explicit regeneration trusts only the pinned HTTPS archive after SHA-256
  verification and independent `git ls-remote` tag identity checks.
- The untracked source cache is regeneration-only. It must not be committed,
  imported by runtime code, or used as a developer-global rotki installation.
- Failure output should report boundary mismatches and setup errors without
  including credentials because this boundary has no credential input.

## Adapter Constraints

- Supported rotki-backed pure methods: `fifo`, `lifo`, `hifo`, `average_cost`.
- Supported comparison scope: aggregate yearly report values and documented
  supporting arithmetic derived from synthetic cases.
- `scope_local_hybrid` remains outside direct rotki execution scope and must stay
  partially project-owned.
- Zero-priced holding reductions are excluded from rotki-backed fixture generation.
- Cases must remain single-currency and synthetic-only.
- Runtime application code must not link, import, or execute rotki.

Authored by: OpenCode
