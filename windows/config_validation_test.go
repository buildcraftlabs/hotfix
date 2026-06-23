//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfigJSON points configPath at a temp APPDATA and writes raw JSON there.
// Returns a cleanup func that restores APPDATA.
func writeConfigJSON(t *testing.T, raw string) func() {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	if err := os.MkdirAll(filepath.Join(dir, "Hotfix"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Hotfix", "config.json"), []byte(raw), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return func() { os.Setenv("APPDATA", orig) }
}

func TestLoadConfig_ClampsCPUThresholdTooHigh(t *testing.T) {
	defer writeConfigJSON(t, `{"enabled":true,"cpu_threshold":250,"kill_duration":60,"whitelist":["a"]}`)()
	cfg := loadConfig()
	if cfg.CPUThreshold != 80.0 {
		t.Errorf("CPUThreshold 250 should be clamped to default 80.0, got %.1f", cfg.CPUThreshold)
	}
}

func TestLoadConfig_ClampsCPUThresholdTooLow(t *testing.T) {
	defer writeConfigJSON(t, `{"enabled":true,"cpu_threshold":0,"kill_duration":60,"whitelist":["a"]}`)()
	cfg := loadConfig()
	if cfg.CPUThreshold != 80.0 {
		t.Errorf("CPUThreshold 0 should be clamped to default 80.0, got %.1f", cfg.CPUThreshold)
	}
}

func TestLoadConfig_ClampsKillDurationTooLow(t *testing.T) {
	defer writeConfigJSON(t, `{"enabled":true,"cpu_threshold":80,"kill_duration":2,"whitelist":["a"]}`)()
	cfg := loadConfig()
	if cfg.KillDuration != 60.0 {
		t.Errorf("KillDuration 2 should be clamped to default 60.0, got %.1f", cfg.KillDuration)
	}
}

func TestLoadConfig_NilWhitelistBecomesDefault(t *testing.T) {
	defer writeConfigJSON(t, `{"enabled":true,"cpu_threshold":80,"kill_duration":60}`)()
	cfg := loadConfig()
	if len(cfg.Whitelist) == 0 {
		t.Error("missing whitelist should fall back to the default whitelist, got empty")
	}
}

func TestLoadConfig_CorruptJSONReturnsDefaults(t *testing.T) {
	defer writeConfigJSON(t, `{this is not valid json`)()
	cfg := loadConfig()
	if cfg.CPUThreshold != 80.0 || cfg.KillDuration != 60.0 || !cfg.Enabled {
		t.Errorf("corrupt JSON should yield defaults, got %+v", cfg)
	}
}

func TestLoadConfig_ValidValuesPreserved(t *testing.T) {
	defer writeConfigJSON(t, `{"enabled":false,"cpu_threshold":65,"kill_duration":120,"kill_on_sleep":false,"whitelist":["steam"]}`)()
	cfg := loadConfig()
	if cfg.Enabled {
		t.Error("enabled=false should be preserved")
	}
	if cfg.CPUThreshold != 65 {
		t.Errorf("valid CPUThreshold 65 should be preserved, got %.1f", cfg.CPUThreshold)
	}
	if cfg.KillDuration != 120 {
		t.Errorf("valid KillDuration 120 should be preserved, got %.1f", cfg.KillDuration)
	}
	if cfg.KillOnSleep {
		t.Error("kill_on_sleep=false should be preserved")
	}
	if len(cfg.Whitelist) != 1 || cfg.Whitelist[0] != "steam" {
		t.Errorf("whitelist should be preserved, got %v", cfg.Whitelist)
	}
}
