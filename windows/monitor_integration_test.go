//go:build windows

package main

import (
	"os/exec"
	"testing"
)

// TestQueryProcesses_Integration exercises the real performance-counter query
// path (PowerShell CIM) against the live machine. This is the end-to-end check
// that would have caught the removed-WMIC regression: the old `wmic` backend
// returns "executable file not found" on Windows 11 24H2+ and yields zero
// processes. Skipped automatically where PowerShell is unavailable.
func TestQueryProcesses_Integration(t *testing.T) {
	if _, err := exec.LookPath("powershell"); err != nil {
		t.Skip("powershell not available; skipping live query test")
	}

	entries, err := queryProcesses()
	if err != nil {
		t.Fatalf("queryProcesses returned error: %v", err)
	}
	// Any running Windows machine has well over a handful of processes.
	if len(entries) < 5 {
		t.Fatalf("expected many processes from a live query, got %d", len(entries))
	}

	// Every returned entry should be well-formed: a non-negative PID (the Idle
	// process legitimately reports PID 0; checkProcesses filters PID<10 later)
	// and a non-empty name.
	for _, e := range entries {
		if e.PID < 0 {
			t.Errorf("entry has negative PID: %+v", e)
		}
		if e.Name == "" {
			t.Errorf("entry has empty name: %+v", e)
		}
		if e.Name == "_Total" {
			t.Errorf("_Total aggregate should have been filtered out")
		}
	}
}
