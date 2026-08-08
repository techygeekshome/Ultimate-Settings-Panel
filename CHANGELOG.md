# Changelog

All notable changes to Ultimate Settings Panel are documented here.
This project loosely follows [Keep a Changelog](https://keepachangelog.com/) and [Semantic Versioning](https://semver.org/).

## [8.0.0] - 2026-08-07

A complete, ground-up reimagining of the Ultimate Settings Panel: a native Windows app **and** a portable web app, replacing the original C# / WinForms program.

### Added
- **Native Windows app** (`Ultimate Settings Panel.exe`) - renders the panel in a Microsoft Edge **WebView2** window (Chromium) and launches tools through a small JavaScript-Go bridge (`window.uspRun`). Every item has a real **Run** button. No install, Windows 10 & 11.
- **Web app** (`index.html`) - a single self-contained HTML/CSS/JavaScript file. No install, no dependencies, works offline and on any device.
- **Instant search** across names, descriptions and the underlying commands (`/` to focus, `Esc` to clear in the web version).
- New **Windows 11 Settings** category - ~70 `ms-settings:` deep links that open the real Settings pages directly.
- New **Terminal & Shell** category - Windows Terminal, PowerShell 7 (`pwsh`) and `winget` app-management commands.
- Modern **diagnostics** - `sfc /scannow`, `DISM .../RestoreHealth`, `powercfg /batteryreport`, `powercfg /energy`, Wi-Fi report, Winsock and TCP/IP reset, and more.
- **Favourites** (saved locally), **light / dark theme** with system detection, **Copy** and **.bat export** (web version).
- 250+ items across 13 categories in total.

### Changed
- All TechyGeeksHome links updated from the old `blog.techygeekshome.info` domain to `techygeekshome.info` (slugs unchanged).
- Twitter links updated to **X** (`x.com`).
- Dead telnet "tricks" replaced with living web equivalents (ASCIImation Star Wars, Telehack, `wttr.in`).
- Internet Explorer entries replaced with Microsoft Edge (including IE mode).

### Fixed
- **Windows Terminal (admin)** now actually elevates (`Start-Process wt -Verb RunAs`) instead of launching a normal `wt`.

### Removed
- **Google Analytics** tracking - Universal Analytics was permanently shut down in July 2023. The app now collects nothing.
- **Internet Explorer** launchers - IE was retired by Microsoft in 2022.
- **Google+** links - the service was discontinued.
- The unused COM `PlaLibrary` reference and the old blog-hosted XML updater from the original app.

### Security
- The desktop app no longer uses `mshta` or drops a script to a temporary folder, avoiding the "living-off-the-land" pattern that anti-virus/EDR heuristics flag.

---

## [6.6] - 2020-03

The final release of the original C# / WinForms application (.NET Framework 4.8).

- Minor code tidy-up.
- Minor bug fixes.
- Slight GUI design change.
