import SwiftUI
import QuickLook
import DiscoKit

// Root: NavigationStack + global actions and sheets.
// URL wrapper with a fresh id on each presentation — so .sheet re-triggers even for the same file.
struct PreviewItem: Identifiable { let id = UUID(); let url: URL }

// QuickLook via a UIKit controller: reliably opens any local file, including repeated opens.
struct QuickLookView: UIViewControllerRepresentable {
    let url: URL
    func makeUIViewController(context: Context) -> QLPreviewController {
        let c = QLPreviewController(); c.dataSource = context.coordinator; return c
    }
    func updateUIViewController(_ controller: QLPreviewController, context: Context) {}
    func makeCoordinator() -> Coordinator { Coordinator(url: url) }
    final class Coordinator: NSObject, QLPreviewControllerDataSource {
        let url: URL
        init(url: URL) { self.url = url }
        func numberOfPreviewItems(in controller: QLPreviewController) -> Int { 1 }
        func previewController(_ controller: QLPreviewController, previewItemAt index: Int) -> QLPreviewItem { url as NSURL }
    }
}

struct BrowserView: View {
    @EnvironmentObject var app: AppState
    @Environment(\.scenePhase) private var scenePhase
    @State private var settingsPresented = false
    @State private var preview: PreviewItem?

    var body: some View {
        NavigationStack {
            FolderView(folder: nil)
                .navigationDestination(for: Node.self) { node in FolderView(folder: node) }
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        Menu {
                            Button { Task { await app.refresh() } } label: { Label(app.t("toolbar.refresh"), systemImage: "arrow.clockwise") }
                            Button { Task { await app.importLocalFiles() } } label: { Label(app.t("toolbar.import"), systemImage: "square.and.arrow.down.on.square") }
                            Button { settingsPresented = true } label: { Label(app.t("settings.title"), systemImage: "gear") }
                            Button { try? app.local?.evictCached() } label: { Label(app.t("toolbar.free"), systemImage: "trash") }
                            Divider()
                            Button(role: .destructive) { app.logout() } label: { Label(app.t("toolbar.logout"), systemImage: "rectangle.portrait.and.arrow.right") }
                        } label: { Image(systemName: "ellipsis.circle") }
                    }
                }
        }
        .task { await app.refresh(); await app.importLocalFiles() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active { Task { await app.importLocalFiles() } }
        }
        .sheet(isPresented: $settingsPresented) { SettingsView() }
        .sheet(item: Binding(get: { app.vaultUnlockFolder },
                             set: { app.vaultUnlockFolder = $0; if $0 == nil { app.vaultUnlockError = nil } })) { folder in
            UnlockVaultView(folder: folder).environmentObject(app)
        }
        .sheet(isPresented: Binding(get: { app.vaultSession != nil }, set: { if !$0 { app.closeVault() } })) {
            if let s = app.vaultSession { VaultBrowserView(session: s).environmentObject(app) }
        }
        .sheet(isPresented: Binding(get: { app.vaultRecoveryToShow != nil }, set: { if !$0 { app.vaultRecoveryToShow = nil } })) {
            if let p = app.vaultRecoveryToShow { RecoveryKeyView(phrase: p).environmentObject(app) }
        }
        .onChange(of: app.fileToPreview) { _, url in
            if let url { preview = PreviewItem(url: url); app.fileToPreview = nil }
        }
        .sheet(item: $preview) { item in QuickLookView(url: item.url).ignoresSafeArea() }
    }
}

// Contents of a single folder (nil = root).
struct FolderView: View {
    @EnvironmentObject var app: AppState
    let folder: Node?
    @State private var newFolderPresented = false
    @State private var newFolderName = ""
    @State private var createVaultPresented = false
    @State private var importing = false
    @State private var renameTarget: Node?
    @State private var renameName = ""

    private var folderID: String? { folder?.id }
    private var folderPath: String { folder?.path ?? "" }

    var body: some View {
        List {
            ForEach(app.children(of: folderID)) { node in row(node) }
        }
        .navigationTitle(folder?.name ?? "DiscoDrive")
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await app.refresh(); await app.importLocalFiles() }
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Menu {
                    Button { newFolderName = ""; newFolderPresented = true } label: { Label(app.t("toolbar.newFolder"), systemImage: "folder.badge.plus") }
                    Button { importing = true } label: { Label(app.t("toolbar.addFile"), systemImage: "doc.badge.plus") }
                    Button { createVaultPresented = true } label: { Label(app.t("vault.createVault"), systemImage: "lock.rectangle") }
                } label: { Image(systemName: "plus") }
            }
        }
        .alert(app.t("toolbar.newFolder"), isPresented: $newFolderPresented) {
            TextField(app.t("dialog.folderName"), text: $newFolderName)
            Button(app.t("dialog.create")) { let n = newFolderName, p = folderPath; Task { await app.createFolder(name: n, inFolderPath: p) } }
            Button(app.t("dialog.cancel"), role: .cancel) {}
        }
        .alert(app.t("menu.rename"), isPresented: Binding(get: { renameTarget != nil }, set: { if !$0 { renameTarget = nil } })) {
            TextField(app.t("dialog.newName"), text: $renameName)
            Button(app.t("menu.rename")) { if let n = renameTarget { let nm = renameName; Task { await app.renameNode(n, to: nm) } } }
            Button(app.t("dialog.cancel"), role: .cancel) {}
        }
        .sheet(isPresented: $createVaultPresented) {
            CreateVaultView(parentPath: folderPath, isPresented: $createVaultPresented).environmentObject(app)
        }
        .fileImporter(isPresented: $importing, allowedContentTypes: [.item], allowsMultipleSelection: true) { result in
            guard case .success(let urls) = result else { return }
            let accessed = urls.filter { $0.startAccessingSecurityScopedResource() }
            let p = folderPath
            Task {
                await app.upload(accessed, toFolderPath: p)
                accessed.forEach { $0.stopAccessingSecurityScopedResource() }
            }
        }
    }

    @ViewBuilder private func row(_ node: Node) -> some View {
        if node.isDir {
            if app.isVault(node) {
                Button { app.vaultUnlockFolder = node } label: {
                    Label(node.name, systemImage: "lock.fill").foregroundStyle(.primary)
                }
            } else {
                NavigationLink(value: node) { Label(node.name, systemImage: "folder") }
            }
        } else {
            Button { Task { await app.openFile(node) } } label: {
                HStack {
                    Image(systemName: "doc")
                    Text(node.name).foregroundStyle(.primary).lineLimit(1)
                    Spacer()
                    Text(ByteCountFormatter.string(fromByteCount: node.size, countStyle: .file))
                        .font(.caption).foregroundStyle(.secondary)
                    if app.isDownloading(node) {
                        ProgressView()
                    } else {
                        statusIcon(app.status(of: node))
                    }
                }
            }
            .swipeActions(edge: .trailing) {
                Button(role: .destructive) { Task { await app.deleteNode(node) } } label: { Label(app.t("menu.delete"), systemImage: "trash") }
                Button { renameName = node.name; renameTarget = node } label: { Label(app.t("menu.rename"), systemImage: "pencil") }.tint(.blue)
            }
            .contextMenu {
                Button(app.t("menu.keepLocal")) { Task { await app.pin(node) } }
                Button(app.t("menu.removeLocal")) { app.removeLocal(node) }
            }
        }
    }
}

@ViewBuilder func statusIcon(_ s: LocalStatus) -> some View {
    switch s {
    case .none:   Image(systemName: "icloud").foregroundStyle(.secondary)
    case .cached: Image(systemName: "checkmark.circle").foregroundStyle(.blue)
    case .pinned: Image(systemName: "pin.fill").foregroundStyle(.orange)
    case .stale:  Image(systemName: "exclamationmark.icloud").foregroundStyle(.yellow)
    }
}
