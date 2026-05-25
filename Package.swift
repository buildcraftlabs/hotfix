// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "Hotfix",
    platforms: [
        .macOS(.v13)
    ],
    targets: [
        .executableTarget(
            name: "Hotfix",
            path: "Sources/Hotfix"
        )
    ]
)
