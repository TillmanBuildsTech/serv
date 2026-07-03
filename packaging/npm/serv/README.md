# @tillmanbuildstech/serv

npm wrapper for [serv](https://github.com/TillmanBuildsTech/serv), a
cross-platform Windows service / systemd / launchd process supervisor.

This package does not contain the `serv` binary itself. On install, a
postinstall script downloads the release archive matching your platform
from [GitHub Releases](https://github.com/TillmanBuildsTech/serv/releases),
verifies its SHA256 checksum against the value pinned in `checksums.json`,
and extracts it into `bin/`.

## Usage

Run once without installing anything permanently:

```sh
npx @tillmanbuildstech/serv status myapp
```

Or install it:

```sh
npm install -g @tillmanbuildstech/serv
serv status myapp
```

Supported platforms: Windows, macOS, Linux — amd64 and arm64.

See the main [installation docs](https://github.com/TillmanBuildsTech/serv/blob/main/docs/installation.md)
for other install methods (Scoop, winget, Homebrew, `go install`).
