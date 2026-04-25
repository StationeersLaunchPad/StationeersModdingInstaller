# Stationeers Modding Installer

A single-file Windows wizard that installs **BepInEx** and **Stationeers LaunchPad** into a Stationeers game folder. No .NET runtime, no dependencies - just run the exe.

---

## What it does

1. Detects the Stationeers install folder automatically (Steam registry + common paths) or lets the user enter one manually.
2. Validates that `rocketstation.exe` exists in the chosen folder.
3. Downloads **BepInEx 5.4.23.2** (win-x64) from GitHub, verifies its SHA-256 checksum, and extracts it into the game root.
4. Fetches the **latest stable Stationeers LaunchPad** client release from the GitHub Releases API and extracts it into `BepInEx/plugins/StationeersLaunchPad/`.
5. Shows real-time progress and log output for each step.

No game files are modified beyond the mod loader setup. The installer can be re-run safely - it overwrites existing BepInEx/SLP files.

---

## How it works

| Layer | Technology |
|---|---|
| UI | Vanilla HTML/CSS/JS (embedded in the exe, no server) |
| Backend | Go 1.22+ |
| Desktop shell | [Wails v2](https://wails.io) (WebView2) |
| Output | Single `.exe`, ~10 MB, no installer required |

### Key packages

```
main.go                         Wails entry point, window config (1100×530)
app.go                          Wails-bound methods called by the frontend
internal/install/installer.go   Core install logic - download, verify, extract
internal/install/checksum.go    SHA-256 verify + sidecar/manifest resolver
internal/steamdetect/           Windows registry Steam path detection
frontend/index.html             Full wizard UI (all slides, CSS, JS in one file)
assets/                         App icon + banner image
```

### Trust model

- **BepInEx** - pinned to a specific version with a hardcoded SHA-256. Any tampered download will be rejected.
- **SLP** - no pinned checksum. Trust anchor is GitHub TLS + the GitHub Releases API. The installer always picks the newest non-prerelease asset matching `StationeersLaunchPad-client-v*.zip`.

---

## Building

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Windows (WebView2 is present on Windows 10 1803+ by default)

### Build command

```powershell
# From the repo root
Copy-Item "assets\iconpng.png" "build\appicon.png" -Force
wails build
```

Output: `build\bin\StationeersModdingInstaller.exe`

> If `wails` is not in your PATH, prefix with `$env:PATH += ";$env:USERPROFILE\go\bin"`.

### Dev mode (live reload)

```powershell
wails dev
```

Opens a browser-backed window with hot reload on frontend changes. Go changes require a restart.

---

## Code signing

The exe is currently unsigned. Windows SmartScreen will show an "Unknown publisher" warning on first run.

For open-source projects, [SignPath Foundation](https://about.signpath.io/product/open-source) offers free EV-grade signing via GitHub Actions - worth looking into.