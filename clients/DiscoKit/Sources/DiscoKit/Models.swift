import Foundation

public struct RemoteChange: Decodable, Sendable {
    public let seq: Int64
    public let op: String
    public let nodeID: String
    public let path: String
    public let isDir: Bool
    public let version: Int64
    public let contentHash: String
    public let size: Int64
    public let deleted: Bool

    enum CodingKeys: String, CodingKey {
        case seq, op, path, version, size, deleted
        case nodeID = "node_id"
        case isDir = "is_dir"
        case contentHash = "content_hash"
    }

    public init(seq: Int64, op: String, nodeID: String, path: String, isDir: Bool,
                version: Int64, contentHash: String, size: Int64, deleted: Bool) {
        self.seq = seq; self.op = op; self.nodeID = nodeID; self.path = path
        self.isDir = isDir; self.version = version; self.contentHash = contentHash
        self.size = size; self.deleted = deleted
    }
}

public struct ChangesPage: Decodable, Sendable {
    public let changes: [RemoteChange]
    public let cursor: Int64
    public let hasMore: Bool
    enum CodingKeys: String, CodingKey {
        case changes, cursor
        case hasMore = "has_more"
    }
}

public struct Node: Equatable, Hashable, Sendable, Identifiable {
    public var id: String
    public var parentID: String?
    public var name: String
    public var isDir: Bool
    public var version: Int64
    public var contentHash: String
    public var size: Int64
    public var path: String   // full path from the root, e.g. "/Folder/file.txt"
}
