import Foundation
import AppKit

class UpdateChecker {
    static let shared = UpdateChecker()
    static let currentVersion = "1.0.2"

    private let releasesURL = URL(string: "https://api.github.com/repos/buildcraftlabs/hotfix/releases/latest")!
    private let downloadURL = URL(string: "https://github.com/buildcraftlabs/hotfix/releases/latest")!

    private init() {}

    func checkForUpdates(userInitiated: Bool = false) {
        var request = URLRequest(url: releasesURL)
        request.setValue("application/vnd.github+json", forHTTPHeaderField: "Accept")
        request.setValue("Hotfix/\(Self.currentVersion)", forHTTPHeaderField: "User-Agent")
        request.timeoutInterval = 10

        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            DispatchQueue.main.async {
                self?.handleResponse(data: data, error: error, userInitiated: userInitiated)
            }
        }.resume()
    }

    private func handleResponse(data: Data?, error: Error?, userInitiated: Bool) {
        if let error = error {
            if userInitiated {
                showAlert(
                    title: "Update Check Failed",
                    message: "Could not reach GitHub: \(error.localizedDescription)",
                    style: .warning
                )
            }
            return
        }

        guard let data = data else {
            if userInitiated {
                showAlert(title: "Update Check Failed", message: "No data received from GitHub.", style: .warning)
            }
            return
        }

        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let tagName = json["tag_name"] as? String else {
            if userInitiated {
                showAlert(title: "Update Check Failed", message: "Could not parse GitHub response.", style: .warning)
            }
            return
        }

        let remoteVersion = tagName.hasPrefix("v") ? String(tagName.dropFirst()) : tagName

        if isNewerVersion(remoteVersion, than: Self.currentVersion) {
            let alert = NSAlert()
            alert.messageText = "Update Available"
            alert.informativeText = "Hotfix \(remoteVersion) is available. You have \(Self.currentVersion)."
            alert.alertStyle = .informational
            alert.addButton(withTitle: "Download Update")
            alert.addButton(withTitle: "Later")

            NSApp.activate(ignoringOtherApps: true)
            let response = alert.runModal()
            if response == .alertFirstButtonReturn {
                NSWorkspace.shared.open(downloadURL)
            }
        } else if userInitiated {
            showAlert(
                title: "You're Up to Date",
                message: "Hotfix \(Self.currentVersion) is the latest version.",
                style: .informational
            )
        }
    }

    private func isNewerVersion(_ remote: String, than current: String) -> Bool {
        let remoteComponents = remote.split(separator: ".").compactMap { Int($0) }
        let currentComponents = current.split(separator: ".").compactMap { Int($0) }

        let maxLen = max(remoteComponents.count, currentComponents.count)
        for i in 0..<maxLen {
            let r = i < remoteComponents.count ? remoteComponents[i] : 0
            let c = i < currentComponents.count ? currentComponents[i] : 0
            if r > c { return true }
            if r < c { return false }
        }
        return false
    }

    private func showAlert(title: String, message: String, style: NSAlert.Style) {
        NSApp.activate(ignoringOtherApps: true)
        let alert = NSAlert()
        alert.messageText = title
        alert.informativeText = message
        alert.alertStyle = style
        alert.addButton(withTitle: "OK")
        alert.runModal()
    }
}
