// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DiscoKit",
    platforms: [.macOS(.v14), .iOS(.v17)],
    products: [.library(name: "DiscoKit", targets: ["DiscoKit"])],
    dependencies: [
        .package(url: "https://github.com/groue/GRDB.swift", from: "6.29.0"),
    ],
    targets: [
        .target(name: "DiscoKit", dependencies: [.product(name: "GRDB", package: "GRDB.swift")],
                resources: [.copy("Resources/4096words_en.txt")]),
        .testTarget(name: "DiscoKitTests", dependencies: ["DiscoKit"],
                    resources: [.copy("Fixtures")]),
    ]
)
