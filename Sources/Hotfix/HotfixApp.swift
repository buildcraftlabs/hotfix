import SwiftUI
import UserNotifications

@main
struct HotfixApp: App {
    @StateObject private var monitor = ProcessMonitor.shared
    @StateObject private var prefs = PreferencesManager.shared

    init() {
        // Report a crash from the previous run (if any), then arm crash capture.
        CrashReporter.reportPending()
        CrashReporter.install()

        logf("Hotfix starting (version \(UpdateChecker.currentVersion))")
        // A delegate is required for banners to appear while the app is active.
        // Without it, macOS silently routes notifications straight to Notification
        // Center whenever Hotfix is frontmost (e.g. the popover/settings is open).
        UNUserNotificationCenter.current().delegate = NotificationDelegate.shared
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

        // Silent background auto-update: check at launch, then every 6 hours.
        DispatchQueue.main.async {
            UpdateChecker.shared.startAutomaticUpdates()
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

/// Ensures kill notifications appear as on-screen banners — including when
/// Hotfix is the active app — instead of being delivered silently to
/// Notification Center.
final class NotificationDelegate: NSObject, UNUserNotificationCenterDelegate {
    static let shared = NotificationDelegate()

    func userNotificationCenter(_ center: UNUserNotificationCenter,
                                willPresent notification: UNNotification,
                                withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .list, .sound])
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
