$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# serv ships a single static serv.exe per architecture (no 32-bit build).
# Pick the release asset + pinned SHA256 for the host architecture.
$arch = $env:PROCESSOR_ARCHITECTURE
switch ($arch) {
  'ARM64' {
    $url      = 'https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.9/serv-windows-arm64.zip'
    $checksum = '246518234627415aed32e0623d10b0b74ec5ae6e570351e98a136b528566166c'
  }
  default {
    # AMD64 (and any 32-bit x86 host, which will run the amd64 binary)
    $url      = 'https://github.com/TillmanBuildsTech/serv/releases/download/v0.1.9/serv-windows-amd64.zip'
    $checksum = 'ee35250e3bed366fc15aa2bb48edcb17247e9948c14bbf8b924ef25358081a06'
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
