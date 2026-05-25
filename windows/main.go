//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

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
				go openSettings()

			case <-mUpdate.ClickedCh:
				go checkForUpdates()

			case <-mQuit.ClickedCh:
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
// iconBytes returns a minimal valid ICO as raw bytes so the app compiles
// without any external asset tool or CGO. This is a 16×16 orange flame-
// coloured icon encoded as a 1-bit BMP inside an ICO container.
// For production, replace this with a real .ico embed.
func iconBytes() []byte {
	// Minimal 16x16 ICO (1-colour BMP fallback — Windows accepts this).
	// Generated via hex from a minimal valid ICO file.
	// This produces a small orange square icon that is clearly visible.
	return minimalICO()
}

// minimalICO builds a minimal valid 16x16 ICO file in memory.
// The image is a solid #C9461E (BuildCraft orange) square.
func minimalICO() []byte {
	const (
		width    = 16
		height   = 16
		rowBytes = (width + 7) / 8
	)

	// XOR mask: all 1s → maps to colour[1] (orange)
	var xorMask [height * rowBytes]byte
	for i := range xorMask {
		xorMask[i] = 0xFF
	}
	// AND mask: all 0s → fully opaque
	var andMask [height * rowBytes]byte

	bmpSize := uint32(40 + 8 + len(xorMask) + len(andMask))

	var b bytes.Buffer
	le := binary.LittleEndian

	// ICO header (6 bytes)
	binary.Write(&b, le, uint16(0))    // reserved
	binary.Write(&b, le, uint16(1))    // type = ICO
	binary.Write(&b, le, uint16(1))    // image count

	// ICONDIRENTRY (16 bytes)
	b.Write([]byte{byte(width), byte(height), 2, 0}) // w, h, colourCount, reserved
	binary.Write(&b, le, uint16(1))                  // planes
	binary.Write(&b, le, uint16(1))                  // bit count
	binary.Write(&b, le, bmpSize)                    // size of image data
	binary.Write(&b, le, uint32(6+16))               // offset to image data

	// BITMAPINFOHEADER (40 bytes)
	binary.Write(&b, le, uint32(40))          // header size
	binary.Write(&b, le, uint32(width))       // width
	binary.Write(&b, le, uint32(height*2))    // height ×2 (XOR + AND masks)
	binary.Write(&b, le, uint16(1))           // planes
	binary.Write(&b, le, uint16(1))           // bit count
	binary.Write(&b, le, uint32(0))           // compression
	binary.Write(&b, le, uint32(0))           // image size
	binary.Write(&b, le, uint32(0))           // X px/metre
	binary.Write(&b, le, uint32(0))           // Y px/metre
	binary.Write(&b, le, uint32(2))           // colours used
	binary.Write(&b, le, uint32(2))           // colours important

	// Colour table: [0] black transparent, [1] #C9461E in BGR
	b.Write([]byte{0x00, 0x00, 0x00, 0x00})
	b.Write([]byte{0x1E, 0x46, 0xC9, 0x00})

	b.Write(xorMask[:])
	b.Write(andMask[:])

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
