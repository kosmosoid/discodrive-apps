import Foundation
import CryptoKit
import Security

// File content encryption (AES-GCM, Cryptomator SIV_GCM). Port of content.go.
extension Vault {
    static let chunkPlainSize = 32 * 1024
    static let headerTotalSize = 68   // nonce(12) + ct(40) + tag(16)

    public func encryptContent(_ plaintext: Data) throws -> Data {
        let headerNonce = AES.GCM.Nonce()                 // random 12 bytes
        let contentKey = Vault.randomBytes(32)
        let headerPayload = Data([UInt8](repeating: 0xFF, count: 8) + contentKey)
        let headerBox = try AES.GCM.seal(headerPayload, using: SymmetricKey(data: Data(encKey)), nonce: headerNonce)
        var out = headerBox.combined!                     // nonce(12) + ct(40) + tag(16) = 68

        let ckKey = SymmetricKey(data: Data(contentKey))
        let headerNonceData = Data(headerNonce)
        var idx: UInt64 = 0
        var offset = 0
        while offset < plaintext.count {
            let end = min(offset + Vault.chunkPlainSize, plaintext.count)
            let chunk = plaintext.subdata(in: offset..<end)
            var aad = Data(beBytes(idx)); aad.append(headerNonceData)
            let box = try AES.GCM.seal(chunk, using: ckKey, nonce: AES.GCM.Nonce(), authenticating: aad)
            out.append(box.combined!)
            idx += 1; offset = end
        }
        return out
    }

    public func decryptContent(_ data: Data) throws -> Data {
        guard data.count >= Vault.headerTotalSize else { throw VaultError.badVaultFile }
        let header = data.subdata(in: 0..<Vault.headerTotalSize)
        let payload = try AES.GCM.open(AES.GCM.SealedBox(combined: header), using: SymmetricKey(data: Data(encKey)))
        guard payload.count == 40 else { throw VaultError.badVaultFile }
        let contentKey = payload.subdata(in: 8..<40)
        let headerNonce = header.subdata(in: 0..<12)
        let ckKey = SymmetricKey(data: contentKey)

        var out = Data()
        let fullFrame = 12 + Vault.chunkPlainSize + 16
        var idx: UInt64 = 0
        var offset = Vault.headerTotalSize
        while offset < data.count {
            let end = min(offset + fullFrame, data.count)
            let frame = data.subdata(in: offset..<end)
            var aad = Data(beBytes(idx)); aad.append(headerNonce)
            let pt = try AES.GCM.open(AES.GCM.SealedBox(combined: frame), using: ckKey, authenticating: aad)
            out.append(pt)
            idx += 1; offset = end
        }
        return out
    }

    static func randomBytes(_ n: Int) -> [UInt8] {
        var b = [UInt8](repeating: 0, count: n)
        _ = b.withUnsafeMutableBytes { SecRandomCopyBytes(kSecRandomDefault, n, $0.baseAddress!) }
        return b
    }
}
