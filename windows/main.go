//go:build windows

package main

import (
	_ "embed"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

// Tray icons: the color flame emoji (U+1F525), matching the flame next to
// "Hotfix Settings" on the settings page. Regenerate via gen-icons.ps1.
//
//go:embed assets/flame16.png
var flamePNG16 []byte

//go:embed assets/flame32.png
var flamePNG32 []byte

// Embed the app icon + version metadata into the .exe. Generates resource.syso
// from versioninfo.json (which points at assets/Hotfix.ico); `go build` then
// links it automatically. Run `go generate ./...` after editing versioninfo.json.
//
//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest -64 -o resource.syso versioninfo.json

// --- Tray state ---
var (
	mStatus       *systray.MenuItem
	mToggle       *systray.MenuItem
	trayMu        sync.Mutex
	statusTimeout *time.Timer
)

func main() {
	// Set up logging before anything else.
	initLog()
	logf("Hotfix starting (version %s)", currentVersion)

	// If a previous run left a crash behind, surface it for one-click reporting.
	reportPendingCrash()

	// Load config.
	configMu.Lock()
	current = loadConfig()
	configMu.Unlock()

	// Hand control to systray — onReady and onExit run on its goroutine.
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set tray icon and tooltip.
	systray.SetIcon(iconBytes())
	systray.SetTooltip("Hotfix — by BuildCraft Labs")

	// Build menu.
	mStatus = systray.AddMenuItem("Hotfix — Watching", "")
	mStatus.Disable()
	systray.AddSeparator()

	mToggle = systray.AddMenuItem("Enable Monitoring", "")
	syncToggleCheck()
	systray.AddSeparator()

	mSettings := systray.AddMenuItem("Settings...", "Open Hotfix settings")
	mUpdate := systray.AddMenuItem("Check for Updates", "Check GitHub for a newer release")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Hotfix")

	// Start monitor if enabled.
	if getConfig().Enabled {
		startMonitor()
	}

	// Watch for system sleep events (KillOnSleep support).
	watchSleep()

	// Begin silent background auto-updates (launch check + periodic poll).
	startAutoUpdater()

	// Event loop. A panic while handling a click is captured and reported
	// rather than silently killing the tray (panics go to a discarded stderr
	// under -H windowsgui). Each action is wrapped so one bad handler can't
	// take down the loop.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				recordCrash("event-loop", r)
				reportPendingCrash()
			}
		}()
		for {
			select {
			case <-mToggle.ClickedCh:
				safe("toggle", toggleEnabled)

			case <-mSettings.ClickedCh:
				safeGo("settings", openSettings)

			case <-mUpdate.ClickedCh:
				safeGo("update", func() { checkForUpdates(false) })

			case <-mQuit.ClickedCh:
				systray.Quit()
				return

			case <-systrayQuitCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	stopMonitor()
	terminateSettingsWindow()
	logf("Hotfix exiting")
	closeLog()
}

// toggleEnabled flips the Enabled flag in config and starts/stops monitoring.
func toggleEnabled() {
	cfg := getConfig()
	cfg.Enabled = !cfg.Enabled
	if err := setConfig(cfg); err != nil {
		logf("main: save config error: %v", err)
	}

	if cfg.Enabled {
		startMonitor()
		setTrayStatus("Watching", false)
	} else {
		stopMonitor()
		setTrayStatus("Disabled", false)
	}

	syncToggleCheck()
}

// syncToggleCheck updates the checkmark on the toggle menu item to reflect config.
func syncToggleCheck() {
	if getConfig().Enabled {
		mToggle.Check()
	} else {
		mToggle.Uncheck()
	}
}

// notifyKilled is called from monitor.go after a successful taskkill.
// It shows a desktop toast and updates the tray tooltip and status label.
func notifyKilled(name string, pid int, cpu float64) {
	// Desktop toast notification (non-blocking).
	notifyKilledToast(name, pid, cpu)

	msg := fmt.Sprintf("Killed: %s", name)
	tooltip := fmt.Sprintf("Hotfix — Killed %s (PID %d, %.0f%% CPU)", name, pid, cpu)

	setTrayStatus(msg, true)
	systray.SetTooltip(tooltip)

	trayMu.Lock()
	if statusTimeout != nil {
		statusTimeout.Stop()
	}
	statusTimeout = time.AfterFunc(3*time.Second, func() {
		setTrayStatus("Watching", false)
		systray.SetTooltip("Hotfix — by BuildCraft Labs")
	})
	trayMu.Unlock()
}

// setTrayStatus updates the disabled status label at the top of the menu.
func setTrayStatus(label string, alert bool) {
	prefix := "Hotfix — "
	if alert {
		prefix = "🔥 Hotfix — "
	}
	mStatus.SetTitle(prefix + label)
}

// --- Tray icon ---

// iconBytes returns an ICO containing the 16x16 and 32x32 color flame PNGs
// (the U+1F525 emoji), matching the settings-page flame.
func iconBytes() []byte {
	return pngToICO(
		pngFrame{size: 16, data: flamePNG16},
		pngFrame{size: 32, data: flamePNG32},
	)
}

type pngFrame struct {
	size int
	data []byte
}

// pngToICO wraps one or more PNG images into a valid ICO container.
func pngToICO(frames ...pngFrame) []byte {
	var b bytes.Buffer
	le := binary.LittleEndian

	n := uint16(len(frames))
	// ICO header: reserved=0, type=1 (ICO), count=n
	binary.Write(&b, le, uint16(0))
	binary.Write(&b, le, uint16(1))
	binary.Write(&b, le, n)

	// Each ICONDIRENTRY is 16 bytes; image data follows all entries.
	offset := uint32(6 + 16*int(n))
	for _, f := range frames {
		sz := byte(f.size) // 0 means 256 for the ICO spec; 32→32
		b.WriteByte(sz)    // width
		b.WriteByte(sz)    // height
		b.WriteByte(0)     // colorCount (0 = no palette)
		b.WriteByte(0)     // reserved
		binary.Write(&b, le, uint16(1))           // planes
		binary.Write(&b, le, uint16(32))          // bitCount
		binary.Write(&b, le, uint32(len(f.data))) // imageSize
		binary.Write(&b, le, offset)              // offset
		offset += uint32(len(f.data))
	}

	for _, f := range frames {
		b.Write(f.data)
	}

	return b.Bytes()
}


// --- Logging ---
var (
	logFile *os.File
	logMu   sync.Mutex
)

// logFilePath returns the absolute path to the log file, or "" if it cannot be
// determined. Used by both initLog and the settings server's /log endpoint.
func logFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Hotfix", "hotfix.log")
}

func initLog() {
	path := logFilePath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	logFile = f
}

func logf(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	msg := fmt.Sprintf("[%s] "+format+"\n",
		append([]any{time.Now().Format("2006-01-02 15:04:05")}, args...)...)
	if logFile != nil {
		_, _ = logFile.WriteString(msg)
	}
	// In debug builds you can uncomment the line below:
	// os.Stderr.WriteString(msg)
}

func closeLog() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		_ = logFile.Close()
	}
}
