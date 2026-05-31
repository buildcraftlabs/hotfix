//go:build windows

package main

import (
	"testing"
)

// Sample WMIC CSV output (realistic format Windows produces).
const sampleWMICOutput = `
Node,IDProcess,Name,PercentProcessorTime

DESKTOP,0,Idle,2
DESKTOP,4,System,0
DESKTOP,1234,claude-desktop,94
DESKTOP,5678,node,81
DESKTOP,9999,_Total,177
`

func TestParseWMICCSV_ParsesProcesses(t *testing.T) {
	entries, err := parseWMICCSV(sampleWMICOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one process entry, got none")
	}
}

func TestParseWMICCSV_SkipsTotalRow(t *testing.T) {
	entries, err := parseWMICCSV(sampleWMICOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range entries {
		if e.Name == "_Total" {
			t.Error("_Total aggregate row should be excluded from results")
		}
	}
}

func TestParseWMICCSV_ParsesNameAndCPU(t *testing.T) {
	entries, err := parseWMICCSV(sampleWMICOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := map[string]float64{}
	for _, e := range entries {
		found[e.Name] = e.CPU
	}

	if cpu, ok := found["claude-desktop"]; !ok {
		t.Error("expected claude-desktop in results")
	} else if cpu != 94.0 {
		t.Errorf("claude-desktop CPU: got %.1f, want 94.0", cpu)
	}

	if cpu, ok := found["node"]; !ok {
		t.Error("expected node in results")
	} else if cpu != 81.0 {
		t.Errorf("node CPU: got %.1f, want 81.0", cpu)
	}
}

func TestParseWMICCSV_ParsesPID(t *testing.T) {
	entries, err := parseWMICCSV(sampleWMICOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range entries {
		if e.Name == "claude-desktop" {
			if e.PID != 1234 {
				t.Errorf("claude-desktop PID: got %d, want 1234", e.PID)
			}
			return
		}
	}
	t.Error("claude-desktop not found")
}

func TestParseWMICCSV_EmptyOutput(t *testing.T) {
	_, err := parseWMICCSV("")
	// Empty output has no header — expect an error, not a panic.
	if err == nil {
		t.Error("expected an error for empty WMIC output, got nil")
	}
}

func TestParseWMICCSV_NoHeader(t *testing.T) {
	_, err := parseWMICCSV("DESKTOP,1234,foo,80\n")
	if err == nil {
		t.Error("expected an error when header row is missing")
	}
}

func TestParseWMICCSV_StripsExeSuffix(t *testing.T) {
	const withExe = `
Node,IDProcess,Name,PercentProcessorTime

DESKTOP,100,chrome.exe,55
`
	entries, err := parseWMICCSV(withExe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range entries {
		if e.Name == "chrome.exe" {
			t.Error(".exe suffix should be stripped from process names")
		}
		if e.Name == "chrome" {
			return // pass
		}
	}
	t.Error("expected 'chrome' after suffix stripping")
}

func TestParseWMICCSV_ZeroCPUIncluded(t *testing.T) {
	entries, err := parseWMICCSV(sampleWMICOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// System at 0% CPU should still appear in results.
	for _, e := range entries {
		if e.Name == "System" && e.CPU == 0 {
			return
		}
	}
	t.Error("expected System process with 0% CPU to be included")
}

func TestSafetyExclusions_ContainsKnownDangerousTargets(t *testing.T) {
	must := []string{"explorer", "svchost", "lsass", "winlogon", "dwm"}
	for _, name := range must {
		if !safetyExclusions[name] {
			t.Errorf("safetyExclusions must contain %q", name)
		}
	}
}

func TestSafetyExclusions_IsNonEmpty(t *testing.T) {
	if len(safetyExclusions) < 5 {
		t.Errorf("safetyExclusions has only %d entries, expected at least 5", len(safetyExclusions))
	}
}
