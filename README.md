# Stationeers Modding Installer

[![Download convenience Installer](https://img.shields.io/badge/Download-convenience%20Installer-ff8300?style=for-the-badge&logo=windows&logoColor=white)](https://github.com/StationeersLaunchPad/StationeersModdingInstaller/releases/latest/download/StationeersModdingInstaller.exe)

A single-file **_Windows_** wizard that **installs BepInEx** and **StationeersLaunchPad** into the Stationeers game folder. 
No .NET runtime, no dependencies - just run the exe and you are good to go. 
Requires WebView2 (default in modern windows distributions)
---

## What it does

1. Detects the Stationeers install folder automatically (Steam registry + common paths) or lets the user enter one manually.
<img width="817" height="492" alt="image" src="https://github.com/user-attachments/assets/992548d3-3f8a-447f-8962-c33d5cf5f61d" />
2. Prompts the user to confirm the install location is correct, offering to change it
<img width="817" height="492" alt="image" src="https://github.com/user-attachments/assets/43a0dc28-6308-4acd-872e-83340bd9b370" />
4. Validates that `rocketstation.exe` exists in the chosen folder.
5. Downloads **BepInEx 5.4.23.2** (win-x64) from GitHub, verifies its SHA-256 checksum, and extracts it into the game root.
<img width="817" height="492" alt="image" src="https://github.com/user-attachments/assets/e394b1c9-d96f-4b82-ad1c-5cf6a9bd80ca" />
7. Fetches the **latest stable Stationeers LaunchPad** client release from the GitHub Releases API and extracts it into `BepInEx/plugins/StationeersLaunchPad/`.
<img width="817" height="492" alt="image" src="https://github.com/user-attachments/assets/9553d0eb-7af7-4e75-a54a-5cd02d3df9df" />
8. Tells the user if the installation succeeded
<img width="817" height="492" alt="image" src="https://github.com/user-attachments/assets/bb948934-5c6d-48bc-b028-4ec9a20234a4" />

**No game files are modified beyond the mod loader setup. The installer can be re-run safely - it overwrites existing BepInEx/SLP files.**

---

## How it works

| Layer | Technology |
|---|---|
| UI | Vanilla HTML/CSS/JS (embedded in the exe, no server) |
| Backend | Go 1.22+ (1.26)|
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
