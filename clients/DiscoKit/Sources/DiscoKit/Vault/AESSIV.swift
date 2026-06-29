import Foundation

// AES-SIV-CMAC (RFC 5297) for encrypting vault entry names. Port of daemon/internal/vault/name.go.
// Cryptomator nuance: S2V always feeds CMAC(AAD) even if AAD is empty (but not nil).
// aad == nil → S2V over plaintext only (used in DirIdHash).
enum AESSIV {
    enum SIVError: Error { case tooShort, authFailed }

    // S2V (RFC 5297): macKey is the CMAC key.
    static func s2v(macKey: [UInt8], aad: [UInt8]?, plaintext: [UInt8]) -> [UInt8] {
        var D = AESCMAC.mac(key: macKey, [UInt8](repeating: 0, count: 16))
        if let aad = aad {
            let aadMAC = AESCMAC.mac(key: macKey, aad)
            D = xorBytes(gfDouble(D), aadMAC)
        }
        if plaintext.count >= 16 {
            // xorend: XOR the last 16 bytes of plaintext with D, then CMAC the whole thing.
            var padded = plaintext
            let off = plaintext.count - 16
            for i in 0..<16 { padded[off + i] ^= D[i] }
            return AESCMAC.mac(key: macKey, padded)
        } else {
            // Pad with 0x80…, dbl(D), XOR, then CMAC.
            var padded = plaintext
            padded.append(0x80)
            while padded.count < 16 { padded.append(0x00) }
            padded = xorBytes(padded, gfDouble(D))
            return AESCMAC.mac(key: macKey, padded)
        }
    }

    // IV = tag with bits 31 and 63 cleared (RFC 5297).
    private static func ivFromTag(_ tag: [UInt8]) -> [UInt8] {
        var iv = tag
        iv[8] &= 0x7f
        iv[12] &= 0x7f
        return iv
    }

    static func encrypt(macKey: [UInt8], encKey: [UInt8], plaintext: [UInt8], aad: [UInt8]?) -> [UInt8] {
        let tag = s2v(macKey: macKey, aad: aad, plaintext: plaintext)
        let ct = AESPrimitives.ctr(key: encKey, iv: ivFromTag(tag), plaintext)
        return tag + ct
    }

    static func decrypt(macKey: [UInt8], encKey: [UInt8], ciphertext: [UInt8], aad: [UInt8]?) throws -> [UInt8] {
        guard ciphertext.count >= 16 else { throw SIVError.tooShort }
        let tag = Array(ciphertext[0..<16])
        let enc = Array(ciphertext[16...])
        let pt = AESPrimitives.ctr(key: encKey, iv: ivFromTag(tag), enc)
        let computed = s2v(macKey: macKey, aad: aad, plaintext: pt)
        guard constantTimeEqual(computed, tag) else { throw SIVError.authFailed }
        return pt
    }

    private static func constantTimeEqual(_ a: [UInt8], _ b: [UInt8]) -> Bool {
        guard a.count == b.count else { return false }
        var v: UInt8 = 0
        for i in 0..<a.count { v |= a[i] ^ b[i] }
        return v == 0
    }
}
