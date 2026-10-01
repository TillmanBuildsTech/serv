# Contributing to serv

Thanks for contributing to **serv** — a cross-platform native service wrapper and modern
successor to NSSM. This guide covers how to build and test the project locally, and how the
GitHub branch model and CI/CD workflows turn a commit into a public release.

---

## Prerequisites

- **Go** `1.25` or newer (see [`go.mod`](go.mod); CI tests `1.23`, `1.24`, and `1.25`).
- **Python 3** — only needed for the version-bump helper (`scripts/bump_version.py`).
- **make** (optional; handy for the wrapper targets below).

The project uses **no CGO** — `CGO_ENABLED=0` builds. There are no C dependencies to install.

---

## Building

```sh
# Build the serv CLI binary into bin/serv (or bin/serv.exe on Windows)
make build

# Equivalent, without make:
go build -o bin/serv ./cmd/serv
```

Cross-compile a binary for another platform (all targets are static, no CGO):

```sh
GOOS=linux   GOARCH=amd64 go build -o serv-linux-amd64   ./cmd/serv
GOOS=windows GOARCH=amd64 go build -o serv-windows-amd64.exe ./cmd/serv
GOOS=darwin  GOARCH=arm64 go build -o serv-darwin-arm64  ./cmd/serv
```

---

## Testing

The suite has two tiers.

### Unit tests

Run across every package with the race detector (this is what CI does):

```sh
go test -race ./...
# or
make test        # runs `go test -v ./...`
```

### Integration tests

The integration tier (under [`test/integration/`](test/integration/)) exercises real
service lifecycle behavior — install/start/stop, restart with backoff, log capture and
rotation, process-tree killing, and lifecycle hooks. They are gated behind the
`integration` build tag so they don't slow down the default run:

```sh
go test -tags=integration ./test/integration/...
```

> **Note:** the *elevated* integration tests (`elevated_*_test.go`) require real OS
> privileges (root on Linux/macOS, an elevated shell on Windows) and are filtered to the
> appropriate platform. CI runs them on Linux where it can. Locally, run them only when
> you actually want to exercise the privilege-escalation paths.

Other quality gates that run in CI and should pass before you open a PR:

```sh
go build ./...   # everything compiles
go vet ./...     # static analysis
```

---

## Branch model

The repository uses three long-lived branches plus short-lived feature branches.

```
task/* (feature worktree)  --push-->  dev   (CI: build + tests only)
        dev  --PR-->  release                (Prerelease workflow: rc builds)
        release  --PR-->  main             (Release workflow: tag + publish)
```

| Branch | Purpose | What CI does on it |
|---|---|---|
| `dev` | Integration branch where all in-flight work accumulates. Never published. | `CI` (build + tests) only. |
| `release` | RC staging. Each merged change here is testable as a pre-release. | `CI` **and** `Prerelease`. |
| `main` | Public releases only. Protect it; no direct pushes. | `CI`; `Release` fires on merged PRs. |

### Feature workflow

1. Branch from `dev`, ideally in an isolated git worktree so you never block others.
2. Implement in small, reviewable commits.
3. Open a **pull request into `dev`**. CI (build + tests) runs automatically.
4. Once it's green, merge. `dev` is where work sticks around; it is not deployed.

There is **no release-branch-based flow** per feature — the single `release` branch is the
staging area for whatever is about to ship.

### Staging a release

When a set of changes on `dev` is ready to test as a release candidate, open a **pull
request from `dev` into `release`** — never push or merge into `release` directly.
Merging that PR pushes `release`, which triggers the `Prerelease` workflow to build an
RC.

---

## Versioning

The current target version lives in one file:
[`internal/version/VERSION`](internal/version/VERSION) (e.g. `0.2.0`). It is bumped
**together with `CHANGELOG.md`**, never one without the other.

Use the helper (see its header for full behavior):

```sh
python3 scripts/bump_version.py 0.3.0 -m "One-line summary of the release"
```

This writes the new version to `VERSION` and inserts a `## [0.3.0]` entry under
`# Changelog`. Commit both together. The helper refuses to run if the version is not
greater than the current one, or if that version already has a tag on `origin`.

The bump happens **once per release cycle, committed on `dev` before opening the
`dev` → `release` PR** — so the first RC is never built from an unbumped version.

### Why CI goes red on `main` sometimes

The `version-guard` job (in `.github/workflows/ci.yml`) hard-fails any push/PR whose
`VERSION` already has a `vX.Y.Z` tag on `origin`. If you merge to `main` **without** bumping
the version first, that job fails with a clear message telling you to bump. That is the
guard working — it stops a release from double-shipping a version. If you see this red X
on a PR that's otherwise ready, bump the version and CHANGELOG, don't work around the guard.

---

## CI/CD workflows

### 1. `ci.yml` — build & test (every push/PR to `main`, `dev`, `release`)

- **Triggers:** push to `main`, `dev`, or `release`; PRs against those branches.
- **`version-guard`:** fails if `internal/version/VERSION` already has a release tag on
  `origin` (see above).
- **`test`:** on `ubuntu-latest` across Go `1.23`/`1.24`/`1.25` — `go build`, `go vet`,
  unit tests with `-race`, integration tests, and `make build`.

This is the only workflow that runs on `dev`; it **never** releases or publishes anything.

### 2. `prerelease.yml` — RC pre-releases (push to `release`, or manual)

- **Triggers:** every push to `release`; also `workflow_dispatch` for on-demand RC builds.
- Computes the next release-candidate number for the current `VERSION`
  (`v0.3.0-rc.1`, `-rc.2`, …) by scanning existing tags.
- Builds all six platform binaries (Windows/Linux/macOS × amd64/arm64), stamps the rc
  suffix into the build (never committed), tags `v<VERSION>-rc.N`, and publishes a GitHub
  **pre-release** with auto-generated release notes.

Use these pre-releases for manual/automated functional testing and to vet a feature before
it ships. Pre-releases are marked `prerelease: true`, so they stay hidden from the "latest"
release, and are **not** published to npm, Homebrew, Scoop, winget, or Chocolatey.

### 3. `release.yml` — production release (merged PR to `main`, or manual)

- **Triggers:** a **merged** pull request into `main` (`pull_request` closed + merged), or
  `workflow_dispatch`.
- **`check`:** warns and sets `should_release=false` if the version is already tagged
  (belt-and-suspenders on top of the CI guard).
- **`build`:** builds all six platform binaries (Windows/Linux/macOS × amd64/arm64).
- **`release`:** tags `v<VERSION>`, uploads binaries to a GitHub **release**, bumps the
  Homebrew/Scoop/winget packaging manifests via `scripts/update_packaging_version.py`,
  commits those manifest bumps back to `main` (as `tbt-devops[bot]`), publishes the npm
  package via trusted publishing, and triggers the Chocolatey job.
- **`chocolatey`:** on `windows-latest`, packs and pushes the Chocolatey package to the
  community feed.

The release workflow is the only path that ships to public package registries.

### 4. `publish-chocolatey.yml` — manual Chocolatey re-publish

- **Triggers:** `workflow_dispatch` only, with a `version` input.
- **Why it exists:** a manual escape hatch to (re)pack and re-push a specific released
  version to Chocolatey — for example when a release's choco package needs to be rebuilt
  after the fact, or as a fallback if the choco step in `release.yml` didn't run. It reads
  the requested `version`, pins the nuspec/install script, downloads the real release
  assets, and pins their SHA256 checksums before packing and pushing.

---

## How a release happens end to end

1. **Develop on `dev`** — merge feature PRs; CI runs build + tests.
2. **Cut the version** — when ready, bump `VERSION` + `CHANGELOG.md` on `dev`
   (`scripts/bump_version.py 0.3.0 -m "..."`) and commit.
3. **Open the `dev` → `release` PR** and merge it. Merging pushes `release`,
   which builds a `v0.3.0-rc.N` GitHub pre-release.
4. **Test the RC** — verify the pre-release binaries behave; push any last-minute
   fixes to `dev` and re-cut via a fresh `dev` → `release` PR if needed.
5. **Ship** — open the PR from `release` to `main`. On merge, `release.yml` tags
   `v0.3.0`, uploads the release, and publishes to npm/Homebrew/Scoop/winget/Chocolatey.

---

## Commit conventions

Keep commits small and focused. Conventional prefixes are used in this repo
(`feat:`, `fix:`, `chore:`, `docs:`, `ci:`) — match them. Write a message a teammate can
follow, and keep your branch's worktree clean when you're done.
