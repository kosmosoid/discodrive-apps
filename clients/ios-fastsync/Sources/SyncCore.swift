import Foundation
import Kfmobile

// Thin wrapper over the gomobile-generated Kfmobile API. Free functions take an NSError pointer
// (not Swift throws); MobileClient methods throw. Every call blocks — invoke off the main thread.
enum SyncError: Error { case unknown }

enum SyncCore {
    static func pairBegin(server: String, name: String, kind: String, insecure: Bool) throws -> MobilePairing {
        var err: NSError?
        guard let p = MobilePairBegin(server, name, kind, insecure, &err) else { throw err ?? SyncError.unknown }
        return p
    }

    static func pairAwait(server: String, deviceCode: String, intervalSec: Int, insecure: Bool) throws -> String {
        var err: NSError?
        let token = MobilePairAwait(server, deviceCode, intervalSec, insecure, &err)
        if let err { throw err }
        return token
    }

    static func newClient(server: String, token: String, syncDir: String, dbPath: String, insecure: Bool) throws -> MobileClient {
        var err: NSError?
        guard let c = MobileNew(server, token, syncDir, dbPath, insecure, &err) else { throw err ?? SyncError.unknown }
        return c
    }
}
