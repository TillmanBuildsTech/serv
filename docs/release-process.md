# Release process

## Branches

- **Feature branches** — PR into `release`.
- **`release`** — integration branch for the next version. Every push here
  (i.e. every merged feature PR) triggers the [Prerelease](../.github/workflows/prerelease.yml)
  workflow, which builds all platform binaries and publishes them as a
  GitHub **pre-release** tagged `v<VERSION>-rc.N`, where `<VERSION>` is
  whatever is currently in [`internal/version/VERSION`](../internal/version/VERSION)
  and `N` auto-increments per push. Use these builds for manual and automated
  functional testing, and to land last-minute fixes before cutting a public
  release.
- **`main`** — public releases only. Merging `release` into `main` (via PR)
  triggers the existing [Release](../.github/workflows/release.yml) workflow:
  it tags `v<VERSION>`, builds binaries, publishes the GitHub release, bumps
  the Homebrew/Scoop/winget manifests, and publishes to npm.

```
feature/*  --PR-->  release  --PR-->  main
                       |                |
                 prerelease.yml    release.yml
              (rc builds, GitHub    (tag, GitHub release,
               pre-release only)     npm, Homebrew, Scoop, winget)
```

## Versioning

`internal/version/VERSION` always holds the plain target version (e.g.
`0.2.0`) — never an `-rc.N` suffix. The prerelease workflow stamps the rc
suffix into a checked-out copy of `VERSION` only for the duration of the
build (so the embedded `version.Version` in rc binaries reads e.g.
`0.2.0-rc.3`); that change is never committed. This means:

- Bump `VERSION` to the next target release once, when starting work on it
  (on the `release` branch), per the existing [CLAUDE.md](../.claude/CLAUDE.md) convention.
- Every subsequent push to `release` produces the next `-rc.N` for that same
  target version, with no further file edits needed.
- Merging `release` into `main` releases exactly that version, unsuffixed.

## What pre-releases do *not* do

To keep the public installation channels stable, pre-release builds:

- Are marked `prerelease: true` on GitHub (hidden from "latest release").
- Are **not** published to npm, Homebrew, Scoop, or winget — those only
  happen on the public `release.yml` run triggered by a merge to `main`.
