import SwiftUI

struct MenuBarPopoverView: View {
    @EnvironmentObject var monitor: ProcessMonitor
    @EnvironmentObject var prefs: PreferencesManager

    var body: some View {
        VStack(spacing: 0) {
            // Header
            VStack(spacing: 2) {
                HStack {
                    Image(systemName: "flame.fill")
                        .font(.system(size: 18, weight: .semibold))
                        .foregroundStyle(Color(hex: "C9461E"))
                    Text("Hotfix")
                        .font(.system(size: 18, weight: .light, design: .rounded))
                        .foregroundStyle(Color(hex: "141416"))
                    Spacer()
                    // Status indicator
                    statusIndicator
                }
                HStack {
                    Text("by BuildCraft Labs")
                        .font(.system(size: 10, weight: .medium, design: .rounded))
                        .foregroundStyle(Color(hex: "141416").opacity(0.45))
                    Spacer()
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 14)
            .padding(.bottom, 12)
            .background(Color(hex: "EDEAE5"))

            // Process list or empty state
            ScrollView {
                VStack(spacing: 0) {
                    if monitor.hotProcesses.isEmpty {
                        emptyStateView
                    } else {
                        hotProcessListView
                    }
                }
            }
            .frame(maxHeight: 220)
            .background(Color(hex: "F6F4F0"))

            // Footer actions
            HStack(spacing: 8) {
                Button(action: {
                    SettingsWindowController.shared.show()
                }) {
                    Label("Settings", systemImage: "gear")
                        .font(.system(size: 12, weight: .medium, design: .rounded))
                }
                .buttonStyle(PopoverButtonStyle(isPrimary: false))
                .focusable(false)

                Spacer()

                Button(action: {
                    NSApplication.shared.terminate(nil)
                }) {
                    Label("Quit", systemImage: "power")
                        .font(.system(size: 12, weight: .medium, design: .rounded))
                }
                .buttonStyle(PopoverButtonStyle(isPrimary: false))
                .focusable(false)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background(Color(hex: "EDEAE5"))
        }
        .frame(width: 300)
        .background(Color(hex: "F6F4F0"))
    }

    // MARK: - Status indicator
    @ViewBuilder
    private var statusIndicator: some View {
        HStack(spacing: 5) {
            Circle()
                .fill(prefs.isEnabled ? Color.green : Color(hex: "141416").opacity(0.25))
                .frame(width: 7, height: 7)
            Text(prefs.isEnabled ? "Watching" : "Paused")
                .font(.system(size: 11, weight: .medium, design: .rounded))
                .foregroundStyle(prefs.isEnabled ? Color.green : Color(hex: "141416").opacity(0.45))
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(
            Capsule()
                .fill(prefs.isEnabled ? Color.green.opacity(0.10) : Color(hex: "141416").opacity(0.06))
        )
    }

    // MARK: - Empty state
    @ViewBuilder
    private var emptyStateView: some View {
        VStack(spacing: 8) {
            Image(systemName: "thermometer.snowflake")
                .font(.system(size: 28, weight: .light))
                .foregroundStyle(Color(hex: "141416").opacity(0.25))
                .padding(.top, 24)
            Text("All cool.")
                .font(.system(size: 13, weight: .semibold, design: .rounded))
                .foregroundStyle(Color(hex: "141416").opacity(0.55))
            Text("No hot processes detected.")
                .font(.system(size: 11, design: .rounded))
                .foregroundStyle(Color(hex: "141416").opacity(0.35))
                .padding(.bottom, 24)
        }
        .frame(maxWidth: .infinity)
    }

    // MARK: - Hot process list
    @ViewBuilder
    private var hotProcessListView: some View {
        VStack(spacing: 0) {
            HStack {
                Text("HOT PROCESSES")
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .tracking(0.8)
                    .foregroundStyle(Color(hex: "141416").opacity(0.35))
                Spacer()
                Text("\(monitor.hotProcesses.count)")
                    .font(.system(size: 9, weight: .bold, design: .rounded))
                    .foregroundStyle(Color(hex: "C9461E"))
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(Capsule().fill(Color(hex: "C9461E").opacity(0.12)))
            }
            .padding(.horizontal, 14)
            .padding(.top, 10)
            .padding(.bottom, 6)

            ForEach(monitor.hotProcesses) { proc in
                HotProcessRow(process: proc)
                if proc.id != monitor.hotProcesses.last?.id {
                    Divider()
                        .padding(.horizontal, 14)
                        .overlay(Color(hex: "141416").opacity(0.07))
                }
            }
            .padding(.bottom, 8)
        }
    }
}

// MARK: - Hot process row
struct HotProcessRow: View {
    let process: HotProcess

    var body: some View {
        VStack(spacing: 5) {
            HStack(alignment: .center, spacing: 8) {
                VStack(alignment: .leading, spacing: 1) {
                    Text(process.name)
                        .font(.system(size: 12, weight: .semibold, design: .rounded))
                        .foregroundStyle(Color(hex: "141416"))
                        .lineLimit(1)
                    Text("PID \(process.pid)")
                        .font(.system(size: 10, design: .rounded))
                        .foregroundStyle(Color(hex: "141416").opacity(0.40))
                }
                Spacer()
                VStack(alignment: .trailing, spacing: 1) {
                    Text(String(format: "%.1f%%", process.cpuPercent))
                        .font(.system(size: 13, weight: .bold, design: .rounded).monospacedDigit())
                        .foregroundStyle(cpuColor(process.cpuPercent))
                    Text(hotDurationLabel(process.hotSeconds))
                        .font(.system(size: 9, design: .rounded))
                        .foregroundStyle(Color(hex: "141416").opacity(0.35))
                }
            }
            // Heat bar
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(Color(hex: "141416").opacity(0.08))
                        .frame(height: 4)
                    RoundedRectangle(cornerRadius: 3)
                        .fill(cpuColor(process.cpuPercent))
                        .frame(width: geo.size.width * min(process.cpuPercent / 100.0, 1.0), height: 4)
                }
            }
            .frame(height: 4)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .contentShape(Rectangle())
    }

    private func cpuColor(_ pct: Double) -> Color {
        if pct >= 90 { return Color(hex: "C9461E") }
        if pct >= 75 { return Color.orange }
        return Color.yellow
    }

    private func hotDurationLabel(_ seconds: Double) -> String {
        if seconds < 60 { return "hot for \(Int(seconds))s" }
        return "hot for \(Int(seconds / 60))m \(Int(seconds.truncatingRemainder(dividingBy: 60)))s"
    }
}

// MARK: - Button style
struct PopoverButtonStyle: ButtonStyle {
    let isPrimary: Bool

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .foregroundStyle(isPrimary ? Color.white : Color(hex: "141416").opacity(0.70))
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(
                RoundedRectangle(cornerRadius: 7)
                    .fill(isPrimary
                          ? Color(hex: "C9461E")
                          : Color(hex: "141416").opacity(configuration.isPressed ? 0.10 : 0.06))
            )
            .scaleEffect(configuration.isPressed ? 0.97 : 1.0)
            .animation(.easeInOut(duration: 0.1), value: configuration.isPressed)
    }
}
