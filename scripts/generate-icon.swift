#!/usr/bin/env swift
// Renders the flame.fill SF Symbol as app icon PNGs at all required sizes.
import AppKit

let iconsetPath = CommandLine.arguments.count > 1
    ? CommandLine.arguments[1]
    : "./icon/AppIcon.iconset"

let sizes: [(Int, String)] = [
    (16,   "icon_16x16.png"),
    (32,   "icon_16x16@2x.png"),
    (32,   "icon_32x32.png"),
    (64,   "icon_32x32@2x.png"),
    (128,  "icon_128x128.png"),
    (256,  "icon_128x128@2x.png"),
    (256,  "icon_256x256.png"),
    (512,  "icon_256x256@2x.png"),
    (512,  "icon_512x512.png"),
    (1024, "icon_512x512@2x.png"),
]

try FileManager.default.createDirectory(atPath: iconsetPath, withIntermediateDirectories: true)

for (size, filename) in sizes {
    let canvas = NSSize(width: size, height: size)
    let image = NSImage(size: canvas)

    image.lockFocus()

    let ctx = NSGraphicsContext.current!.cgContext

    // Dark rounded-rect background
    let radius = CGFloat(size) * 0.22
    let bgColor = NSColor(red: 0.078, green: 0.078, blue: 0.086, alpha: 1.0)  // #141416
    bgColor.setFill()
    let bgPath = NSBezierPath(roundedRect: NSRect(x: 0, y: 0, width: size, height: size),
                               xRadius: radius, yRadius: radius)
    bgPath.fill()

    // Flame SF Symbol, orange, centered
    let symbolSize = CGFloat(size) * 0.62
    let symbolConfig = NSImage.SymbolConfiguration(pointSize: symbolSize, weight: .regular)
    if let flame = NSImage(systemSymbolName: "flame.fill", accessibilityDescription: nil)?
        .withSymbolConfiguration(symbolConfig) {

        let accent = NSColor(red: 0.788, green: 0.275, blue: 0.118, alpha: 1.0) // #C9461E
        let tinted = flame.copy() as! NSImage
        tinted.lockFocus()
        accent.set()
        NSRect(x: 0, y: 0, width: tinted.size.width, height: tinted.size.height).fill(using: .sourceAtop)
        tinted.unlockFocus()

        let x = (CGFloat(size) - tinted.size.width) / 2
        let y = (CGFloat(size) - tinted.size.height) / 2
        tinted.draw(in: NSRect(x: x, y: y, width: tinted.size.width, height: tinted.size.height))
    }

    image.unlockFocus()

    // Write PNG
    guard let tiffData = image.tiffRepresentation,
          let bitmap = NSBitmapImageRep(data: tiffData),
          let png = bitmap.representation(using: .png, properties: [:]) else {
        print("Failed to render \(filename)")
        continue
    }

    let outPath = "\(iconsetPath)/\(filename)"
    try png.write(to: URL(fileURLWithPath: outPath))
    print("✓ \(filename) (\(size)px)")
}

print("Done. Run: iconutil -c icns \(iconsetPath)")
