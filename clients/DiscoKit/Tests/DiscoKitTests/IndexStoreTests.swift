import XCTest
import GRDB
@testable import DiscoKit

final class IndexStoreTests: XCTestCase {
    func makeStore() throws -> IndexStore { try IndexStore(dbQueue: DatabaseQueue()) }

    func testApplyPutThenReadNode() throws {
        let store = try makeStore()
        let ch = RemoteChange(seq: 1, op: "put", nodeID: "n1", path: "/docs/a.txt",
                              isDir: false, version: 5, contentHash: "h", size: 10, deleted: false)
        try store.apply([ch])
        let node = try store.node(id: "n1")
        XCTAssertEqual(node?.name, "a.txt")
        XCTAssertEqual(node?.version, 5)
        XCTAssertNil(node?.parentID) // /docs parent was not in the batch → nil
    }

    func testChildrenByParentNodeID() throws {
        let store = try makeStore()
        try store.apply([
            RemoteChange(seq: 1, op: "put", nodeID: "dir", path: "/docs", isDir: true,
                         version: 1, contentHash: "", size: 0, deleted: false),
            RemoteChange(seq: 2, op: "put", nodeID: "f1", path: "/docs/a.txt", isDir: false,
                         version: 1, contentHash: "", size: 3, deleted: false),
            RemoteChange(seq: 3, op: "put", nodeID: "root1", path: "/top.txt", isDir: false,
                         version: 1, contentHash: "", size: 1, deleted: false),
        ])
        XCTAssertEqual(try store.children(of: "dir").map(\.id), ["f1"])
        XCTAssertEqual(Set(try store.children(of: nil).map(\.id)), ["dir", "root1"])
    }

    func testDeleteSubtreeAndCursor() throws {
        let store = try makeStore()
        try store.apply([
            RemoteChange(seq: 1, op: "put", nodeID: "dir", path: "/d", isDir: true, version: 1, contentHash: "", size: 0, deleted: false),
            RemoteChange(seq: 2, op: "put", nodeID: "f1", path: "/d/a", isDir: false, version: 1, contentHash: "", size: 1, deleted: false),
        ])
        try store.setCursor(2)
        XCTAssertEqual(try store.cursor(), 2)
        try store.apply([
            RemoteChange(seq: 3, op: "del", nodeID: "dir", path: "/d", isDir: true, version: 2, contentHash: "", size: 0, deleted: true),
        ])
        XCTAssertNil(try store.node(id: "dir"))
        XCTAssertNil(try store.node(id: "f1"))
    }
}
