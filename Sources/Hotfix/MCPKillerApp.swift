import SwiftUI
import UserNotifications

@main
struct HotfixApp: App {
    @StateObject private var monitor = ProcessMonitor.shared
    @StateObject private var prefs = PreferencesManager.shared

    init() {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { _, _ in }

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
