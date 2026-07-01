import Foundation

/// Thread-safe file logger that mirrors the Windows logger format.
/// Writes to ~/Library/Logs/Hotfix/hotfix.log and also prints to stderr.
final class Log {
    static let shared = Log()

    private let queue = DispatchQueue(label: "com.buildcraftlabs.hotfix.log")
    private var handle: FileHandle?
    private let dateFormatter: DateFormatter

    /// Cap the log at 5 MB; beyond that it is rolled to hotfix.log.1 (one backup)
    /// and a fresh log is started, so the file never grows unbounded.
    private let maxBytes: UInt64 = 5 * 1024 * 1024

    /// Resolved log file URL (~/Library/Logs/Hotfix/hotfix.log), or nil if it
    /// could not be created.
    let fileURL: URL?

    /// Default logs directory: ~/Library/Logs/Hotfix.
    static var defaultDirectory: URL? {
        FileManager.default.urls(for: .libraryDirectory, in: .userDomainMask)
            .first?
            .appendingPathComponent("Logs/Hotfix", isDirectory: true)
    }

    convenience init() {
        self.init(logsDirectory: Log.defaultDirectory)
    }

    /// Designated initializer. `logsDirectory` is injectable so tests can write
    /// to a temporary directory instead of the real ~/Library/Logs location.
    init(logsDirectory: URL?) {
        dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd HH:mm:ss"
        dateFormatter.locale = Locale(identifier: "en_US_POSIX")

        guard let logsDir = logsDirectory else {
            fileURL = nil
            return
        }

        let fm = FileManager.default
        try? fm.createDirectory(at: logsDir, withIntermediateDirectories: true)
        let url = logsDir.appendingPathComponent("hotfix.log")
        fileURL = url

        if !fm.fileExists(atPath: url.path) {
            fm.createFile(atPath: url.path, contents: nil)
        }
        handle = try? FileHandle(forWritingTo: url)
        handle?.seekToEndOfFile()
    }

    /// Format a single timestamped log line. Pure — no side effects.
    func formattedLine(_ message: String, date: Date = Date()) -> String {
        "[\(dateFormatter.string(from: date))] \(message)\n"
    }

    /// Append a line to the log. Safe to call from any thread/actor.
    func log(_ message: String) {
        let line = formattedLine(message)
        queue.async { [weak self] in
            FileHandle.standardError.write(Data(line.utf8))
            guard let self, let data = line.data(using: .utf8) else { return }
            self.handle?.write(data)
            self.rotateIfNeeded()
        }
    }

    /// Roll the log to hotfix.log.1 once it exceeds `maxBytes`. Runs on `queue`,
    /// so it is serialized with writes and needs no extra locking.
    private func rotateIfNeeded() {
        guard let url = fileURL, let handle,
              let size = try? handle.offset(), size > maxBytes else { return }

        try? handle.close()
        let backup = url.deletingLastPathComponent().appendingPathComponent("hotfix.log.1")
        let fm = FileManager.default
        try? fm.removeItem(at: backup)
        try? fm.moveItem(at: url, to: backup)
        fm.createFile(atPath: url.path, contents: nil)

        self.handle = try? FileHandle(forWritingTo: url)
        self.handle?.seekToEndOfFile()
    }

    /// Block until all pending asynchronous writes have completed. Intended for
    /// tests that need to read back what was just logged.
    func flush() {
        queue.sync {}
    }
}

/// Global convenience shorthand: `logf("started")`.
func logf(_ message: String) {
    Log.shared.log(message)
}
