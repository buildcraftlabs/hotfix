import Foundation
import SwiftUI

class PreferencesManager: ObservableObject {
    static let shared = PreferencesManager()

    @AppStorage("isEnabled") var isEnabled: Bool = true
    @AppStorage("cpuThreshold") var cpuThreshold: Double = 80.0
    @AppStorage("killDuration") var killDuration: Double = 60.0
    @AppStorage("killOnSleep") var killOnSleep: Bool = true

    private let whitelistKey = "processWhitelist"

    private let defaultWhitelist: [String] = [
        "Xcode", "swift", "clang", "swiftc", "node", "python3"
    ]

    @Published var whitelist: [String] = []

    private init() {
        loadWhitelist()
    }

    // MARK: - Whitelist persistence
    private func loadWhitelist() {
        if let data = UserDefaults.standard.data(forKey: whitelistKey),
           let decoded = try? JSONDecoder().decode([String].self, from: data) {
            whitelist = decoded
        } else {
            // First run — set defaults
            whitelist = defaultWhitelist
            saveWhitelist()
        }
    }

    private func saveWhitelist() {
        if let encoded = try? JSONEncoder().encode(whitelist) {
            UserDefaults.standard.set(encoded, forKey: whitelistKey)
        }
    }

    func addToWhitelist(_ name: String) {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !whitelist.contains(trimmed) else { return }
        whitelist.append(trimmed)
        saveWhitelist()
    }

    func removeFromWhitelist(_ name: String) {
        whitelist.removeAll { $0 == name }
        saveWhitelist()
    }

    func isWhitelisted(_ name: String) -> Bool {
        whitelist.contains(name)
    }
}
