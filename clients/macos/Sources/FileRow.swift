import SwiftUI
import DiscoKit

struct FileRow: View {
    let node: Node
    let status: LocalStatus

    var body: some View {
        HStack {
            Image(systemName: node.isDir ? "folder" : "doc")
            Text(node.name)
            Spacer()
            if !node.isDir {
                Text(ByteCountFormatter.string(fromByteCount: node.size, countStyle: .file))
                    .foregroundStyle(.secondary).font(.caption)
                statusIcon
            }
        }
    }

    @ViewBuilder private var statusIcon: some View {
        switch status {
        case .none:   Image(systemName: "icloud").foregroundStyle(.secondary)
        case .cached: Image(systemName: "checkmark.circle").foregroundStyle(.blue)
        case .pinned: Image(systemName: "pin.fill").foregroundStyle(.orange)
        case .stale:  Image(systemName: "exclamationmark.icloud").foregroundStyle(.yellow)
        }
    }
}
