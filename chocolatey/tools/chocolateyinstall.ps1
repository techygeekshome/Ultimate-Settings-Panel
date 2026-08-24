$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# USP 8.0 ships as a portable zip - no installer. Chocolatey unpacks it into the package's
# tools folder and auto-shims the executable, so it can be launched from any terminal.
$packageArgs = @{
  packageName    = 'ultimate-settings-panel'
  unzipLocation  = $toolsDir
  url            = 'https://github.com/techygeekshome/Ultimate-Settings-Panel/releases/download/v8.0.1/Ultimate-Settings-Panel.zip'
  checksum       = '975d3c213d8094bb5ad332c89c67a3a2fd440ea42aef3ebb1372ad9984ae8e3e'
  checksumType   = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# The shipped filename contains spaces, which makes for an awkward shim to type. Give it a
# short alias as well, so `usp` works from any terminal.
$exe = Join-Path $toolsDir 'Ultimate Settings Panel.exe'
if (Test-Path $exe) {
  Install-BinFile -Name 'usp' -Path $exe
} else {
  throw "Expected 'Ultimate Settings Panel.exe' inside the archive but it was not found."
}
