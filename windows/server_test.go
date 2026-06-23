//go:build windows

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempAppData redirects config/log storage to a temp dir for the duration
// of a test.
func withTempAppData(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", dir)
	_ = os.MkdirAll(filepath.Join(dir, "Hotfix"), 0755)
	t.Cleanup(func() { os.Setenv("APPDATA", orig) })
}

func TestHandleConfig_ReturnsJSON(t *testing.T) {
	withTempAppData(t)
	_ = setConfig(defaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()
	handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var cfg Config
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if cfg.CPUThreshold != 80.0 {
		t.Errorf("CPUThreshold: got %.1f, want 80.0", cfg.CPUThreshold)
	}
}

func TestHandleConfig_RejectsNonGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/config", nil)
	rec := httptest.NewRecorder()
	handleConfig(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /config: got %d, want 405", rec.Code)
	}
}

func TestHandleSave_AcceptsValidConfig(t *testing.T) {
	withTempAppData(t)

	body := `{"enabled":true,"cpu_threshold":75,"kill_duration":90,"kill_on_sleep":true,"whitelist":["steam"]}`
	req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSave(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("valid save: got %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	got := getConfig()
	if got.CPUThreshold != 75 || got.KillDuration != 90 {
		t.Errorf("config not persisted: got %+v", got)
	}
}

func TestHandleSave_RejectsLowCPUThreshold(t *testing.T) {
	withTempAppData(t)
	body := `{"enabled":true,"cpu_threshold":0,"kill_duration":60,"whitelist":[]}`
	req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSave(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("cpu_threshold=0 should be rejected: got %d, want 400", rec.Code)
	}
}

func TestHandleSave_RejectsHighCPUThreshold(t *testing.T) {
	withTempAppData(t)
	body := `{"enabled":true,"cpu_threshold":250,"kill_duration":60,"whitelist":[]}`
	req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSave(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("cpu_threshold=250 should be rejected: got %d, want 400", rec.Code)
	}
}

func TestHandleSave_RejectsLowKillDuration(t *testing.T) {
	withTempAppData(t)
	body := `{"enabled":true,"cpu_threshold":80,"kill_duration":2,"whitelist":[]}`
	req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSave(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("kill_duration=2 should be rejected: got %d, want 400", rec.Code)
	}
}

func TestHandleSave_RejectsInvalidJSON(t *testing.T) {
	withTempAppData(t)
	req := httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(`{nope`))
	rec := httptest.NewRecorder()
	handleSave(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON should be rejected: got %d, want 400", rec.Code)
	}
}

func TestHandleSave_RejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/save", nil)
	rec := httptest.NewRecorder()
	handleSave(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /save: got %d, want 405", rec.Code)
	}
}

func TestHandleLog_ServesPlainText(t *testing.T) {
	withTempAppData(t)
	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	rec := httptest.NewRecorder()
	handleLog(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /log: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type: got %q, want text/plain*", ct)
	}
}

func TestHandleSettings_ServesHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handleSettings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: got %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Hotfix Settings") {
		t.Error("settings page should contain the title 'Hotfix Settings'")
	}
}

func TestHandleSettings_NotFoundForOtherPaths(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/bogus", nil)
	rec := httptest.NewRecorder()
	handleSettings(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /bogus: got %d, want 404", rec.Code)
	}
}

func monitorRunning() bool {
	monitorMu.Lock()
	defer monitorMu.Unlock()
	return monitorStop != nil
}

// Enabling monitoring from the web Settings UI must actually start the monitor
// (and disabling must stop it) — not just persist the flag. A high threshold and
// long kill duration guarantee the immediate first poll never terminates a real
// process during the test.
func TestHandleSave_StartsAndStopsMonitor(t *testing.T) {
	withTempAppData(t)
	stopMonitor()
	t.Cleanup(stopMonitor)

	if monitorRunning() {
		t.Fatal("precondition: monitor should be stopped before the test")
	}

	enable := `{"enabled":true,"cpu_threshold":95,"kill_duration":300,"whitelist":[]}`
	rec := httptest.NewRecorder()
	handleSave(rec, httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(enable)))
	if rec.Code != http.StatusOK {
		t.Fatalf("enable save: got %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if !monitorRunning() {
		t.Error("monitor should be running after enabling monitoring via /save")
	}

	disable := `{"enabled":false,"cpu_threshold":95,"kill_duration":300,"whitelist":[]}`
	rec2 := httptest.NewRecorder()
	handleSave(rec2, httptest.NewRequest(http.MethodPost, "/save", strings.NewReader(disable)))
	if rec2.Code != http.StatusOK {
		t.Fatalf("disable save: got %d, want 200", rec2.Code)
	}
	if monitorRunning() {
		t.Error("monitor should be stopped after disabling monitoring via /save")
	}
}
