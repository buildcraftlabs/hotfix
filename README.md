# Hotfix

**Keep your Mac cool.** Hotfix is a lightweight macOS menu bar app that monitors CPU usage and automatically terminates runaway processes before your fan spins up and your battery drains.

Built by [BuildCraft Labs](https://github.com/buildcraftlabs).

---

## The Problem

Claude extensions, AI tools, and background daemons increasingly run rogue on macOS — consuming 50–100% CPU for hours, spinning up fans, draining batteries, and generating heat with no visible indication anything is wrong.

## How It Works

1. **Monitor** — Hotfix polls all running processes every 5 seconds
2. **Detect** — Any process sustaining CPU usage above your configured threshold gets flagged
3. **Kill** — After your configured duration, it terminates the offender with SIGTERM and notifies you

## Features

- **Menu bar native** — Lives in your menu bar, zero Dock presence
- **Configurable threshold** — Set the CPU % that triggers monitoring (default: 80%)
- **Configurable duration** — How long a process must be hot before being killed (default: 60s)
- **Kill on sleep** — Optionally terminate hot processes when your Mac sleeps
- **Exclusion list** — Protect specific processes from ever being killed (Xcode, node, etc.)
- **Auto-updates** — Checks GitHub releases for new versions on launch
- **Safety exclusions** — System-critical processes (WindowServer, launchd, Finder) are permanently protected

## Download

Download the latest DMG from [Releases](https://github.com/buildcraftlabs/hotfix/releases/latest).

> **Note:** Hotfix is not yet notarized. On first open, right-click the app → **Open** to bypass Gatekeeper.

## Requirements

- macOS 13 (Ventura) or later
- Apple Silicon or Intel

## Build from Source

Requires Xcode Command Line Tools.

```bash
git clone https://github.com/buildcraftlabs/hotfix.git
cd hotfix
bash scripts/build.sh
open "dist/Hotfix.dmg"
```

The build script produces a universal binary (arm64 + x86_64), assembles the `.app` bundle, ad-hoc signs it, and packages it into a DMG.

## Releasing a New Version

1. Update `currentVersion` in `Sources/Hotfix/UpdateChecker.swift`
2. Update `CFBundleShortVersionString` in `Resources/Info.plist`
3. Run `bash scripts/build.sh`
4. Create a GitHub release tagged `v<version>` and attach `dist/Hotfix.dmg`

Users with Hotfix installed will be notified of the update on next launch.

## Configuration

All settings are accessible from the menu bar popover → **Settings**.

| Setting | Default | Description |
|---------|---------|-------------|
| CPU Threshold | 80% | Processes above this level are monitored |
| Kill After | 60s | Duration before termination |
| Kill on Sleep | On | Kill hot processes when Mac sleeps |
| Exclusions | Xcode, swift, clang, node, python3 | Processes never killed |

## License

MIT © [BuildCraft Labs](https://github.com/buildcraftlabs)
