import SwiftUI
import UserNotifications

@main
struct HotfixApp: App {
    @StateObject private var monitor = ProcessMonitor.shared
    @StateObject private var prefs = PreferencesManager.shared

    init() {
        logf("Hotfix starting (version \(UpdateChecker.currentVersion))")
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
            if let error {
                logf("notification authorization error: \(error.localizedDescription)")
            } else {
                logf("notification authorization \(granted ? "granted" : "denied")")
            }
        }

        if PreferencesManager.shared.isEnabled {
            DispatchQueue.main.async {
                ProcessMonitor.shared.start()
            }
        }
    }

    var body: some Scene {
        MenuBarExtra {
            MenuBarPopoverView()
                .environmentObject(monitor)
                .environmentObject(prefs)
                .accentColor(Color(hex: "C9461E"))
        } label: {
            MenuBarLabel(isKilling: monitor.isKilling)
        }
        .menuBarExtraStyle(.window)
    }
}

struct MenuBarLabel: View {
    let isKilling: Bool

    var body: some View {
        Image(systemName: isKilling ? "flame.fill" : "flame")
            .symbolRenderingMode(.hierarchical)
            .foregroundStyle(isKilling ? Color(hex: "C9461E") : .primary)
    }
}
