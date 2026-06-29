import SwiftUI
import AppKit
import DiscoKit

struct PairingView: View {
    @EnvironmentObject var app: AppState
    @State private var serverString = "https://"
    @State private var info: PairingInfo?
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        VStack(spacing: 16) {
            Text(app.t("pair.title")).font(.title2)
            TextField("https://files.example.com", text: $serverString)
                .textFieldStyle(.roundedBorder).frame(width: 360)
            if let info {
                Text("\(app.t("pair.deviceCode")): \(info.userCode)").font(.headline)
                Text(app.t("pair.confirmHint"))
                    .multilineTextAlignment(.center).foregroundStyle(.secondary)
                ProgressView()
            } else {
                Button(busy ? "…" : app.t("pair.connect")) { Task { await connect() } }
                    .disabled(busy || URL(string: serverString) == nil)
            }
            if let error { Text(error).foregroundStyle(.red) }
        }
        .padding(40)
    }

    private func connect() async {
        guard let url = URL(string: serverString) else { return }
        busy = true; error = nil
        do {
            let info = try await app.startPairing(serverURL: url)
            self.info = info
            // verification_uri may be relative (e.g. /app/pair?code=…) —
            // resolve it against the server base URL.
            if let v = URL(string: info.verificationURI, relativeTo: url)?.absoluteURL {
                NSWorkspace.shared.open(v)
            }
            try await app.confirmPairing(serverURL: url, info: info)
        } catch { self.error = error.localizedDescription; self.info = nil }
        busy = false
    }
}
