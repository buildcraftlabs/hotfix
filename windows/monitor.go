//go:build windows

package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// safetyExclusions are processes that must NEVER be killed, regardless of config.
var safetyExclusions = map[string]bool{
	"explorer":  true,
	"svchost":   true,
	"lsass":     true,
	"csrss":     true,
	"wininit":   true,
	"services":  true,
	"winlogon":  true,
	"System":    true,
	"Registry":  true,
	"smss":      true,
	"dwm":       true,
	"Idle":      true,
	"MsMpEng":   true, // Windows Defender — never kill
}

// HotProcess tracks a process that has been above the CPU threshold.
type HotProcess struct {
	PID      int
	Name     string
	CPU      float64
	HotSince time.Time
}

var (
	hotMu       sync.Mutex
	hotMap      = map[int]*HotProcess{} // pid → entry
	monitorStop chan struct{}
	monitorOnce sync.Once
	monitorMu   sync.Mutex // guards monitorStop + monitorOnce
)

// startMonitor launches the background polling goroutine.
func startMonitor() {
	monitorMu.Lock()
	defer monitorMu.Unlock()
	monitorOnce.Do(func() {
		monitorStop = make(chan struct{})
		stop := monitorStop // capture for goroutine
		go monitorLoop(stop)
	})
}

// stopMonitor signals the monitor goroutine to exit and resets state.
func stopMonitor() {
	monitorMu.Lock()
	ch := monitorStop
	monitorOnce = sync.Once{} // allow future restart
	monitorStop = nil
	monitorMu.Unlock()

	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	hotMu.Lock()
	hotMap = map[int]*HotProcess{}
	hotMu.Unlock()
}

func monitorLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Immediate first check
	checkProcesses()

	for {
		select {
		case <-ticker.C:
			checkProcesses()
		case <-stop:
			return
		}
	}
}

// checkProcesses polls WMIC, evaluates thresholds, and kills when warranted.
func checkProcesses() {
	cfg := getConfig()
	if !cfg.Enabled {
		return
	}

	entries, err := queryProcesses()
	if err != nil {
		logf("monitor: query error: %v", err)
		return
	}

	// Build user-whitelist lookup (case-insensitive prefix match).
	wl := map[string]bool{}
	for _, name := range cfg.Whitelist {
		wl[strings.ToLower(name)] = true
	}

	now := time.Now()

	hotMu.Lock()
	defer hotMu.Unlock()

	// Build new set of hot pids so we can evict cooled-down entries.
	hotThisCycle := map[int]bool{}

	for _, e := range entries {
		if e.PID < 10 {
			continue
		}
		if safetyExclusions[e.Name] || safetyExclusions[strings.ToLower(e.Name)] {
			continue
		}
		if wl[strings.ToLower(e.Name)] {
			continue
		}

		if e.CPU < cfg.CPUThreshold {
			continue
		}

		hotThisCycle[e.PID] = true

		hp, exists := hotMap[e.PID]
		if !exists {
			hp = &HotProcess{
				PID:      e.PID,
				Name:     e.Name,
				CPU:      e.CPU,
				HotSince: now,
			}
			hotMap[e.PID] = hp
		} else {
			hp.CPU = e.CPU
		}

		elapsed := now.Sub(hp.HotSince).Seconds()
		logf("monitor: %s (PID %d) at %.1f%% CPU for %.0fs (threshold %.0fs)",
			hp.Name, hp.PID, hp.CPU, elapsed, cfg.KillDuration)

		if elapsed >= cfg.KillDuration {
			killProcess(hp)
			delete(hotMap, e.PID)
		}
	}

	// Evict processes that have cooled down.
	for pid := range hotMap {
		if !hotThisCycle[pid] {
			delete(hotMap, pid)
		}
	}
}

// processEntry is a single row from the WMIC output.
type processEntry struct {
	PID  int
	Name string
	CPU  float64 // percent
}

// queryProcesses runs WMIC and parses CPU percentages for all processes.
func queryProcesses() ([]processEntry, error) {
	// WMIC returns CSV with a blank first line, then a header, then data.
	// Columns: Node,IDProcess,Name,PercentProcessorTime
	cmd := exec.Command("wmic",
		"path", "Win32_PerfFormattedData_PerfProc_Process",
		"get", "IDProcess,Name,PercentProcessorTime",
		"/format:csv",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	// Suppress stderr so it doesn't appear anywhere.
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wmic: %w", err)
	}

	return parseWMICCSV(out.String())
}

// parseWMICCSV parses the CSV output from the WMIC command.
// Format (after skipping the blank first line):
//
//	Node,IDProcess,Name,PercentProcessorTime
//	HOSTNAME,0,_Total,5
//	HOSTNAME,4,System,0
//	...
func parseWMICCSV(raw string) ([]processEntry, error) {
	// Normalize line endings and strip leading/trailing blank lines.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")

	// Find the header line (starts with "Node," case-insensitive).
	startIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "node,") {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		return nil, fmt.Errorf("wmic: no header found in output")
	}

	// Re-join from header onward so csv.Reader can parse it.
	csvData := strings.Join(lines[startIdx:], "\n")
	r := csv.NewReader(strings.NewReader(csvData))
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("wmic: read header: %w", err)
	}

	// Map column name → index (case-insensitive).
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	pidCol, hasPID := colIdx["idprocess"]
	nameCol, hasName := colIdx["name"]
	cpuCol, hasCPU := colIdx["percentprocessortime"]
	if !hasPID || !hasName || !hasCPU {
		return nil, fmt.Errorf("wmic: unexpected columns: %v", header)
	}

	var entries []processEntry
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows silently.
			continue
		}
		if len(row) <= max3(pidCol, nameCol, cpuCol) {
			continue
		}

		pidStr := strings.TrimSpace(row[pidCol])
		name := strings.TrimSpace(row[nameCol])
		cpuStr := strings.TrimSpace(row[cpuCol])

		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid < 0 {
			continue
		}

		cpu, err := strconv.ParseFloat(cpuStr, 64)
		if err != nil {
			continue
		}

		// Skip the aggregate "_Total" row.
		if name == "_Total" || name == "" {
			continue
		}

		// Strip ".exe" suffix for comparison friendliness (optional).
		name = strings.TrimSuffix(name, ".exe")

		entries = append(entries, processEntry{PID: pid, Name: name, CPU: cpu})
	}

	return entries, nil
}

// killProcess terminates a process via taskkill and updates the tray label.
func killProcess(hp *HotProcess) {
	logf("monitor: killing %s (PID %d) — %.1f%% CPU for %.0fs",
		hp.Name, hp.PID, hp.CPU, time.Since(hp.HotSince).Seconds())

	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(hp.PID), "/F")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		logf("monitor: taskkill failed for PID %d: %v", hp.PID, err)
		return
	}

	logf("monitor: killed %s (PID %d)", hp.Name, hp.PID)
	notifyKilled(hp.Name, hp.PID, hp.CPU)
}

// killAllHot terminates every currently tracked hot process (sleep handler).
func killAllHot() {
	hotMu.Lock()
	defer hotMu.Unlock()
	for pid, hp := range hotMap {
		killProcess(hp)
		delete(hotMap, pid)
	}
}

// watchSleep starts a goroutine that detects Windows sleep/suspend events via
// PowerShell and calls killAllHot when KillOnSleep is enabled.
// This uses the Win32_PowerManagementEvent WMI event (pure PowerShell, no CGO).
func watchSleep() {
	go func() {
		// PowerShell script: block until a suspend event fires, then print "sleep".
		// Win32_PowerManagementEvent EventType 4 = suspend.
		ps := `$q = "SELECT * FROM Win32_PowerManagementEvent WHERE EventType = 4";` +
			`$null = Register-WmiEvent -Query $q -SourceIdentifier "HotfixSleep";` +
			`while($true){` +
			`$e = Wait-Event -SourceIdentifier "HotfixSleep";` +
			`Remove-Event -SourceIdentifier "HotfixSleep";` +
			`Write-Output "sleep"` +
			`}`

		cmd := exec.Command("powershell", "-NonInteractive", "-NoProfile",
			"-WindowStyle", "Hidden", "-Command", ps)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logf("sleep-watcher: pipe error: %v", err)
			return
		}
		cmd.Stderr = io.Discard

		if err := cmd.Start(); err != nil {
			logf("sleep-watcher: start error: %v", err)
			return
		}

		buf := make([]byte, 64)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				line := strings.TrimSpace(string(buf[:n]))
				if strings.Contains(line, "sleep") {
					cfg := getConfig()
					if cfg.Enabled && cfg.KillOnSleep {
						logf("sleep-watcher: suspend detected — killing hot processes")
						killAllHot()
					}
				}
			}
			if err != nil {
				break
			}
		}
		_ = cmd.Wait()
		logf("sleep-watcher: exited")
	}()
}

// max3 returns the largest of three ints (avoids importing math).
func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
