# Publishing serv to Chocolatey

This documents how to build and publish the `serv` Chocolatey package to the
[community repository](https://community.chocolatey.org/packages/serv) so
users can `choco install serv`.

The package follows community-package conventions: a `serv.nuspec`, a
`tools/chocolateyInstall.ps1` that downloads the official `serv.exe` for the
host architecture and verifies its SHA256 checksum, a
`tools/chocolateyUninstall.ps1`, and a `tools/VERIFICATION.txt` for the
package verifier.

## Prerequisites

- A Windows machine (or a Windows VM) with [Chocolatey installed](https://chocolatey.org/install).
- A [Chocolatey Community Repository API key](https://community.chocolatey.org/account) from an account
  approved to publish. Request access by following the
  [package maintainer steps](https://docs.chocolatey.org/en-us/community-repository/commands/push).

## 1. Build the package

`choco pack` must run on Windows (it produces the `.nupkg` from the nuspec):

```powershell
cd packaging/chocolatey
choco pack
```

This produces `serv.0.1.9.nupkg`. The package version and the two pinned
SHA256 checksums are bumped automatically by
[`scripts/update_packaging_version.py`](../scripts/update_packaging_version.py)
during each release, so you normally just rebuild after a release.

> On non-Windows hosts the `.nupkg` is just a zip with `serv.nuspec` at the
> root plus `tools/`; it can be assembled with `zip` and validated with
> `unzip -l`, but only `choco pack`/`choco install` can exercise it end to end.

## 2. Verify locally

```powershell
choco install serv --source packaging/chocolatey -y
serv version
choco uninstall serv -y
```

## 3. Publish to the community feed

```powershell
choco apikey --key <API_KEY> --source https://push.chocolatey.org/
choco push serv.0.1.9.nupkg --source https://push.chocolatey.org/
```

The first push goes through the [package verifier](https://docs.chocolatey.org/en-us/community-repository/moderation/package-verifier)
and manual moderation before it is listed publicly.

## 4. Keep in sync with releases

`scripts/update_packaging_version.py` bumps the nuspec version, the download
URLs, and the two SHA256 checksums on every release — the same automation
that updates Homebrew/Scoop/winget/npm. No manual edits needed; just rebuild
and re-push after each release.
