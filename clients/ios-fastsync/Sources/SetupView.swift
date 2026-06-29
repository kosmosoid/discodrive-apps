import SwiftUI

struct SetupView: View {
    @EnvironmentObject var model: AppModel
    @Environment(\.openURL) private var openURL
    @State private var server = "https://"
    @State private var insecure = false

    var body: some View {
        VStack(spacing: 18) {
            Image(systemName: "arrow.triangle.2.circlepath").font(.system(size: 52)).foregroundStyle(.tint)
            Text("DiscoDrive FastSync").font(.title2.bold())
            TextField("https://files.example.com", text: $server)
                .textFieldStyle(.roundedBorder)
                .autocorrectionDisabled().textInputAutocapitalization(.never).keyboardType(.URL)
            Toggle("Self-signed certificate", isOn: $insecure)
            if let code = model.pendingUserCode {
                Text("Code: \(code)").font(.headline)
                Text("Confirm this code in the browser to pair.")
                    .foregroundStyle(.secondary).multilineTextAlignment(.center)
                ProgressView()
            } else {
                Button(model.working ? "…" : "Pair device") { Task { await pair() } }
                    .buttonStyle(.borderedProminent)
                    .disabled(model.working || URL(string: server) == nil)
            }
            if let e = model.lastError { Text(e).foregroundStyle(.red).font(.caption) }
        }
        .padding(28)
    }

    private func pair() async {
        guard let p = await model.startPairing(server: server, insecure: insecure) else { return }
        if let u = URL(string: p.verificationURL) { openURL(u) }
        await model.finishPairing(server: server, pairing: p, insecure: insecure)
    }
}
