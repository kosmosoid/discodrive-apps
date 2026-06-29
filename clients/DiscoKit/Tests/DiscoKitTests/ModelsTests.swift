import XCTest
@testable import DiscoKit

final class ModelsTests: XCTestCase {
    func testDecodeChangesPage() throws {
        let json = """
        {"changes":[
          {"seq":1,"op":"put","node_id":"n1","path":"/foo.txt","is_dir":false,"version":3,"content_hash":"abc","size":10,"deleted":false}
        ],"cursor":1,"has_more":false}
        """.data(using: .utf8)!
        let page = try JSONDecoder().decode(ChangesPage.self, from: json)
        XCTAssertEqual(page.cursor, 1)
        XCTAssertFalse(page.hasMore)
        XCTAssertEqual(page.changes.first?.nodeID, "n1")
        XCTAssertEqual(page.changes.first?.path, "/foo.txt")
        XCTAssertEqual(page.changes.first?.version, 3)
    }
}
