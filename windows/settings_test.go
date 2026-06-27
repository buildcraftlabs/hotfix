//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempAppData redirects config/log storage to a temp dir for a test.
func withTempAppData(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	_ = os.MkdirAll(filepath.Join(dir, "Hotfix"), 0755)
	t.Cleanup(func() { os.Setenv("APPDATA", orig) })
}

func monitorRunning() bool {
	monitorMu.Lock()
	defer monitorMu.Unlock()
	return monitorStop != nil
}

// MARK: - applySavedConfig (the settings popover's save binding)

func TestApplySavedConfig_PersistsValidConfig(t *testing.T) {
	withTempAppData(t)
	t.Cleanup(stopMonitor)

	cfg := Config{Enabled: false, CPUThreshold: 75, KillDuration: 90, KillOnSleep: true, Whitelist: []string{"steam"}}
	if err := applySavedConfig(cfg); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	got := getConfig()
	if got.CPUThreshold != 75 || got.KillDuration != 90 {
		t.Errorf("config not persisted: got %+v", got)
	}
}

func TestApplySavedConfig_RejectsLowCPUThreshold(t *testing.T) {
	withTempAppData(t)
	if err := applySavedConfig(Config{CPUThreshold: 0, KillDuration: 60}); err == nil {
		t.Error("cpu_threshold=0 should be rejected")
	}
}

func TestApplySavedConfig_RejectsHighCPUThreshold(t *testing.T) {
	withTempAppData(t)
	if err := applySavedConfig(Config{CPUThreshold: 250, KillDuration: 60}); err == nil {
		t.Error("cpu_threshold=250 should be rejected")
	}
}

func TestApplySavedConfig_RejectsLowKillDuration(t *testing.T) {
	withTempAppData(t)
	if err := applySavedConfig(Config{CPUThreshold: 80, KillDuration: 2}); err == nil {
		t.Error("kill_duration=2 should be rejected")
	}
}

func TestApplySavedConfig_NilWhitelistBecomesEmpty(t *testing.T) {
	withTempAppData(t)
	t.Cleanup(stopMonitor)
	if err := applySavedConfig(Config{CPUThreshold: 80, KillDuration: 60, Whitelist: nil}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wl := getConfig().Whitelist; wl == nil {
		t.Error("nil whitelist should be normalized to an empty slice")
	}
}

// Enabling monitoring from Settings must actually start the monitor (and
// disabling must stop it) — not just persist the flag. A high threshold + long
// kill duration guarantee the first poll never terminates a real process.
func TestApplySavedConfig_StartsAndStopsMonitor(t *testing.T) {
	withTempAppData(t)
	stopMonitor()
	t.Cleanup(stopMonitor)

	if monitorRunning() {
		t.Fatal("precondition: monitor should be stopped before the test")
	}

	if err := applySavedConfig(Config{Enabled: true, CPUThreshold: 95, KillDuration: 300}); err != nil {
		t.Fatalf("enable save failed: %v", err)
	}
	if !monitorRunning() {
		t.Error("monitor should be running after enabling monitoring")
	}

	if err := applySavedConfig(Config{Enabled: false, CPUThreshold: 95, KillDuration: 300}); err != nil {
		t.Fatalf("disable save failed: %v", err)
	}
	if monitorRunning() {
		t.Error("monitor should be stopped after disabling monitoring")
	}
}

// MARK: - readLogText

func TestReadLogText_PlaceholderWhenEmpty(t *testing.T) {
	withTempAppData(t)
	if got := readLogText(); got == "" {
		t.Error("readLogText should never return an empty string")
	}
}

// MARK: - embedded settings page

func TestSettingsHTML_Embedded(t *testing.T) {
	if !strings.Contains(settingsHTML, "Hotfix Settings") {
		t.Error("embedded settings page should contain the title 'Hotfix Settings'")
	}
	if strings.Contains(settingsHTML, "fetch('/") {
		t.Error("settings page should no longer use fetch() endpoints (use bound hotfix* funcs)")
	}
}
