import SwiftUI
import UIKit
import QuickLook
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
        NavigationStack {
            VStack(spacing: 16) {
                Image(systemName: "lock.fill").font(.system(size: 44)).foregroundStyle(.orange)
                Text(folder.name).font(.headline)
                if useRecovery {
                    TextEditor(text: $recovery).font(.system(.body, design: .monospaced))
                        .frame(height: 110).overlay(RoundedRectangle(cornerRadius: 8).stroke(.secondary.opacity(0.4)))
                        .autocorrectionDisabled().textInputAutocapitalization(.never)
                } else {
                    if let bn = biometryName, app.vaultHasSavedPassword(folder) {
                        Button { Task { await app.openVaultBiometric(folder) } } label: {
                            Label(String(format: app.t("vault.unlockWith"), bn), systemImage: app.vaultBiometry.sfSymbol)
                        }.buttonStyle(.borderedProminent).disabled(app.vaultUnlocking)
                    }
                    SecureField(app.t("vault.password"), text: $password).textFieldStyle(.roundedBorder)
                        .disabled(app.vaultUnlocking)
                    if biometryName != nil { Toggle(app.t("vault.remember"), isOn: $remember).disabled(app.vaultUnlocking) }
                }
                if app.vaultUnlocking {
                    HStack(spacing: 8) { ProgressView(); Text(app.t("vault.unlocking")).foregroundStyle(.secondary) }
                        .font(.callout)
                } else if let e = app.vaultUnlockError {
                    Text(e).foregroundStyle(.red).font(.caption)
                }
                Button(useRecovery ? app.t("vault.usePassword") : app.t("vault.useRecovery")) {
                    useRecovery.toggle(); app.vaultUnlockError = nil
                }.font(.callout).disabled(app.vaultUnlocking)
                Button(app.t("vault.unlock")) { unlock() }
                    .buttonStyle(.bordered)
                    .disabled(app.vaultUnlocking || (useRecovery ? recovery.isEmpty : password.isEmpty))
                Spacer()
            }
            .padding()
            .toolbar { ToolbarItem(placement: .topBarLeading) {
                Button(app.t("dialog.cancel")) { app.vaultUnlockFolder = nil; app.vaultUnlockError = nil }
            } }
            .task {
                if app.vaultBiometry != .none, app.vaultHasSavedPassword(folder) {
                    await app.openVaultBiometric(folder)
                }
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

struct CreateVaultView: View {
    @EnvironmentObject var app: AppState
    let parentPath: String
    @Binding var isPresented: Bool
    @State private var name = ""
    @State private var password = ""

    var body: some View {
        NavigationStack {
            Form {
                TextField(app.t("vault.name"), text: $name)
                SecureField(app.t("vault.password"), text: $password)
            }
            .navigationTitle(app.t("vault.createVault"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) { Button(app.t("dialog.cancel")) { isPresented = false } }
                ToolbarItem(placement: .topBarTrailing) {
                    Button(app.t("dialog.create")) {
                        let n = name, p = password, pp = parentPath
                        isPresented = false
                        Task { await app.createVault(name: n, inFolderPath: pp, password: p) }
                    }.disabled(name.isEmpty || password.isEmpty)
                }
            }
        }
    }
}

struct RecoveryKeyView: View {
    @EnvironmentObject var app: AppState
    let phrase: String

    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Image(systemName: "key.fill").font(.system(size: 44)).foregroundStyle(.orange)
                Text(app.t("vault.recoveryHint")).font(.callout).foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                ScrollView {
                    Text(phrase).textSelection(.enabled)
                        .font(.system(.body, design: .monospaced)).frame(maxWidth: .infinity, alignment: .leading).padding(8)
                }.overlay(RoundedRectangle(cornerRadius: 8).stroke(.secondary.opacity(0.4)))
                Button(app.t("vault.copy")) { UIPasteboard.general.string = phrase }
                Spacer()
            }
            .padding()
            .navigationTitle(app.t("vault.recoveryTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar { ToolbarItem(placement: .topBarTrailing) {
                Button(app.t("vault.recoverySaved")) { app.vaultRecoveryToShow = nil }
            } }
        }
    }
}

// Decrypted vault browser.
struct VaultBrowserView: View {
    @EnvironmentObject var app: AppState
    let session: AppState.VaultSession

    @State private var stack: [(dirID: String, name: String)] = [("", "")]
    @State private var entries: [VaultEntry] = []
    @State private var loading = false
    @State private var error: String?
    @State private var newFolderPresented = false
    @State private var newFolderName = ""
    @State private var renameTarget: VaultEntry?
    @State private var renameName = ""
    @State private var importing = false
    @State private var previewURL: URL?

    private var currentDirID: String { stack.last!.dirID }
    private var title: String { session.name + stack.dropFirst().map { " / " + $0.name }.joined() }

    var body: some View {
        NavigationStack {
            List(entries, id: \.name) { e in
                Button {
                    if e.isDir, let id = e.dirID { stack.append((id, e.name)); Task { await load() } }
                    else { Task { await openFile(e) } }
                } label: {
                    Label(e.name, systemImage: e.isDir ? "folder" : "doc").foregroundStyle(.primary)
                }
                .swipeActions(edge: .trailing) {
                    Button(role: .destructive) { Task { await deleteEntry(e) } } label: { Label(app.t("menu.delete"), systemImage: "trash") }
                    Button { renameName = e.name; renameTarget = e } label: { Label(app.t("menu.rename"), systemImage: "pencil") }.tint(.blue)
                }
            }
            .overlay {
                if loading { ProgressView() }
                else if entries.isEmpty { Text(app.t("browse.empty")).foregroundStyle(.secondary) }
            }
            .navigationTitle(title).navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    if stack.count > 1 {
                        Button { stack.removeLast(); Task { await load() } } label: { Image(systemName: "chevron.left") }
                    } else {
                        Button(app.t("vault.close")) { app.closeVault() }
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button { newFolderName = ""; newFolderPresented = true } label: { Label(app.t("toolbar.newFolder"), systemImage: "folder.badge.plus") }
                        Button { importing = true } label: { Label(app.t("vault.addFile"), systemImage: "doc.badge.plus") }
                    } label: { Image(systemName: "plus") }
                }
            }
            .alert(app.t("toolbar.newFolder"), isPresented: $newFolderPresented) {
                TextField(app.t("dialog.folderName"), text: $newFolderName)
                Button(app.t("dialog.create")) { let n = newFolderName; Task { await makeFolder(n) } }
                Button(app.t("dialog.cancel"), role: .cancel) {}
            }
            .alert(app.t("menu.rename"), isPresented: Binding(get: { renameTarget != nil }, set: { if !$0 { renameTarget = nil } })) {
                TextField(app.t("dialog.newName"), text: $renameName)
                Button(app.t("menu.rename")) { if let e = renameTarget { let nm = renameName; Task { await renameEntry(e, nm) } } }
                Button(app.t("dialog.cancel"), role: .cancel) {}
            }
            .fileImporter(isPresented: $importing, allowedContentTypes: [.item], allowsMultipleSelection: true) { result in
                guard case .success(let urls) = result else { return }
                let accessed = urls.filter { $0.startAccessingSecurityScopedResource() }
                Task { await upload(accessed); accessed.forEach { $0.stopAccessingSecurityScopedResource() } }
            }
            .quickLookPreview(Binding(get: { previewURL }, set: { previewURL = $0 }))
            .overlay(alignment: .bottom) { if let error { Text(error).foregroundStyle(.red).font(.caption).padding(6) } }
        }
        .task { await load() }
    }

    private func load() async {
        loading = true; error = nil
        do {
            let list = try await session.vault.listEntries(dirID: currentDirID, source: session.io)
            entries = list.sorted { ($0.isDir ? 0 : 1, $0.name) < ($1.isDir ? 0 : 1, $1.name) }
        } catch { self.error = error.localizedDescription }
        loading = false
    }
    private func openFile(_ e: VaultEntry) async {
        guard let cp = e.contentPath else { return }
        do {
            let data = try await session.vault.decryptFile(at: cp, source: session.io)
            let tmp = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString, isDirectory: true)
            try FileManager.default.createDirectory(at: tmp, withIntermediateDirectories: true)
            let url = tmp.appendingPathComponent(e.name)
            try data.write(to: url)
            previewURL = url
        } catch { self.error = error.localizedDescription }
    }
    private func upload(_ urls: [URL]) async {
        for url in urls {
            if let data = try? Data(contentsOf: url) {
                try? await session.vault.addFile(name: url.lastPathComponent, data: data, parentDirID: currentDirID, sink: session.io)
            }
        }
        await app.refresh(); await load()
    }
    private func makeFolder(_ name: String) async {
        guard !name.isEmpty else { return }
        _ = try? await session.vault.createFolder(name: name, parentDirID: currentDirID, sink: session.io)
        await app.refresh(); await load()
    }
    private func deleteEntry(_ e: VaultEntry) async {
        try? await session.vault.deleteEntry(e, source: session.io, sink: session.io)
        await app.refresh(); await load()
    }
    private func renameEntry(_ e: VaultEntry, _ newName: String) async {
        guard !newName.isEmpty, newName != e.name else { return }
        try? await session.vault.renameEntry(e, to: newName, parentDirID: currentDirID, source: session.io, sink: session.io)
        await app.refresh(); await load()
    }
}
