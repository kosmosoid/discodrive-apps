import XCTest
@testable import DiscoKit

final class PairingTests: XCTestCase {
    func testPairInitThenPollApproved() async throws {
        var polls = 0
        MockURLProtocol.handler = { req in
            switch req.url!.path {
            case "/pair/init":
                return (201, [:], Data(#"{"device_code":"DC","user_code":"AB-CD","verification_uri":"https://x.test/pair","interval":1,"expires_in":300}"#.utf8))
            case "/pair/token":
                polls += 1
                if polls == 1 { return (200, [:], Data(#"{"status":"pending"}"#.utf8)) }
                return (200, [:], Data(#"{"status":"approved","device_token":"DEVTOK"}"#.utf8))
            default: return (404, [:], Data())
            }
        }
        let p = Pairing(baseURL: URL(string: "https://x.test")!, session: MockURLProtocol.session())
        let info = try await p.start(deviceName: "Mac")
        XCTAssertEqual(info.userCode, "AB-CD")
        let token = try await p.poll(deviceCode: info.deviceCode, interval: .milliseconds(1))
        XCTAssertEqual(token, "DEVTOK")
    }
}
