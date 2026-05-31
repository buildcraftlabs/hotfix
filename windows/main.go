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

//go:embed assets/icon.png
var iconPNG32 []byte

//go:embed assets/icon16.png
var iconPNG16 []byte

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

	// Load config.
	configMu.Lock()
	current = loadConfig()
	configMu.Unlock()

	// Pre-start the HTTP server so it's ready instantly.
	startServer()

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

	mSettings := systray.AddMenuItem("Settings...", "Open settings in browser")
	mUpdate := systray.AddMenuItem("Check for Updates", "Check GitHub for a newer release")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Hotfix")

	// Start monitor if enabled.
	if getConfig().Enabled {
		startMonitor()
	}

	// Watch for system sleep events (KillOnSleep support).
	watchSleep()

	// Event loop.
	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				toggleEnabled()

			case <-mSettings.ClickedCh:
				go openSettingsPage()

			case <-mUpdate.ClickedCh:
				go checkForUpdates()

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
// It updates the tray tooltip and status label temporarily.
func notifyKilled(name string, pid int, cpu float64) {
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

// iconBytes returns an ICO containing the 16x16 and 32x32 flame PNGs.
// Windows supports PNG frames inside ICO containers (Vista+).
func iconBytes() []byte {
	return pngToICO(
		pngFrame{size: 16, data: iconPNG16},
		pngFrame{size: 32, data: iconPNG32},
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

func initLog() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(dir, "Hotfix")
	_ = os.MkdirAll(logDir, 0755)
	f, err := os.OpenFile(filepath.Join(logDir, "hotfix.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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
