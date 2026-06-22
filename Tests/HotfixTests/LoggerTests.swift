import XCTest
@testable import Hotfix

final class LoggerTests: XCTestCase {

    /// Returns a fresh, isolated temporary directory for each test so writes
    /// never touch the real ~/Library/Logs/Hotfix/hotfix.log.
    private func makeTempDir() -> URL {
        let dir = FileManager.default.temporaryDirectory
            .appendingPathComponent("HotfixLoggerTests-\(UUID().uuidString)", isDirectory: true)
        return dir
    }

    // MARK: - File creation & path

    func testCreatesLogFileOnInit() throws {
        let dir = makeTempDir()
        let log = Log(logsDirectory: dir)
        let url = try XCTUnwrap(log.fileURL)
        XCTAssertEqual(url.lastPathComponent, "hotfix.log")
        XCTAssertTrue(FileManager.default.fileExists(atPath: url.path),
                      "Log file should be created during init")
        try? FileManager.default.removeItem(at: dir)
    }

    func testFileURLNestedUnderProvidedDirectory() throws {
        let dir = makeTempDir()
        let log = Log(logsDirectory: dir)
        let url = try XCTUnwrap(log.fileURL)
        XCTAssertEqual(url.deletingLastPathComponent().path, dir.path)
        try? FileManager.default.removeItem(at: dir)
    }

    func testNilDirectoryProducesNilURL() {
        let log = Log(logsDirectory: nil)
        XCTAssertNil(log.fileURL)
        // Logging with no destination must not crash.
        log.log("this should be a no-op")
        log.flush()
    }

    func testDefaultDirectoryEndsWithLogsHotfix() throws {
        let dir = try XCTUnwrap(Log.defaultDirectory)
        XCTAssertEqual(dir.lastPathComponent, "Hotfix")
        XCTAssertEqual(dir.deletingLastPathComponent().lastPathComponent, "Logs")
    }

    func testSharedLogFileURLIsTheExpectedPath() throws {
        let url = try XCTUnwrap(Log.shared.fileURL)
        XCTAssertTrue(url.path.hasSuffix("Logs/Hotfix/hotfix.log"),
                      "Shared logger should point at ~/Library/Logs/Hotfix/hotfix.log")
    }

    // MARK: - Line formatting (pure)

    func testFormattedLineHasTimestampMessageAndNewline() {
        let log = Log(logsDirectory: nil)
        let line = log.formattedLine("hello world", date: Date(timeIntervalSince1970: 0))
        XCTAssertTrue(line.hasPrefix("["), "Line should start with the timestamp bracket")
        XCTAssertTrue(line.contains("] hello world"), "Line should contain '] ' then the message")
        XCTAssertTrue(line.hasSuffix("\n"), "Line should be newline-terminated")
    }

    func testFormattedLineTimestampMatchesExpectedPattern() {
        let log = Log(logsDirectory: nil)
        let line = log.formattedLine("x")
        // Expect: [YYYY-MM-DD HH:MM:SS] x\n
        let pattern = #"^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] x\n$"#
        let range = line.range(of: pattern, options: .regularExpression)
        XCTAssertNotNil(range, "Got unexpected line format: \(line)")
    }

    // MARK: - Writing & appending

    func testLogWritesMessageToFile() throws {
        let dir = makeTempDir()
        let log = Log(logsDirectory: dir)
        let url = try XCTUnwrap(log.fileURL)

        let marker = "kill event \(UUID().uuidString)"
        log.log(marker)
        log.flush()

        let contents = try String(contentsOf: url, encoding: .utf8)
        XCTAssertTrue(contents.contains(marker), "Logged message should be in the file")
        try? FileManager.default.removeItem(at: dir)
    }

    func testLogAppendsMultipleLinesInOrder() throws {
        let dir = makeTempDir()
        let log = Log(logsDirectory: dir)
        let url = try XCTUnwrap(log.fileURL)

        log.log("first")
        log.log("second")
        log.flush()

        let contents = try String(contentsOf: url, encoding: .utf8)
        let firstIdx = try XCTUnwrap(contents.range(of: "first"))
        let secondIdx = try XCTUnwrap(contents.range(of: "second"))
        XCTAssertLessThan(firstIdx.lowerBound, secondIdx.lowerBound,
                          "Lines should be appended in call order")
        // Two log calls → two newline-terminated lines.
        let lineCount = contents.split(separator: "\n", omittingEmptySubsequences: true).count
        XCTAssertEqual(lineCount, 2)
        try? FileManager.default.removeItem(at: dir)
    }

    func testLogPersistsAcrossSeparateLoggerInstances() throws {
        let dir = makeTempDir()

        let first = Log(logsDirectory: dir)
        first.log("from-first-instance")
        first.flush()

        // A second logger pointed at the same directory should append, not truncate.
        let second = Log(logsDirectory: dir)
        second.log("from-second-instance")
        second.flush()

        let url = try XCTUnwrap(second.fileURL)
        let contents = try String(contentsOf: url, encoding: .utf8)
        XCTAssertTrue(contents.contains("from-first-instance"))
        XCTAssertTrue(contents.contains("from-second-instance"))
        try? FileManager.default.removeItem(at: dir)
    }
}
