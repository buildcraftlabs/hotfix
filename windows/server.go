//go:build windows

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

//go:embed assets/settings.html
var assets embed.FS

var (
	serverPort int
	serverOnce sync.Once
	serverMu   sync.Mutex
)

// startServer launches the HTTP settings server on a random port and returns
// the port number. Calling it more than once returns the existing port.
func startServer() int {
	serverOnce.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			logf("server: listen error: %v", err)
			return
		}

		serverMu.Lock()
		serverPort = ln.Addr().(*net.TCPAddr).Port
		serverMu.Unlock()

		logf("server: listening on http://127.0.0.1:%d", serverPort)

		mux := http.NewServeMux()
		mux.HandleFunc("/", handleSettings)
		mux.HandleFunc("/config", handleConfig)
		mux.HandleFunc("/save", handleSave)
		mux.HandleFunc("/log", handleLog)

		go func() {
			if err := http.Serve(ln, mux); err != nil {
				logf("server: serve error: %v", err)
			}
		}()
	})

	serverMu.Lock()
	defer serverMu.Unlock()
	return serverPort
}


// handleSettings serves the embedded settings HTML page.
func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := assets.ReadFile("assets/settings.html")
	if err != nil {
		http.Error(w, "settings page not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleLog serves the Hotfix log file as plain text so it can be viewed in
// the browser from the settings page.
func handleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	path := logFilePath()
	if path == "" {
		_, _ = w.Write([]byte("Log file location unavailable."))
		return
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		_, _ = w.Write([]byte("No log entries yet."))
		return
	}
	_, _ = w.Write(data)
}

// handleConfig returns the current config as JSON.
func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := getConfig()
	setCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

// handleSave receives a JSON config body, validates it, and saves.
func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setCORSHeaders(w)

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	var cfg Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate.
	if cfg.CPUThreshold < 1 || cfg.CPUThreshold > 100 {
		http.Error(w, "cpu_threshold must be 1–100", http.StatusBadRequest)
		return
	}
	if cfg.KillDuration < 5 {
		http.Error(w, "kill_duration must be >= 5", http.StatusBadRequest)
		return
	}
	if cfg.Whitelist == nil {
		cfg.Whitelist = []string{}
	}

	if err := setConfig(cfg); err != nil {
		logf("server: save config error: %v", err)
		http.Error(w, "save error", http.StatusInternalServerError)
		return
	}

	logf("server: config saved (enabled=%v, threshold=%.0f%%, kill_after=%.0fs)",
		cfg.Enabled, cfg.CPUThreshold, cfg.KillDuration)

	// Start or stop the monitor to match the new enabled state. Both calls are
	// idempotent, so saving repeatedly (or saving the same state) is safe. This
	// is required because the monitor goroutine is only launched at startup when
	// monitoring is enabled — without this, toggling Enabled on from the web UI
	// would not begin monitoring until the app was restarted.
	if cfg.Enabled {
		startMonitor()
	} else {
		stopMonitor()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// openSettingsPage opens the embedded settings HTML in the user's default browser.
func openSettingsPage() {
	serverMu.Lock()
	port := serverPort
	serverMu.Unlock()
	if port == 0 {
		return
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	// Use cmd /c start to open the URL without flashing a console window.
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}
