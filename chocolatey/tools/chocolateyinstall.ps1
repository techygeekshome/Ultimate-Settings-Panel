$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# USP 8.0 ships as a portable zip - no installer. Chocolatey unpacks it into the package's
# tools folder and auto-shims the executable, so it can be launched from any terminal.
$packageArgs = @{
  packageName    = 'ultimate-settings-panel'
  unzipLocation  = $toolsDir
  url            = 'https://github.com/techygeekshome/Ultimate-Settings-Panel/releases/download/v8.0.0/Ultimate-Settings-Panel.zip'
  checksum       = '470d6caf7c1d7926cfe041f86cd66b255a35b8956084608d8defa89929626e7b'
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
