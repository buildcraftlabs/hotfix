import Foundation
import AppKit
import UserNotifications

// MARK: - HotProcess model
struct HotProcess: Identifiable, Equatable {
    let id: Int32       // pid as id
    var pid: Int32
    var name: String
    var cpuPercent: Double
    var hotSeconds: Double
}

// MARK: - ProcessMonitor
@MainActor
class ProcessMonitor: ObservableObject {
    static let shared = ProcessMonitor()

    @Published var hotProcesses: [HotProcess] = []
    @Published var isKilling: Bool = false
    @Published var lastKilledName: String? = nil
    @Published var isRunning: Bool = false

    /// System processes that must NEVER be killed
    let safetyExclusions: Set<String> = [
        "kernel_task", "WindowServer", "loginwindow", "launchd",
        "mds_stores", "bird", "mds", "mdworker", "mdworker_shared",
        "coreaudiod", "diskarbitrationd", "configd", "bluetoothd",
        "com.apple.WebKit", "SystemUIServer", "Finder", "Dock"
    ]

    private var timer: Timer?
    /// pid → first-seen-hot timestamp (seconds since reference date)
    private var hotStartTimes: [Int32: TimeInterval] = [:]
    private var sleepObserver: Any?

    private init() {}

    // MARK: - Start / Stop
    func start() {
        guard !isRunning else { return }
        isRunning = true
        hotStartTimes = [:]

        // Register for sleep notification
        sleepObserver = NSWorkspace.shared.notificationCenter.addObserver(
            forName: NSWorkspace.willSleepNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            guard let self else { return }
            Task { @MainActor in
                self.handleSleepNotification()
            }
        }

        // Schedule 5-second polling timer
        let t = Timer(timeInterval: 5.0, repeats: true) { [weak self] _ in
            guard let self else { return }
            Task { @MainActor in
                self.checkProcesses()
            }
        }
        RunLoop.main.add(t, forMode: .common)
        self.timer = t

        // Immediate first check
        checkProcesses()
    }

    func stop() {
        isRunning = false
        timer?.invalidate()
        timer = nil
        hotStartTimes = [:]
        hotProcesses = []
        isKilling = false

        if let obs = sleepObserver {
            NSWorkspace.shared.notificationCenter.removeObserver(obs)
            sleepObserver = nil
        }
    }

    // MARK: - Sleep handler
    private func handleSleepNotification() {
        guard PreferencesManager.shared.killOnSleep else { return }
        // Kill all currently tracked hot processes immediately
        let toKill = hotProcesses
        for proc in toKill {
            terminateProcess(proc)
        }
    }

    // MARK: - Process check
    func checkProcesses() {
        guard isRunning else { return }
        let prefs = PreferencesManager.shared
        let threshold = prefs.cpuThreshold
        let killDuration = prefs.killDuration
        let userWhitelist = Set(prefs.whitelist)

        let now = Date().timeIntervalSinceReferenceDate

        // Run ps in background, parse on main
        Task.detached(priority: .userInitiated) {
            let parsed = Self.runPS()
            await MainActor.run {
                self.processResults(
                    parsed: parsed,
                    threshold: threshold,
                    killDuration: killDuration,
                    userWhitelist: userWhitelist,
                    now: now
                )
            }
        }
    }

    private func processResults(
        parsed: [(pid: Int32, cpu: Double, name: String)],
        threshold: Double,
        killDuration: Double,
        userWhitelist: Set<String>,
        now: TimeInterval
    ) {
        var newHot: [HotProcess] = []
        var updatedStartTimes: [Int32: TimeInterval] = [:]

        for entry in parsed {
            let pid = entry.pid
            let cpu = entry.cpu
            let name = entry.name

            // Safety checks
            guard pid >= 100 else { continue }
            guard !safetyExclusions.contains(name) else { continue }
            guard !userWhitelist.contains(name) else { continue }

            if cpu >= threshold {
                let startTime = hotStartTimes[pid] ?? now
                updatedStartTimes[pid] = startTime
                let hotSecs = now - startTime

                let proc = HotProcess(
                    id: pid,
                    pid: pid,
                    name: name,
                    cpuPercent: cpu,
                    hotSeconds: hotSecs
                )
                newHot.append(proc)

                // Should we kill it?
                if hotSecs >= killDuration {
                    terminateProcess(proc)
                    // Don't carry over start time after kill
                    updatedStartTimes.removeValue(forKey: pid)
                }
            }
        }

        hotStartTimes = updatedStartTimes
        hotProcesses = newHot.sorted { $0.cpuPercent > $1.cpuPercent }
    }

    // MARK: - Termination
    func terminateProcess(_ proc: HotProcess) {
        let result = kill(proc.pid, SIGTERM)
        if result == 0 {
            lastKilledName = proc.name
            isKilling = true

            // Reset isKilling visual after 2 seconds
            DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) { [weak self] in
                self?.isKilling = false
            }

            sendKillNotification(name: proc.name, pid: proc.pid, cpu: proc.cpuPercent)
        }
    }

    // MARK: - Notification
    private func sendKillNotification(name: String, pid: Int32, cpu: Double) {
        let content = UNMutableNotificationContent()
        content.title = "Process Terminated"
        content.body = "\(name) (PID \(pid)) was using \(String(format: "%.1f", cpu))% CPU and has been terminated."
        content.sound = .default

        let request = UNNotificationRequest(
            identifier: "kill-\(pid)-\(Date().timeIntervalSince1970)",
            content: content,
            trigger: nil
        )
        UNUserNotificationCenter.current().add(request) { error in
            if let error {
                print("[Hotfix] Notification error: \(error.localizedDescription)")
            }
        }
    }

    // MARK: - ps runner (nonisolated, runs off main actor)
    nonisolated static func runPS() -> [(pid: Int32, cpu: Double, name: String)] {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/bin/ps")
        process.arguments = ["-Ao", "pid,pcpu,comm", "-r"]

        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = Pipe() // suppress errors

        do {
            try process.run()
            process.waitUntilExit()
        } catch {
            print("[Hotfix] ps error: \(error)")
            return []
        }

        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        guard let output = String(data: data, encoding: .utf8) else { return [] }

        return parsePS(output: output)
    }

    nonisolated static func parsePS(output: String) -> [(pid: Int32, cpu: Double, name: String)] {
        var results: [(pid: Int32, cpu: Double, name: String)] = []
        let lines = output.components(separatedBy: "\n")

        for (index, line) in lines.enumerated() {
            // Skip header line
            if index == 0 { continue }
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            guard !trimmed.isEmpty else { continue }

            // Split by whitespace, but comm may contain path — take last component
            // Format: "  PID  %CPU COMM"
            // ps -o pid,pcpu,comm: columns are fixed-width for pid and pcpu, rest is comm
            let parts = trimmed.components(separatedBy: .whitespaces).filter { !$0.isEmpty }
            guard parts.count >= 3 else { continue }

            guard let pid = Int32(parts[0]) else { continue }
            guard let cpu = Double(parts[1]) else { continue }

            // Everything from parts[2] onward is the comm (may contain spaces if path)
            let commFull = parts[2...].joined(separator: " ")
            // Take last path component
            let name: String
            if commFull.contains("/") {
                name = URL(fileURLWithPath: commFull).lastPathComponent
            } else {
                name = commFull
            }

            // Only include processes with non-zero CPU to avoid noise
            guard cpu > 0 else { continue }

            results.append((pid: pid, cpu: cpu, name: name))
        }

        return results
    }
}
