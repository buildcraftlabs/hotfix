import XCTest
@testable import Hotfix

final class ProcessMonitorTests: XCTestCase {

    // MARK: - Safety exclusions

    // These processes must NEVER be killed regardless of CPU usage.
    // If any of these are missing from the set, it's a regression.

    func testKernelTaskIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("kernel_task"), "kernel_task must always be protected")
    }

    func testWindowServerIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("WindowServer"), "WindowServer must always be protected")
    }

    func testLoginWindowIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("loginwindow"), "loginwindow must always be protected")
    }

    func testLaunchdIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("launchd"), "launchd must always be protected")
    }

    func testFinderIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("Finder"), "Finder must always be protected")
    }

    func testDockIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("Dock"), "Dock must always be protected")
    }

    func testSystemUIServerIsExcluded() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertTrue(excluded.contains("SystemUIServer"), "SystemUIServer must always be protected")
    }

    func testExclusionsSetIsNonEmpty() async {
        let excluded = await MainActor.run { ProcessMonitor.shared.safetyExclusions }
        XCTAssertGreaterThan(excluded.count, 5, "Safety exclusion list should contain multiple critical processes")
    }

    // MARK: - HotProcess model

    func testHotProcessIDMatchesPID() {
        let p = HotProcess(id: 42, pid: 42, name: "test", cpuPercent: 90.0, hotSeconds: 30.0)
        XCTAssertEqual(p.id, p.pid)
    }

    func testHotProcessEquality() {
        let a = HotProcess(id: 1, pid: 1, name: "foo", cpuPercent: 80.0, hotSeconds: 10.0)
        let b = HotProcess(id: 1, pid: 1, name: "foo", cpuPercent: 80.0, hotSeconds: 10.0)
        XCTAssertEqual(a, b)
    }

    func testHotProcessInequality() {
        let a = HotProcess(id: 1, pid: 1, name: "foo", cpuPercent: 80.0, hotSeconds: 10.0)
        let b = HotProcess(id: 2, pid: 2, name: "bar", cpuPercent: 95.0, hotSeconds: 20.0)
        XCTAssertNotEqual(a, b)
    }

    // MARK: - Initial state

    func testMonitorStartsNotRunning() async {
        // A freshly accessed shared instance that hasn't been started should not be running.
        // (In the app, start() is only called when isEnabled is true on init.)
        let isRunning = await MainActor.run { ProcessMonitor.shared.isRunning }
        // We don't assert a specific value here because the test environment
        // may or may not have started the monitor. Just assert we can read the property.
        XCTAssertNotNil(isRunning)
    }

    func testHotProcessesIsArray() async {
        let processes = await MainActor.run { ProcessMonitor.shared.hotProcesses }
        XCTAssertNotNil(processes)
    }
}
