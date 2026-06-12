# Repository-Controlled rotki Boundary

This directory records the repository-controlled rotki provenance required by
BUG-001 and the BUG-002 source-acquisition supersession. It does not vendor a
rotki source checkout, built artifact, or executable. This checkout therefore
does not rely on a developer-local rotki installation and does not claim rotki
was executed here.

## Why This Exists

- BUG-001 supersedes the previous hledger-only acceptance assumption.
- rotki is the planned pure-method external oracle boundary for `fifo`, `lifo`,
  `hifo`, and `average_cost` aggregate comparisons.
- `scope_local_hybrid` remains a composite oracle and is not documented here as a
  native rotki method.
- BUG-002 rejects committed raw rotki captures, hand-authored rotki datasets,
  developer-global installations, and vendored rotki source as oracle evidence.
- Explicit golden-fixture regeneration must download or reuse the verified
  pinned rotki source under `.cache/empiricaloracle/rotki-source/`, which is an
  untracked project-local cache path.
- Repository-controlled normalization inputs under `testdata/empirical/rotki/`
  are retained as historical bootstrap metadata only. They are not the source of
  regenerated oracle data.

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
- `README.md`: pinned provenance, adapter constraints, and bootstrap boundary policy.

## Platform Support And Boundary Scope

- The upstream README states rotki supports Windows, macOS, and Linux.
- This repository currently vendors no rotki executable, wheel, package lock, or
  local source tree under `third_party/rotki/`.
- Empirical tests must use committed golden fixtures by default and must not rely
  on a developer-local rotki installation.
- Explicit regeneration may only use the verified untracked source cache path
  documented above and must fail when provenance, checksum, commit or tag, or
  adapter constraints do not match.
- Normal fixture-backed empirical tests must not download rotki, require the
  untracked source cache, or invoke oracle generation while committed golden
  fixtures are present.

## Adapter Constraints

- Supported rotki-backed pure methods: `fifo`, `lifo`, `hifo`, `average_cost`.
- Supported comparison scope: aggregate yearly report values and documented
  supporting arithmetic derived from synthetic cases.
- `scope_local_hybrid` remains outside direct rotki execution scope and must stay
  partially project-owned.
- Zero-priced holding reductions are excluded from rotki-backed fixture generation.
- Cases must remain single-currency and synthetic-only.
- Runtime application code must not link, import, or execute rotki.

## hledger-Only Supersession

- The earlier hledger-only oracle assumption is superseded for BUG-001 acceptance.
- The motivating defect was that the hledger-backed empirical run skipped most
  supported fixture groups before project calculation and oracle comparison.

Authored by: OpenCode
