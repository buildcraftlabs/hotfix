import XCTest
@testable import Hotfix

final class UpdateCheckerTests: XCTestCase {

    private let checker = UpdateChecker.shared

    // MARK: - Newer version available

    func testNewerPatchVersion() {
        XCTAssertTrue(checker.isNewerVersion("1.0.4", than: "1.0.3"))
    }

    func testNewerMinorVersion() {
        XCTAssertTrue(checker.isNewerVersion("1.1.0", than: "1.0.3"))
    }

    func testNewerMajorVersion() {
        XCTAssertTrue(checker.isNewerVersion("2.0.0", than: "1.9.9"))
    }

    func testNewerWithMissingPatchComponent() {
        // "1.1" should be treated as "1.1.0", which is newer than "1.0.3"
        XCTAssertTrue(checker.isNewerVersion("1.1", than: "1.0.3"))
    }

    // MARK: - Same version (not newer)

    func testSameVersion() {
        XCTAssertFalse(checker.isNewerVersion("1.0.3", than: "1.0.3"))
    }

    func testSameVersionWithMissingPatch() {
        XCTAssertFalse(checker.isNewerVersion("1.0", than: "1.0.0"))
    }

    func testSameVersionMajorOnly() {
        XCTAssertFalse(checker.isNewerVersion("1", than: "1.0.0"))
    }

    // MARK: - Older version (not newer)

    func testOlderPatchVersion() {
        XCTAssertFalse(checker.isNewerVersion("1.0.2", than: "1.0.3"))
    }

    func testOlderMinorVersion() {
        XCTAssertFalse(checker.isNewerVersion("1.0.9", than: "1.1.0"))
    }

    func testOlderMajorVersion() {
        XCTAssertFalse(checker.isNewerVersion("1.9.9", than: "2.0.0"))
    }

    // MARK: - Current version constant

    func testCurrentVersionIsNonEmpty() {
        XCTAssertFalse(UpdateChecker.currentVersion.isEmpty)
    }

    func testCurrentVersionFormat() {
        // Version must be semver-like: at least "major.minor"
        let components = UpdateChecker.currentVersion.split(separator: ".")
        XCTAssertGreaterThanOrEqual(components.count, 2)
        for component in components {
            XCTAssertNotNil(Int(component), "Version component '\(component)' is not an integer")
        }
    }

    // MARK: - Tag name stripping

    func testTagWithVPrefix() {
        // The app strips the leading "v" from GitHub tag names before comparing.
        // "v1.0.4" → "1.0.4", which should be newer than "1.0.3".
        let tag = "v1.0.4"
        let stripped = tag.hasPrefix("v") ? String(tag.dropFirst()) : tag
        XCTAssertTrue(checker.isNewerVersion(stripped, than: "1.0.3"))
    }

    func testTagWithoutVPrefix() {
        let tag = "1.0.4"
        let stripped = tag.hasPrefix("v") ? String(tag.dropFirst()) : tag
        XCTAssertTrue(checker.isNewerVersion(stripped, than: "1.0.3"))
    }
}
