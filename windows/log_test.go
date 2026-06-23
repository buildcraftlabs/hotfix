//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogf_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	defer os.Setenv("APPDATA", orig)

	// Reset the package-level logger around this test.
	logMu.Lock()
	oldFile := logFile
	logFile = nil
	logMu.Unlock()
	defer func() {
		closeLog()
		logMu.Lock()
		logFile = oldFile
		logMu.Unlock()
	}()

	initLog()
	logf("hello %d %s", 42, "world")
	closeLog()

	path := filepath.Join(dir, "Hotfix", "hotfix.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	if !strings.Contains(string(data), "hello 42 world") {
		t.Errorf("log file missing message; contents: %q", string(data))
	}
}

func TestLogf_AppendsNewline(t *testing.T) {
	dir := t.TempDir()
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	defer os.Setenv("APPDATA", orig)

	logMu.Lock()
	oldFile := logFile
	logFile = nil
	logMu.Unlock()
	defer func() {
		closeLog()
		logMu.Lock()
		logFile = oldFile
		logMu.Unlock()
	}()

	initLog()
	logf("line one")
	logf("line two")
	closeLog()

	data, _ := os.ReadFile(filepath.Join(dir, "Hotfix", "hotfix.log"))
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d: %q", len(lines), string(data))
	}
}

func TestLogFilePath_NonEmpty(t *testing.T) {
	dir := t.TempDir()
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	defer os.Setenv("APPDATA", orig)

	p := logFilePath()
	if p == "" {
		t.Fatal("logFilePath returned empty string")
	}
	if !strings.HasSuffix(p, filepath.Join("Hotfix", "hotfix.log")) {
		t.Errorf("logFilePath: got %q, want suffix Hotfix/hotfix.log", p)
	}
}
