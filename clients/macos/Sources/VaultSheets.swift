import SwiftUI
import AppKit
import DiscoKit

// Vault unlock: biometrics / password / recovery key.
struct UnlockVaultView: View {
    @EnvironmentObject var app: AppState
    let folder: Node
    @State private var password = ""
    @State private var recovery = ""
    @State private var useRecovery = false
    @State private var remember = true

    private var biometryName: String? { app.vaultBiometry.displayName }

    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: "lock.fill").font(.largeTitle).foregroundStyle(.orange)
            Text(folder.name).font(.headline)
            if useRecovery {
                TextEditor(text: $recovery)
                    .font(.system(.body, design: .monospaced))
                    .frame(width: 330, height: 90)
                    .overlay(RoundedRectangle(cornerRadius: 6).stroke(.secondary.opacity(0.4)))
            } else {
                if let bn = biometryName, app.vaultHasSavedPassword(folder) {
                    Button {
                        Task { await app.openVaultBiometric(folder) }
                    } label: {
                        Label(String(format: app.t("vault.unlockWith"), bn), systemImage: app.vaultBiometry.sfSymbol)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(app.vaultUnlocking)
                    Text(app.t("vault.password")).font(.caption).foregroundStyle(.secondary)
                }
                SecureField(app.t("vault.password"), text: $password)
                    .textFieldStyle(.roundedBorder).frame(width: 300)
                    .onSubmit { unlock() }
                    .disabled(app.vaultUnlocking)
                if biometryName != nil {
                    Toggle(app.t("vault.remember"), isOn: $remember).frame(width: 300).disabled(app.vaultUnlocking)
                }
            }
            if app.vaultUnlocking {
                HStack(spacing: 8) { ProgressView().controlSize(.small); Text(app.t("vault.unlocking")).foregroundStyle(.secondary) }
                    .font(.callout)
            } else if let e = app.vaultUnlockError {
                Text(e).foregroundStyle(.red).font(.caption)
            }
            Button(useRecovery ? app.t("vault.usePassword") : app.t("vault.useRecovery")) {
                useRecovery.toggle(); app.vaultUnlockError = nil
            }.buttonStyle(.link).disabled(app.vaultUnlocking)
            HStack {
                Button(app.t("dialog.cancel")) { app.vaultUnlockFolder = nil; app.vaultUnlockError = nil }
                    .disabled(app.vaultUnlocking)
                Button(app.t("vault.unlock")) { unlock() }
                    .disabled(app.vaultUnlocking || (useRecovery ? recovery.isEmpty : password.isEmpty))
                    .keyboardShortcut(.defaultAction)
            }
        }
        .padding(30).frame(width: 390)
        .task {
            // Biometry available and password saved → prompt Touch ID immediately.
            if app.vaultBiometry != .none, app.vaultHasSavedPassword(folder) {
                await app.openVaultBiometric(folder)
            }
        }
    }

    private func unlock() {
        Task {
            if useRecovery { await app.openVaultWithRecovery(folder, phrase: recovery) }
            else { await app.openVault(folder, password: password, remember: remember) }
        }
    }
}

// New vault creation.
struct CreateVaultView: View {
    @EnvironmentObject var app: AppState
    let parentPath: String
    @Binding var isPresented: Bool
    @State private var name = ""
    @State private var password = ""

    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: "lock.rectangle").font(.largeTitle).foregroundStyle(.orange)
            Text(app.t("vault.createVault")).font(.headline)
            TextField(app.t("vault.name"), text: $name).textFieldStyle(.roundedBorder).frame(width: 300)
            SecureField(app.t("vault.password"), text: $password).textFieldStyle(.roundedBorder).frame(width: 300)
            HStack {
                Button(app.t("dialog.cancel")) { isPresented = false }
                Button(app.t("dialog.create")) {
                    let n = name, p = password, pp = parentPath
                    isPresented = false
                    Task { await app.createVault(name: n, inFolderPath: pp, password: p) }
                }
                .disabled(name.isEmpty || password.isEmpty)
                .keyboardShortcut(.defaultAction)
            }
        }
        .padding(30).frame(width: 380)
    }
}

// Show the recovery key of a newly created vault.
struct RecoveryKeyView: View {
    @EnvironmentObject var app: AppState
    let phrase: String

    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: "key.fill").font(.largeTitle).foregroundStyle(.orange)
            Text(app.t("vault.recoveryTitle")).font(.headline)
            Text(app.t("vault.recoveryHint")).font(.caption).foregroundStyle(.secondary)
                .multilineTextAlignment(.center).frame(width: 420)
            ScrollView {
                Text(phrase).textSelection(.enabled)
                    .font(.system(.body, design: .monospaced)).padding(8)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            .frame(width: 430, height: 120)
            .overlay(RoundedRectangle(cornerRadius: 6).stroke(.secondary.opacity(0.4)))
            HStack {
                Button(app.t("vault.copy")) {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(phrase, forType: .string)
                }
                Button(app.t("vault.recoverySaved")) { app.vaultRecoveryToShow = nil }
                    .keyboardShortcut(.defaultAction)
            }
        }
        .padding(28).frame(width: 470)
    }
}
