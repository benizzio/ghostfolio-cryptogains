# hledger Source Metadata

## Upstream Identity

- Project: `hledger`
- Upstream repository: `https://github.com/simonmichael/hledger`
- Upstream release tag: `https://github.com/simonmichael/hledger/releases/tag/1.99.2`
- Upstream source archive URL: `https://github.com/simonmichael/hledger/archive/refs/tags/1.99.2.tar.gz`
- Selected version: `1.99.2`
- Release status: prerelease
- Tag commit: `ad6068782cb03a0433546b80c62cd771a655ef15`
- License: `GPL-3.0-or-later`

## Selection Rationale

The feature research originally assumed stable hledger `1.52.1`. Direct
validation for this feature packet showed that the required
`FIFO/LIFO/HIFO/AVERAGE lot + gain behavior` is available only in prerelease
`1.99.2`, so the vendored oracle target is `1.99.2`.

## Complete Corresponding Source

| Item | Repository path | Checksum |
| --- | --- | --- |
| committed source archive | `third_party/hledger/source/hledger-source-1.99.2.tar.gz` | `sha256:1ea46d762f973fed0550ae57aee38d8036a754bf1e3064a27b307e2ecbeaccdf` |

The repository keeps the complete corresponding source as the upstream source
archive for tag `1.99.2`.

## Executable Metadata

The repository path contract is always `third_party/hledger/bin/<goos>-<goarch>/hledger`.

| Repo platform path | Repository path | Commit status | Committed artifact checksum | Validated upstream release asset digest |
| --- | --- | --- | --- | --- |
| `linux-amd64` | `third_party/hledger/bin/linux-amd64/hledger` | committed | `sha256:801f1abfae1bf3b741567a7eea9ee6a4227544b6dcdb02b71b36ffcd26cec409` | `sha256:4d94d701b1a9e82aa2ea1b9997ddadbd94fecba21b4bdce9f4c85e8c1a3d2b9e` |
| `darwin-arm64` | `third_party/hledger/bin/darwin-arm64/hledger` | not committed | not applicable | `sha256:55bcb1d8341902f751d8b79b27d6f01f3c51cd7453996d7bd1eba17a2a567292` |
| `darwin-amd64` | `third_party/hledger/bin/darwin-amd64/hledger` | not committed | not applicable | `sha256:bb5090978b84e9957fe2d7052703ec000f5a6161908ce0c3b813386450674bfe` |
| `windows-amd64` | `third_party/hledger/bin/windows-amd64/hledger` | not committed | not applicable | `sha256:5c5881924727e2635a9f69f88191bce8ef924c23009688f305c3df13e1198ee2` |

Only `linux-amd64` is repo-supported in this packet because it is the only
committed executable artifact. The other upstream digests are recorded so future
vendoring work can verify them before adding those platform artifacts.

## Compliance Notes

- The vendored executable must never be committed without the matching complete
  corresponding source.
- Runtime application code must not link, import, or execute hledger.
- Empirical tooling must not fetch or build hledger during normal test
  execution.
