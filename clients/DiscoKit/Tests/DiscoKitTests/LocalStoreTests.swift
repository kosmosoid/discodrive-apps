import XCTest
import GRDB
@testable import DiscoKit

final class LocalStoreTests: XCTestCase {
    func makeStore() throws -> (LocalStore, URL) {
        let dir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let store = try LocalStore(dbQueue: DatabaseQueue(), contentDir: dir)
        return (store, dir)
    }

    func tmpFile(_ s: String) throws -> URL {
        let u = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try Data(s.utf8).write(to: u); return u
    }

    func testStoreCachedThenStatusAndStale() throws {
        let (store, dir) = try makeStore()
        XCTAssertEqual(try store.status(nodeID: "n1", serverVersion: 1), .none)
        let f = try tmpFile("hello")
        try store.store(nodeID: "n1", version: 1, from: f, pinned: false, relPath: "/Папка/a.txt")
        XCTAssertEqual(try store.status(nodeID: "n1", serverVersion: 1), .cached)
        XCTAssertEqual(try store.status(nodeID: "n1", serverVersion: 2), .stale) // server version bumped
        let url = store.localURL(nodeID: "n1")!
        XCTAssertEqual(try String(contentsOf: url, encoding: .utf8), "hello")
        // mirrors the tree with the proper file name
        XCTAssertEqual(url, dir.appendingPathComponent("Папка/a.txt"))
        XCTAssertEqual(url.lastPathComponent, "a.txt")
    }

    func testPinAndEvictKeepsPinned() throws {
        let (store, _) = try makeStore()
        try store.store(nodeID: "c", version: 1, from: try tmpFile("c"), pinned: false, relPath: "c.txt")
        try store.store(nodeID: "p", version: 1, from: try tmpFile("p"), pinned: true, relPath: "sub/p.txt")
        try store.pin(nodeID: "c")
        try store.unpin(nodeID: "c") // back to cached
        try store.evictCached()
        XCTAssertEqual(try store.status(nodeID: "c", serverVersion: 1), .none)   // evicted
        XCTAssertEqual(try store.status(nodeID: "p", serverVersion: 1), .pinned) // retained
    }
}
