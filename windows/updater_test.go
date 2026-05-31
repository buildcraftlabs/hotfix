//go:build windows

package main

import "testing"

// MARK: - isNewer

func TestIsNewer_NewerPatch(t *testing.T) {
	if !isNewer("1.0.4", "1.0.3") {
		t.Error("1.0.4 should be newer than 1.0.3")
	}
}

func TestIsNewer_NewerMinor(t *testing.T) {
	if !isNewer("1.1.0", "1.0.3") {
		t.Error("1.1.0 should be newer than 1.0.3")
	}
}

func TestIsNewer_NewerMajor(t *testing.T) {
	if !isNewer("2.0.0", "1.9.9") {
		t.Error("2.0.0 should be newer than 1.9.9")
	}
}

func TestIsNewer_SameVersion(t *testing.T) {
	if isNewer("1.0.3", "1.0.3") {
		t.Error("1.0.3 should NOT be newer than 1.0.3")
	}
}

func TestIsNewer_OlderPatch(t *testing.T) {
	if isNewer("1.0.2", "1.0.3") {
		t.Error("1.0.2 should NOT be newer than 1.0.3")
	}
}

func TestIsNewer_OlderMinor(t *testing.T) {
	if isNewer("1.0.9", "1.1.0") {
		t.Error("1.0.9 should NOT be newer than 1.1.0")
	}
}

func TestIsNewer_OlderMajor(t *testing.T) {
	if isNewer("1.9.9", "2.0.0") {
		t.Error("1.9.9 should NOT be newer than 2.0.0")
	}
}

// MARK: - parseSemver

func TestParseSemver_Standard(t *testing.T) {
	got := parseSemver("1.2.3")
	want := [3]int{1, 2, 3}
	if got != want {
		t.Errorf("parseSemver(1.2.3): got %v, want %v", got, want)
	}
}

func TestParseSemver_MissingPatch(t *testing.T) {
	got := parseSemver("1.2")
	if got[0] != 1 || got[1] != 2 || got[2] != 0 {
		t.Errorf("parseSemver(1.2): got %v, want [1 2 0]", got)
	}
}

func TestParseSemver_MajorOnly(t *testing.T) {
	got := parseSemver("3")
	if got[0] != 3 || got[1] != 0 || got[2] != 0 {
		t.Errorf("parseSemver(3): got %v, want [3 0 0]", got)
	}
}

func TestParseSemver_ZeroVersion(t *testing.T) {
	got := parseSemver("0.0.0")
	want := [3]int{0, 0, 0}
	if got != want {
		t.Errorf("parseSemver(0.0.0): got %v, want %v", got, want)
	}
}

// MARK: - currentVersion constant

func TestCurrentVersionIsSet(t *testing.T) {
	if currentVersion == "" {
		t.Error("currentVersion must not be empty")
	}
}

func TestCurrentVersionIsSemver(t *testing.T) {
	v := parseSemver(currentVersion)
	// At minimum the major component should be non-zero for a released product.
	if v[0] == 0 && v[1] == 0 && v[2] == 0 {
		t.Errorf("currentVersion %q parses as 0.0.0 — check the constant", currentVersion)
	}
}
