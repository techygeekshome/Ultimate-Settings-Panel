# Chocolatey package — ultimate-settings-panel

Source for the [community.chocolatey.org](https://community.chocolatey.org/packages/ultimate-settings-panel)
package. The `packageSourceUrl` in the nuspec points at this repository, so this is where the
package source belongs.

## What it does

The package does **not** embed the application. `chocolateyinstall.ps1` downloads
`Ultimate-Settings-Panel.zip` from the GitHub release for the matching tag, verifies it against a
SHA-256 checksum, unpacks it into the package's `tools` folder, and lets Chocolatey auto-shim the
executable. It also registers a short `usp` alias, because the shipped filename contains spaces.

Because the archive is downloaded rather than embedded, the package must **not** contain a
`tools\VERIFICATION.txt`. That file is only for packages that embed a binary. Including it is
what the 8.0.0 submission was rejected for.

## Building and pushing a new version

```powershell
# from this folder
choco pack
choco push ultimate-settings-panel.<version>.nupkg --source https://push.chocolatey.org/
```

`choco push` needs the API key from your community.chocolatey.org account
(`choco apikey --key <key> --source https://push.chocolatey.org/`, done once per machine).

## Checklist for a new release

1. Cut the GitHub release and note the asset URL.
2. `Get-FileHash Ultimate-Settings-Panel.zip -Algorithm SHA256` and put the hash in
   `tools/chocolateyinstall.ps1` alongside the new URL.
3. Bump `<version>` and `<releaseNotes>` in the nuspec.
4. `choco pack`, then install locally from the nupkg to check the shim works.
5. `choco push`.
