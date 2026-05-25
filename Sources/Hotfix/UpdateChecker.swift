import Foundation
import AppKit

class UpdateChecker {
    static let shared = UpdateChecker()
    static let currentVersion = "1.0.2"

    private let releasesURL = URL(string: "https://api.github.com/repos/buildcraftlabs/hotfix/releases/latest")!
    private let releasesPageURL = URL(string: "https://github.com/buildcraftlabs/hotfix/releases/latest")!

    private init() {}

    func checkForUpdates(userInitiated: Bool = false) {
        var request = URLRequest(url: releasesURL)
        request.setValue("application/vnd.github+json", forHTTPHeaderField: "Accept")
        request.setValue("Hotfix/\(Self.currentVersion)", forHTTPHeaderField: "User-Agent")
        request.timeoutInterval = 10

        URLSession.shared.dataTask(with: request) { [weak self] data, _, error in
            DispatchQueue.main.async {
                self?.handleResponse(data: data, error: error, userInitiated: userInitiated)
            }
        }.resume()
    }

    private func handleResponse(data: Data?, error: Error?, userInitiated: Bool) {
        if let error = error {
            if userInitiated {
                showAlert(title: "Update Check Failed",
                          message: "Could not reach GitHub: \(error.localizedDescription)",
                          style: .warning)
            }
            return
        }

        guard let data = data,
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let tagName = json["tag_name"] as? String else {
            if userInitiated {
                showAlert(title: "Update Check Failed", message: "Could not parse GitHub response.", style: .warning)
            }
            return
        }

        let remoteVersion = tagName.hasPrefix("v") ? String(tagName.dropFirst()) : tagName

        if isNewerVersion(remoteVersion, than: Self.currentVersion) {
            let dmgURL = (json["assets"] as? [[String: Any]])?
                .first(where: { ($0["name"] as? String)?.hasSuffix(".dmg") == true })
                .flatMap { $0["browser_download_url"] as? String }
                .flatMap { URL(string: $0) }

            let alert = NSAlert()
            alert.messageText = "Update Available"
            alert.informativeText = "Hotfix \(remoteVersion) is available (you have \(Self.currentVersion)).\n\nInstall in the background and relaunch?"
            alert.alertStyle = .informational
            alert.addButton(withTitle: dmgURL != nil ? "Install Now" : "Download Update")
            alert.addButton(withTitle: "Later")

            NSApp.activate(ignoringOtherApps: true)
            if alert.runModal() == .alertFirstButtonReturn {
                if let url = dmgURL {
                    downloadAndInstall(from: url, version: remoteVersion)
                } else {
                    NSWorkspace.shared.open(releasesPageURL)
                }
            }
        } else if userInitiated {
            showAlert(title: "You're Up to Date",
                      message: "Hotfix \(Self.currentVersion) is the latest version.",
                      style: .informational)
        }
    }

    private func downloadAndInstall(from url: URL, version: String) {
        showAlert(title: "Downloading Update",
                  message: "Hotfix \(version) is downloading. The app will relaunch automatically when the update is ready.",
                  style: .informational)

        URLSession.shared.downloadTask(with: url) { [weak self] tempURL, _, error in
            DispatchQueue.main.async {
                guard let self = self else { return }
                if let error = error {
                    self.showAlert(title: "Download Failed",
                                   message: error.localizedDescription,
                                   style: .warning)
                    return
                }
                guard let tempURL = tempURL else { return }
                self.installDMG(at: tempURL, version: version)
            }
        }.resume()
    }

    private func installDMG(at dmgURL: URL, version: String) {
        let tmp = NSTemporaryDirectory()
        let stableDMG = URL(fileURLWithPath: tmp).appendingPathComponent("Hotfix_\(version).dmg")
        let stagedApp = URL(fileURLWithPath: tmp).appendingPathComponent("Hotfix_update.app")

        try? FileManager.default.removeItem(at: stableDMG)
        do {
            try FileManager.default.moveItem(at: dmgURL, to: stableDMG)
        } catch {
            showAlert(title: "Install Failed",
                      message: "Could not stage download: \(error.localizedDescription)",
                      style: .critical)
            return
        }

        // Mount DMG
        let mount = Process()
        mount.executableURL = URL(fileURLWithPath: "/usr/bin/hdiutil")
        mount.arguments = ["attach", stableDMG.path, "-nobrowse", "-quiet"]
        mount.standardOutput = Pipe()
        mount.standardError = Pipe()
        do { try mount.run() } catch {
            showAlert(title: "Install Failed", message: "Could not mount update.", style: .critical)
            return
        }
        mount.waitUntilExit()

        // Copy app to staging area (not yet replacing the running bundle)
        try? FileManager.default.removeItem(at: stagedApp)
        let copy = Process()
        copy.executableURL = URL(fileURLWithPath: "/bin/cp")
        copy.arguments = ["-R", "/Volumes/Hotfix/Hotfix.app", stagedApp.path]
        copy.standardOutput = Pipe()
        copy.standardError = Pipe()
        do { try copy.run() } catch {}
        copy.waitUntilExit()

        // Detach and clean up DMG
        let detach = Process()
        detach.executableURL = URL(fileURLWithPath: "/usr/bin/hdiutil")
        detach.arguments = ["detach", "/Volumes/Hotfix", "-quiet", "-force"]
        try? detach.run()
        detach.waitUntilExit()
        try? FileManager.default.removeItem(at: stableDMG)

        guard copy.terminationStatus == 0 else {
            showAlert(title: "Install Failed", message: "Could not copy update.", style: .critical)
            return
        }

        // Write a shell script that replaces /Applications/Hotfix.app after we quit
        let scriptPath = URL(fileURLWithPath: tmp).appendingPathComponent("hotfix_update.sh")
        let script = """
        #!/bin/bash
        sleep 2
        rm -rf /Applications/Hotfix.app
        cp -R "\(stagedApp.path)" /Applications/Hotfix.app
        xattr -rd com.apple.quarantine /Applications/Hotfix.app 2>/dev/null
        open /Applications/Hotfix.app
        rm -rf "\(stagedApp.path)"
        rm -- "$0"
        """
        try? script.write(to: scriptPath, atomically: true, encoding: .utf8)
        try? FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: scriptPath.path)

        // Launch the script in the background then quit
        let launcher = Process()
        launcher.executableURL = URL(fileURLWithPath: "/bin/bash")
        launcher.arguments = [scriptPath.path]
        try? launcher.run()

        NSApp.terminate(nil)
    }

    private func isNewerVersion(_ remote: String, than current: String) -> Bool {
        let rv = remote.split(separator: ".").compactMap { Int($0) }
        let cv = current.split(separator: ".").compactMap { Int($0) }
        let maxLen = max(rv.count, cv.count)
        for i in 0..<maxLen {
            let r = i < rv.count ? rv[i] : 0
            let c = i < cv.count ? cv[i] : 0
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
