$ErrorActionPreference = 'Stop'

$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$servExe  = Join-Path $toolsDir 'serv.exe'

if (Test-Path $servExe) {
  Remove-Item $servExe -Force
}

# The Chocolatey shim for serv.exe is removed automatically on uninstall.
