import XCTest
@testable import Hotfix

final class CrashReporterTests: XCTestCase {

    private func queryItems(_ url: URL) throws -> [String: String] {
        let comps = try XCTUnwrap(URLComponents(url: url, resolvingAgainstBaseURL: false))
        return Dictionary(uniqueKeysWithValues: (comps.queryItems ?? []).map { ($0.name, $0.value ?? "") })
    }

    func testBuildIssueURL_pointsAtNewIssueWithCrashLabel() throws {
        let url = try XCTUnwrap(CrashReporter.buildIssueURL(
            crash: "boom\nat frame", version: "9.9.9", arch: "arm64", recentLog: nil))
        XCTAssertTrue(url.absoluteString.hasPrefix(CrashReporter.issueBaseURL))
        let items = try queryItems(url)
        XCTAssertEqual(items["labels"], "crash")
        XCTAssertEqual(items["title"], "Crash report — Hotfix v9.9.9 (macOS/arm64)")
    }

    func testBuildIssueURL_bodyIncludesCrashAndMeta() throws {
        let url = try XCTUnwrap(CrashReporter.buildIssueURL(
            crash: "boom\nat frame", version: "9.9.9", arch: "arm64", recentLog: nil))
        let body = try queryItems(url)["body"] ?? ""
        XCTAssertTrue(body.contains("boom"))
        XCTAssertTrue(body.contains("**Version:** 9.9.9"))
        XCTAssertTrue(body.contains("macOS (arm64)"))
        XCTAssertFalse(body.contains("Recent log"), "no log section expected when no log is supplied")
    }

    func testBuildIssueURL_includesRecentLogWhenPresent() throws {
        let url = try XCTUnwrap(CrashReporter.buildIssueURL(
            crash: "x", version: "1.0.0", arch: "x86_64", recentLog: "line1\nline2"))
        let body = try queryItems(url)["body"] ?? ""
        XCTAssertTrue(body.contains("Recent log"))
        XCTAssertTrue(body.contains("line2"))
    }

    func testTruncate_keepsShortStringsAndCutsLongOnes() {
        XCTAssertEqual(CrashReporter.truncate("hello", 10), "hello")
        let t = CrashReporter.truncate(String(repeating: "a", count: 100), 10)
        XCTAssertTrue(t.hasPrefix(String(repeating: "a", count: 10)))
        XCTAssertTrue(t.contains("truncated"))
    }

    func testTailLines_returnsLastNLines() {
        XCTAssertEqual(CrashReporter.tailLines("a\nb\nc\nd\ne", 2), "d\ne")
        XCTAssertEqual(CrashReporter.tailLines("only", 5), "only")
    }
}
