$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# serv ships a single static serv.exe per architecture (no 32-bit build).
# Pick the release asset + pinned SHA256 for the host architecture.
$arch = $env:PROCESSOR_ARCHITECTURE
switch ($arch) {
  'ARM64' {
    $url      = 'https://github.com/TillmanBuildsTech/serv/releases/download/v0.3.0/serv-windows-arm64.zip'
    $checksum = '3e107c9fdd738afbf1c3928ad1eb170fa6d22dc1aa60ac3e23354ddbbd6f252d'
  }
  default {
    # AMD64 (and any 32-bit x86 host, which will run the amd64 binary)
    $url      = 'https://github.com/TillmanBuildsTech/serv/releases/download/v0.3.0/serv-windows-amd64.zip'
    $checksum = '9fcfaa58a76b7b5b2e4996d7dc48b5411795d7de53e86dfd7188bb3757837059'
  }
}

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url           = $url
  checksum      = $checksum
  checksumType  = 'sha256'
  softwareName  = 'serv*'
}

Install-ChocolateyZipPackage @packageArgs

# Install-ChocolateyZipPackage drops serv.exe into $toolsDir; Chocolatey
# auto-creates a shim (serv on PATH) for it on package install.
