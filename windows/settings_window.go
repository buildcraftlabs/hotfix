//go:build windows

package main

import (
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/gonutz/wui/v2"
)

var (
	settingsMu  sync.Mutex
	settingsWin *wui.Window
)

// showSettingsWindow opens a native Win32 settings panel near the taskbar.
// Subsequent calls while the window is open are no-ops.
func showSettingsWindow() {
	settingsMu.Lock()
	if settingsWin != nil {
		settingsMu.Unlock()
		return
	}
	settingsMu.Unlock()

	// Win32 windows must be created and driven from the same OS thread.
	runtime.LockOSThread()

	cfg := getConfig()

	const (
		px    = 20
		inner = 400
	)

	w := wui.NewWindow()
	w.SetTitle("Hotfix — Settings")
	w.SetInnerSize(inner, 440)
	w.SetResizable(false)
	w.SetHasMaxButton(false)

	y := 16

	// ── Enable Monitoring ──────────────────────────────────────────────────
	chkEnabled := wui.NewCheckBox()
	chkEnabled.SetBounds(px, y, inner-2*px, 26)
	chkEnabled.SetText("Enable Monitoring")
	chkEnabled.SetChecked(cfg.Enabled)
	w.Add(chkEnabled)
	y += 48

	// ── CPU Threshold ──────────────────────────────────────────────────────
	lblCPU := wui.NewLabel()
	lblCPU.SetBounds(px, y, 200, 20)
	lblCPU.SetText("CPU Threshold")
	lblCPUVal := wui.NewLabel()
	lblCPUVal.SetBounds(inner-px-56, y, 56, 20)
	lblCPUVal.SetText(strconv.Itoa(int(cfg.CPUThreshold)) + " %")
	w.Add(lblCPU)
	w.Add(lblCPUVal)
	y += 24

	sldCPU := wui.NewSlider()
	sldCPU.SetBounds(px, y, inner-2*px, 28)
	sldCPU.SetMin(1)
	sldCPU.SetMax(100)
	sldCPU.SetCursorPosition(int(cfg.CPUThreshold))
	sldCPU.SetOnChange(func(v int) {
		lblCPUVal.SetText(strconv.Itoa(v) + " %")
	})
	w.Add(sldCPU)
	y += 48

	// ── Kill After ─────────────────────────────────────────────────────────
	lblKill := wui.NewLabel()
	lblKill.SetBounds(px, y, 200, 20)
	lblKill.SetText("Kill After")
	lblKillVal := wui.NewLabel()
	lblKillVal.SetBounds(inner-px-56, y, 56, 20)
	lblKillVal.SetText(strconv.Itoa(int(cfg.KillDuration)) + " s")
	w.Add(lblKill)
	w.Add(lblKillVal)
	y += 24

	sldKill := wui.NewSlider()
	sldKill.SetBounds(px, y, inner-2*px, 28)
	sldKill.SetMin(5)
	sldKill.SetMax(300)
	sldKill.SetCursorPosition(int(cfg.KillDuration))
	sldKill.SetOnChange(func(v int) {
		lblKillVal.SetText(strconv.Itoa(v) + " s")
	})
	w.Add(sldKill)
	y += 48

	// ── Kill on Sleep ──────────────────────────────────────────────────────
	chkSleep := wui.NewCheckBox()
	chkSleep.SetBounds(px, y, inner-2*px, 26)
	chkSleep.SetText("Kill on Sleep")
	chkSleep.SetChecked(cfg.KillOnSleep)
	w.Add(chkSleep)
	y += 48

	// ── Exclusions ─────────────────────────────────────────────────────────
	lblEx := wui.NewLabel()
	lblEx.SetBounds(px, y, inner-2*px, 20)
	lblEx.SetText("Exclusions (comma-separated):")
	w.Add(lblEx)
	y += 26

	editEx := wui.NewEditLine()
	editEx.SetBounds(px, y, inner-2*px, 26)
	editEx.SetText(strings.Join(cfg.Whitelist, ", "))
	w.Add(editEx)
	y += 46

	// ── Save button ────────────────────────────────────────────────────────
	btnSave := wui.NewButton()
	btnSave.SetBounds(px, y, inner-2*px, 36)
	btnSave.SetText("Save Settings")
	btnSave.SetOnClick(func() {
		prevEnabled := cfg.Enabled

		newCfg := getConfig()
		newCfg.Enabled = chkEnabled.Checked()
		newCfg.CPUThreshold = float64(sldCPU.CursorPosition())
		newCfg.KillDuration = float64(sldKill.CursorPosition())
		newCfg.KillOnSleep = chkSleep.Checked()

		parts := strings.Split(editEx.Text(), ",")
		wl := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				wl = append(wl, p)
			}
		}
		newCfg.Whitelist = wl

		if err := setConfig(newCfg); err != nil {
			logf("settings: save error: %v", err)
		} else {
			if newCfg.Enabled && !prevEnabled {
				startMonitor()
			} else if !newCfg.Enabled && prevEnabled {
				stopMonitor()
			}
			syncToggleCheck()
			logf("settings: saved (enabled=%v, cpu=%.0f%%, kill=%.0fs)",
				newCfg.Enabled, newCfg.CPUThreshold, newCfg.KillDuration)
		}
		w.Close()
	})
	w.Add(btnSave)

	settingsMu.Lock()
	settingsWin = w
	settingsMu.Unlock()

	w.Show() // blocks until window closes

	settingsMu.Lock()
	settingsWin = nil
	settingsMu.Unlock()
}
