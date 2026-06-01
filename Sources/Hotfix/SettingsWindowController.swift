import AppKit
import SwiftUI

// Manages the settings window independently of the MenuBarExtra scene.
// Using NSWindowController avoids the unreliable showSettingsWindow: selector
// that breaks inside MenuBarExtra popovers.
final class SettingsWindowController: NSWindowController, NSWindowDelegate {
    static let shared = SettingsWindowController()

    private init() {
        let rootView = SettingsView()
            .environmentObject(ProcessMonitor.shared)
            .environmentObject(PreferencesManager.shared)
            .accentColor(Color(hex: "C9461E"))

        let hostingController = NSHostingController(rootView: rootView)
        let window = NSWindow(contentViewController: hostingController)
        window.title = "Hotfix Settings"
        window.styleMask = [.titled, .closable, .miniaturizable]
        window.setContentSize(NSSize(width: 640, height: 480))
        window.isReleasedWhenClosed = false
        window.center()

        super.init(window: window)
        window.delegate = self
    }

    required init?(coder: NSCoder) { fatalError() }

    func show() {
        NSApp.activate(ignoringOtherApps: true)
        window?.makeKeyAndOrderFront(nil)
    }
}
