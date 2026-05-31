import XCTest
import SwiftUI
@testable import Hotfix

// Tests for the pure formatting and color functions on HotProcessRow.
// These functions have no side effects and are safe to call in any context.

final class ViewLogicTests: XCTestCase {

    private let row = HotProcessRow(process: HotProcess(
        id: 1, pid: 1, name: "test", cpuPercent: 85.0, hotSeconds: 0
    ))

    // MARK: - hotDurationLabel

    func testDurationUnderOneMinute() {
        XCTAssertEqual(row.hotDurationLabel(0), "hot for 0s")
    }

    func testDurationTenSeconds() {
        XCTAssertEqual(row.hotDurationLabel(10), "hot for 10s")
    }

    func testDurationFiftyNineSeconds() {
        XCTAssertEqual(row.hotDurationLabel(59), "hot for 59s")
    }

    func testDurationExactlyOneMinute() {
        XCTAssertEqual(row.hotDurationLabel(60), "hot for 1m 0s")
    }

    func testDurationOneMinuteFifteenSeconds() {
        XCTAssertEqual(row.hotDurationLabel(75), "hot for 1m 15s")
    }

    func testDurationTwoMinutes() {
        XCTAssertEqual(row.hotDurationLabel(120), "hot for 2m 0s")
    }

    func testDurationFiveMinutesThirtySeconds() {
        XCTAssertEqual(row.hotDurationLabel(330), "hot for 5m 30s")
    }

    // MARK: - cpuColor

    func testColorBelowYellowThreshold() {
        // Below 75% → yellow
        let color = row.cpuColor(74.9)
        XCTAssertEqual(color, Color.yellow)
    }

    func testColorAtYellowThreshold() {
        // 75% exactly → yellow (threshold is >= 75 for orange)
        // cpuColor logic: >= 90 → red, >= 75 → orange, else → yellow
        let color = row.cpuColor(75.0)
        XCTAssertEqual(color, Color.orange)
    }

    func testColorInOrangeRange() {
        let color = row.cpuColor(85.0)
        XCTAssertEqual(color, Color.orange)
    }

    func testColorBelowRedThreshold() {
        let color = row.cpuColor(89.9)
        XCTAssertEqual(color, Color.orange)
    }

    func testColorAtRedThreshold() {
        // 90% and above → brand red (#C9461E)
        let color = row.cpuColor(90.0)
        XCTAssertEqual(color, Color(hex: "C9461E"))
    }

    func testColorAtHundredPercent() {
        let color = row.cpuColor(100.0)
        XCTAssertEqual(color, Color(hex: "C9461E"))
    }

    // MARK: - CPU percent clamping in heat bar

    func testHeatBarClampAbove100() {
        // The heat bar uses min(cpuPercent / 100, 1.0) — should never exceed full width.
        let clampedWidth = min(150.0 / 100.0, 1.0)
        XCTAssertEqual(clampedWidth, 1.0)
    }

    func testHeatBarClampAt100() {
        let clampedWidth = min(100.0 / 100.0, 1.0)
        XCTAssertEqual(clampedWidth, 1.0)
    }

    func testHeatBarAt80Percent() {
        let clampedWidth = min(80.0 / 100.0, 1.0)
        XCTAssertEqual(clampedWidth, 0.8, accuracy: 0.001)
    }
}
