//go:build windows

package main

import "testing"

// Real WMIC output uses CRLF line endings — make sure parsing handles them.
func TestParseWMICCSV_HandlesCRLF(t *testing.T) {
	raw := "\r\nNode,IDProcess,Name,PercentProcessorTime\r\n\r\nDESKTOP,1234,chrome,90\r\nDESKTOP,0,_Total,90\r\n"
	entries, err := parseWMICCSV(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Name == "chrome" && e.PID == 1234 && e.CPU == 90 {
			found = true
		}
	}
	if !found {
		t.Errorf("CRLF output not parsed correctly; got %+v", entries)
	}
}

// WMIC sorts requested columns alphabetically, so real-world column order can
// differ from the order requested. parseWMICCSV maps by header name, so a
// reordered header must still parse.
func TestParseWMICCSV_HandlesReorderedColumns(t *testing.T) {
	raw := "\nNode,Name,PercentProcessorTime,IDProcess\n\nDESKTOP,node,81,5678\n"
	entries, err := parseWMICCSV(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Name != "node" || e.PID != 5678 || e.CPU != 81 {
		t.Errorf("reordered columns parsed wrong: %+v", e)
	}
}

func TestParseWMICCSV_SkipsNegativeAndNonNumericPID(t *testing.T) {
	raw := "\nNode,IDProcess,Name,PercentProcessorTime\n\nDESKTOP,-1,bad,90\nDESKTOP,abc,worse,90\nDESKTOP,77,good,90\n"
	entries, err := parseWMICCSV(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range entries {
		if e.Name == "bad" || e.Name == "worse" {
			t.Errorf("process with invalid PID should be skipped, got %+v", e)
		}
	}
	if len(entries) != 1 || entries[0].Name != "good" {
		t.Errorf("expected only 'good' to survive, got %+v", entries)
	}
}

func TestParseWMICCSV_SkipsNonNumericCPU(t *testing.T) {
	raw := "\nNode,IDProcess,Name,PercentProcessorTime\n\nDESKTOP,100,weird,N/A\nDESKTOP,101,ok,50\n"
	entries, err := parseWMICCSV(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range entries {
		if e.Name == "weird" {
			t.Error("process with non-numeric CPU should be skipped")
		}
	}
}

// A process instance with a perf-counter "#N" suffix (e.g. svchost#3) must
// still be recognized as a protected system process. WMIC's PerfProc class
// emits these suffixes for duplicate image names.
func TestSafetyExclusions_ProtectsInstanceSuffixedSystemProcs(t *testing.T) {
	if isProtected("svchost") != true {
		t.Fatal("sanity: bare svchost should be protected")
	}
	if !isProtected("svchost#3") {
		t.Error("svchost#3 (a perf-counter instance of svchost) must be protected, but isProtected returned false")
	}
}
