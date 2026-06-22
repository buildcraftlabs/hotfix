# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Hotfix is a dual-platform (macOS + Windows) system tray app that monitors CPU usage and kills runaway processes before fans spin up. The macOS app is written in Swift/SwiftUI; the Windows app is written in Go.

## Build Commands

### macOS
```bash
# Full build → dist/Hotfix.app + dist/Hotfix.dmg
bash scripts/build.sh

# Swift compile only (no bundling)
swift build -c release --arch arm64 --arch x86_64

# Run directly from build output (no DMG needed)
open dist/Hotfix.app
```

### Windows
```powershell
cd windows
go build -ldflags "-H windowsgui -s -w" -o ..\dist\Hotfix.exe .
```

## Release Process

1. Bump the version everywhere: `Sources/Hotfix/UpdateChecker.swift` (`currentVersion`), `Resources/Info.plist` (`CFBundleShortVersionString` **and** bump `CFBundleVersion`), `windows/updater.go` (`currentVersion`), the hardcoded version label in `windows/assets/settings.html` (About card), and the **four download-button URLs in `docs/index.html`** (`Hotfix-v<version>-macOS.dmg` / `Hotfix-v<version>-Windows.exe`, in both the hero and the platform-cards section). `windows/main.go` only references `updater.go`'s `currentVersion`, so no literal there. (Version-comparison fixtures in `windows/updater_test.go` are not app versions — leave them.)
2. Create a GitHub release tagged `v<version>` — the `Build` workflow runs automatically on `macos-latest` and `windows-latest`. Each release gets **four** assets: stable `Hotfix.dmg` / `Hotfix.exe` **plus** version+OS-named `Hotfix-v<version>-macOS.dmg` / `Hotfix-v<version>-Windows.exe` (the site's `docs/index.html` download buttons point to these versioned names, so they 404 until the release exists and step 1's URLs match the new version).
3. A user-facing feature is **not shipped** until this release is cut and the Build run succeeds with all four assets attached — the website serves only released binaries.

> Logs: macOS → `~/Library/Logs/Hotfix/hotfix.log`; Windows → `%APPDATA%\Hotfix\hotfix.log`. Both surface in Settings via an in-app log viewer. Desktop notifications fire on every successful kill (macOS notification center; Windows WinRT toast).

## Architecture

### macOS (`Sources/Hotfix/`)

The app is a `MenuBarExtra`-only app (no Dock icon). Core objects are singletons shared via `@StateObject`:

- **`ProcessMonitor`** (`@MainActor` singleton) — 5-second `Timer` loop that reads process CPU via `ps`, tracks hot start times in a `[pid → TimeInterval]` dict, kills via `kill()`, and publishes `hotProcesses` / `isKilling` to the UI. Also listens for `NSWorkspace.willSleepNotification` to kill on sleep.
- **`PreferencesManager`** — Wraps `@AppStorage` for all settings (threshold, duration, kill-on-sleep, whitelist). Whitelist is stored as JSON in `UserDefaults`.
- **`UpdateChecker`** — Hits the GitHub releases API on launch; compares semver tags.
- **`SettingsWindowController`** / **`SettingsView`** — Native SwiftUI settings panel opened from the menu.
- **`MenuBarPopoverView`** — The popover shown when clicking the tray flame icon. Shows hot processes and quick toggles.
- **`Log`** (`Logger.swift`) — Thread-safe file logger (`logf("…")`) that appends to `~/Library/Logs/Hotfix/hotfix.log` (and stderr). Mirrors the Windows logger format. Desktop notifications use `UNUserNotificationCenter` (authorization requested at launch).

Safety exclusions (kernel_task, WindowServer, Finder, etc.) are hardcoded in `ProcessMonitor` and can never be overridden by user settings.

### Windows (`windows/`)

A single Go binary with `//go:build windows` on every file. No CGO; uses `github.com/getlantern/systray` for the system tray and `github.com/gonutz/wui/v2` for the native Win32 settings window.

- **`main.go`** — Entry point: init logging, load config, start HTTP server, hand control to `systray.Run`.
- **`monitor.go`** — 5-second poll loop using `wmic` CSV output. Tracks hot processes in `hotMap`, calls `taskkill /F` when threshold exceeded. Sleep detection via a PowerShell WMI event subscription.
- **`config.go`** — Reads/writes JSON config from `%APPDATA%\Hotfix\config.json`. Thread-safe via `sync.RWMutex`.
- **`settings_window.go`** — Opens a native Win32 window (`wui`) for settings. Must be called and driven from the same OS thread (`runtime.LockOSThread`).
- **`server.go`** — Local HTTP server on a random port (`127.0.0.1:0`) serving the embedded `assets/settings.html` and a `/config` + `/save` JSON API. Used as a fallback settings UI.
- **`updater.go`** — GitHub releases API check; self-updates by downloading the new `.exe` to a temp path and launching it with a replace-and-restart batch script.
- **`notify.go`** — Desktop toast notifications via hidden PowerShell (WinRT `Windows.UI.Notifications` toast, with a `NotifyIcon` balloon-tip fallback). Title/body are passed through env vars to avoid quoting/injection. Called from `notifyKilled` in addition to the tray-label update. File logging is handled by `initLog`/`logf` in `main.go` (writes to `%APPDATA%\Hotfix\hotfix.log`).

All console-spawning child processes (`wmic`, `taskkill`, `powershell`) use `HideWindow: true` in `SysProcAttr` to prevent flash windows (since the binary is built with `-H windowsgui`).

## Key Constraints

- **Tests** — Go unit tests live in `windows/*_test.go` (build-tagged `//go:build windows`). They run on the Windows CI runner via `go test ./...`. Swift tests run via `swift test` on the macOS runner. There is no way to execute the Windows tests locally on macOS.
- **Version must be bumped in multiple files** — forgetting one will cause the update checker to behave incorrectly or CI to produce a mismatched binary.
- macOS binary is **not notarized**; users must right-click → Open on first launch.
- Windows build sets `-H windowsgui`, so `fmt.Print` / `log` output goes nowhere — use the file logger (`initLog` / `logf`).
