//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_CPUThreshold(t *testing.T) {
	cfg := defaultConfig()
	if cfg.CPUThreshold != 80.0 {
		t.Errorf("default CPUThreshold: got %.1f, want 80.0", cfg.CPUThreshold)
	}
}

func TestDefaultConfig_KillDuration(t *testing.T) {
	cfg := defaultConfig()
	if cfg.KillDuration != 60.0 {
		t.Errorf("default KillDuration: got %.1f, want 60.0", cfg.KillDuration)
	}
}

func TestDefaultConfig_EnabledByDefault(t *testing.T) {
	cfg := defaultConfig()
	if !cfg.Enabled {
		t.Error("monitoring should be enabled by default")
	}
}

func TestDefaultConfig_KillOnSleepByDefault(t *testing.T) {
	cfg := defaultConfig()
	if !cfg.KillOnSleep {
		t.Error("kill-on-sleep should be on by default")
	}
}

func TestDefaultConfig_WhitelistNonEmpty(t *testing.T) {
	cfg := defaultConfig()
	if len(cfg.Whitelist) == 0 {
		t.Error("default whitelist should contain at least the core Windows processes")
	}
}

func TestDefaultConfig_WhitelistContainsExplorer(t *testing.T) {
	cfg := defaultConfig()
	for _, name := range cfg.Whitelist {
		if name == "explorer" {
			return
		}
	}
	t.Error("default whitelist must include 'explorer'")
}

func TestLoadConfig_FallsBackToDefaultsOnMissingFile(t *testing.T) {
	// Point configPath to a temp dir with no config file.
	dir := t.TempDir()
	origPath := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	defer os.Setenv("APPDATA", origPath)

	cfg := loadConfig()

	// Should get defaults, not zero values.
	if cfg.CPUThreshold == 0 {
		t.Error("loadConfig should return defaults when config file is missing, got CPUThreshold=0")
	}
	if !cfg.Enabled {
		t.Error("loadConfig should default to enabled=true when file is missing")
	}
}

func TestSetAndLoadConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	origPath := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	defer os.Setenv("APPDATA", origPath)

	// Ensure the Hotfix subdir exists (mimicking real config path).
	_ = os.MkdirAll(filepath.Join(dir, "Hotfix"), 0755)

	want := Config{
		Enabled:      false,
		CPUThreshold: 75.0,
		KillDuration: 90.0,
		KillOnSleep:  false,
		Whitelist:    []string{"steam", "ffmpeg"},
	}

	if err := setConfig(want); err != nil {
		t.Fatalf("setConfig: %v", err)
	}

	got := loadConfig()

	if got.Enabled != want.Enabled {
		t.Errorf("Enabled: got %v, want %v", got.Enabled, want.Enabled)
	}
	if got.CPUThreshold != want.CPUThreshold {
		t.Errorf("CPUThreshold: got %.1f, want %.1f", got.CPUThreshold, want.CPUThreshold)
	}
	if got.KillDuration != want.KillDuration {
		t.Errorf("KillDuration: got %.1f, want %.1f", got.KillDuration, want.KillDuration)
	}
	if got.KillOnSleep != want.KillOnSleep {
		t.Errorf("KillOnSleep: got %v, want %v", got.KillOnSleep, want.KillOnSleep)
	}
	if len(got.Whitelist) != len(want.Whitelist) {
		t.Errorf("Whitelist length: got %d, want %d", len(got.Whitelist), len(want.Whitelist))
	}
}
