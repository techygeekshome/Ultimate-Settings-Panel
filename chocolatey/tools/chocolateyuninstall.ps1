$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# Remove the short alias; Chocolatey removes the unpacked files and auto-shims itself.
$exe = Join-Path $toolsDir 'Ultimate Settings Panel.exe'
Uninstall-BinFile -Name 'usp' -Path $exe
