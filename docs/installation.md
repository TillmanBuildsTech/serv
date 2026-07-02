# Installation

Serv is distributed as a single static binary, `serv` (`serv.exe` on
Windows). Pick whichever method fits your platform.

## Windows

### Scoop

Once a bucket is published, install with:

```powershell
scoop bucket add tillmanbuildstech <bucket-url>
scoop install serv
```

The manifest itself is at
[`packaging/scoop/serv.json`](../packaging/scoop/serv.json) — point Scoop's
`checkver`/`autoupdate` at your release download URLs once releases are
published, or install directly from the manifest file:

```powershell
scoop install packaging/scoop/serv.json
```

### winget

The manifest files live under [`packaging/winget/`](../packaging/winget/).
Once submitted to the winget community repository (or hosted privately),
install with:

```powershell
winget install TillmanBuildsTech.Serv
```

### Binary download / build from source

See [Building from source](#building-from-source) below, or download a
release archive from this repository's releases page once one is published.

## macOS

### Homebrew

The formula is at
[`packaging/homebrew/serv.rb`](../packaging/homebrew/serv.rb). Once
published to a tap, install with:

```sh
brew tap tillmanbuildstech/tap
brew install serv
```

Or install directly from the formula file:

```sh
brew install --formula packaging/homebrew/serv.rb
```

## Linux

Serv ships as a plain binary; no distribution-specific package is required.
Download or build the `serv` binary for your architecture and place it on
your `PATH` (e.g. `/usr/local/bin`).

## Building from source

Requires Go 1.25 or later.

```sh
git clone https://github.com/TillmanBuildsTech/serv.git
cd serv
make build
```

This produces `bin/serv` (`bin/serv.exe` on Windows). Run `make test` to run
the unit test suite, or `go test -tags=integration ./test/integration/...`
for the integration suite (installing a real service requires
Administrator/root privileges; those specific tests skip themselves
otherwise).

## Verifying the install

```sh
serv version
```

## Next steps

- [Configuration reference](configuration.md) — every YAML field, its type,
  default, and an example.
- [Hooks](hooks.md) — running commands at lifecycle events.
- The [README](../README.md) quick start walks through installing your
  first service end to end.
