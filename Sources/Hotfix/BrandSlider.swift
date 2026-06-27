import SwiftUI
import AppKit

/// A horizontal slider whose filled track is rendered in the app's brand color.
///
/// SwiftUI's `Slider` ignores `.tint()` for the filled track on macOS — it always
/// draws the system control accent color. Wrapping `NSSlider` lets us set the
/// documented `trackFillColor`, so the fill reliably matches the brand color.
struct BrandSlider: NSViewRepresentable {
    @Binding var value: Double
    let range: ClosedRange<Double>
    let step: Double
    var fillColor: Color = Color(hex: "C9461E")

    func makeNSView(context: Context) -> NSSlider {
        let slider = NSSlider(value: value,
                              minValue: range.lowerBound,
                              maxValue: range.upperBound,
                              target: context.coordinator,
                              action: #selector(Coordinator.valueChanged(_:)))
        slider.isContinuous = true
        slider.controlSize = .regular
        slider.trackFillColor = NSColor(fillColor)
        return slider
    }

    func updateNSView(_ slider: NSSlider, context: Context) {
        context.coordinator.parent = self
        slider.minValue = range.lowerBound
        slider.maxValue = range.upperBound
        slider.trackFillColor = NSColor(fillColor)
        if slider.doubleValue != value {
            slider.doubleValue = value
        }
    }

    func makeCoordinator() -> Coordinator { Coordinator(self) }

    final class Coordinator: NSObject {
        var parent: BrandSlider
        init(_ parent: BrandSlider) { self.parent = parent }

        @objc func valueChanged(_ sender: NSSlider) {
            // Snap to the nearest step so behavior matches SwiftUI's stepped Slider.
            let lower = parent.range.lowerBound
            let snapped = (parent.step > 0)
                ? lower + (round((sender.doubleValue - lower) / parent.step) * parent.step)
                : sender.doubleValue
            let clamped = min(max(snapped, parent.range.lowerBound), parent.range.upperBound)
            if parent.value != clamped {
                parent.value = clamped
            }
        }
    }
}
