import SwiftUI
import DiscoKit

@main
struct DiscoDriveApp: App {
    @StateObject private var app = AppState()

    init() {
        #if DEBUG
        // Debug builds may talk to a self-hosted server with a self-signed cert.
        // Release builds keep strict TLS validation.
        DiscoNet.allowInsecureTLS = true
        #endif
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if app.paired { BrowserView() } else { PairingView() }
            }
            .environmentObject(app)
            .onAppear { app.bootstrap() }
        }
    }
}
