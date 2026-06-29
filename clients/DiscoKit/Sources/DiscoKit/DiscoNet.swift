import Foundation

public enum DiscoNet {
    // Opt-in to accepting a self-signed / unverified server certificate — for LOCAL
    // testing against a self-hosted server only. Default is false → strict system TLS
    // validation. Set this to true at app launch (e.g. under #if DEBUG) BEFORE the first
    // network request; it is read once when `session` is first created.
    nonisolated(unsafe) public static var allowInsecureTLS = false

    // Shared session used by all core requests.
    public static let session: URLSession = {
        if allowInsecureTLS {
            return URLSession(configuration: .default, delegate: ServerTrustDelegate(), delegateQueue: nil)
        }
        return URLSession(configuration: .default)
    }()
}

// INSECURE — accepts ANY server TLS certificate (no chain verification). Used only when
// DiscoNet.allowInsecureTLS is set; never opt in for production.
final class ServerTrustDelegate: NSObject, URLSessionDelegate, @unchecked Sendable {
    func urlSession(_ session: URLSession, didReceive challenge: URLAuthenticationChallenge,
                    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
        if challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
           let trust = challenge.protectionSpace.serverTrust {
            completionHandler(.useCredential, URLCredential(trust: trust))
        } else {
            completionHandler(.performDefaultHandling, nil)
        }
    }
}
