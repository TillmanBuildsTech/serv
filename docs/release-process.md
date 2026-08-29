# Release process

## Branches

- **Task branches** — created by kanban workers in isolated git worktrees, based off
  `dev`. Each task pushes its branch into `dev` on completion.
- **`dev`** — the integration branch where all in-flight work accumulates. CI runs on
  every push here (tests only — no release, no artifacts, no deploy). Safe for automated
  pushes. When a set of changes is ready to test together, cut/merge `dev` into `release`.
- **`release`** — RC cut for the next version. Every push here (i.e. every merged feature
  PR) triggers the [Prerelease](../.github/workflows/prerelease.yml) workflow, which builds
  all platform binaries and publishes them as a GitHub **pre-release** tagged
  `v<VERSION>-rc.N`, where `<VERSION>` is whatever is currently in
  [`internal/version/VERSION`](../internal/version/VERSION) and `N` auto-increments per
  push. Can also be triggered on demand via `workflow_dispatch`. Use these builds for
  manual and automated functional testing, and to land last-minute fixes before cutting a
  public release.
- **`main`** — public releases only. Merging `release` into `main` (via PR)
  triggers the existing [Release](../.github/workflows/release.yml) workflow:
  it tags `v<VERSION>`, builds binaries, publishes the GitHub release, bumps
  the Homebrew/Scoop/winget manifests, and publishes to npm and Chocolatey.

```
task/* (worktree)  --push-->  dev  (CI: tests only)
        dev  --cut/merge-->  release  --PR-->  main
                    |                        |
              prerelease.yml            release.yml
           (rc builds, GitHub        (tag, GitHub release,
            pre-release only)         npm, Homebrew, Scoop, winget, Chocolatey)
```

## Versioning

`internal/version/VERSION` always holds the plain target version (e.g.
`0.2.0`) — never an `-rc.N` suffix. The prerelease workflow stamps the rc
suffix into a checked-out copy of `VERSION` only for the duration of the
build (so the embedded `version.Version` in rc binaries reads e.g.
`0.2.0-rc.3`); that change is never committed.

### How versions and the CHANGELOG are bumped (enforced)

- **`CHANGELOG.md` is the curated, authoritative per-version log** (Option A).
  GitHub release notes are auto-generated as a convenience, but the CHANGELOG
  is the maintained record and is kept in sync with `VERSION`.
- **`VERSION` and `CHANGELOG.md` are always bumped together** — never one
  without the other — as a single step at the **`dev` → `release` cut**, before
  an RC is built. Use the helper:

  ```sh
  python3 scripts/bump_version.py 0.2.0 -m "One-line summary of the release"
  ```

  This writes the new version to `internal/version/VERSION` and inserts a
  `## [0.2.0]` entry under `# Changelog`. Commit both in the same commit.

- **CI guard (enforced):** the `version-guard` job in `ci.yml` fails the build
  if the `VERSION` in `internal/version/VERSION` already has a `vX.Y.Z` tag on
  `origin`. So a merge to `main` cannot ship a version that was already
  released — the bump must happen first.
- **Release workflow (belt-and-suspenders):** the `check` job in `release.yml`
  warns (and sets `should_release=false`) if the version is already tagged, so
  a stale `main` merge never double-ships.

This means:

- Bump `VERSION` + `CHANGELOG.md` once per release cycle, at the `dev` →
  `release` cut, via `scripts/bump_version.py`.
- Every subsequent push to `release` produces the next `-rc.N` for that same
  target version, with no further file edits needed.
- Merging `release` into `main` releases exactly that version, unsuffixed.

## What pre-releases do *not* do

To keep the public installation channels stable, pre-release builds:

- Are marked `prerelease: true` on GitHub (hidden from "latest release").
- Are **not** published to npm, Homebrew, Scoop, or winget — those only
  happen on the public `release.yml` run triggered by a merge to `main`.
