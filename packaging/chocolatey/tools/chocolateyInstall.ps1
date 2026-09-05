$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# serv ships a single static serv.exe per architecture (no 32-bit build).
# Pick the release asset + pinned SHA256 for the host architecture.
$arch = $env:PROCESSOR_ARCHITECTURE
switch ($arch) {
  'ARM64' {
    $url      = 'https://github.com/TillmanBuildsTech/serv/releases/download/v0.2.0/serv-windows-arm64.zip'
    $checksum = '6706eb7f00b70886844440778da212090883eb60ae149fa28446fe9d2df61848'
  }
  default {
    # AMD64 (and any 32-bit x86 host, which will run the amd64 binary)
    $url      = 'https://github.com/TillmanBuildsTech/serv/releases/download/v0.2.0/serv-windows-amd64.zip'
    $checksum = '23509306da91521c6d266baa4d67f78e88cf539e07bc086ac4bc50dbfd1fa849'
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
