import Foundation
import CommonCrypto

// Low-level AES operations via CommonCrypto — needed for AES Key Wrap, CMAC, and SIV
// (CryptoKit doesn't expose these). Only the primitives missing from CryptoKit; GCM/HMAC/SHA come from CryptoKit.
enum AESPrimitives {
    // Encrypt/decrypt a single 16-byte block in ECB mode without padding.
    static func ecb(_ op: Int, key: [UInt8], _ inBlock: [UInt8]) -> [UInt8] {
        precondition(inBlock.count == 16)
        var out = [UInt8](repeating: 0, count: 16)
        var moved = 0
        let status = CCCrypt(CCOperation(op), CCAlgorithm(kCCAlgorithmAES), CCOptions(kCCOptionECBMode),
                             key, key.count, nil, inBlock, 16, &out, 16, &moved)
        precondition(status == kCCSuccess && moved == 16, "AES-ECB block failed: \(status)")
        return out
    }

    static func ecbEncrypt(key: [UInt8], _ block: [UInt8]) -> [UInt8] { ecb(kCCEncrypt, key: key, block) }
    static func ecbDecrypt(key: [UInt8], _ block: [UInt8]) -> [UInt8] { ecb(kCCDecrypt, key: key, block) }

    // AES-CTR keystream (used by AES-SIV). iv = 16-byte initial counter (big-endian).
    static func ctr(key: [UInt8], iv: [UInt8], _ data: [UInt8]) -> [UInt8] {
        precondition(iv.count == 16)
        var cryptor: CCCryptorRef?
        let create = CCCryptorCreateWithMode(
            CCOperation(kCCEncrypt), CCMode(kCCModeCTR), CCAlgorithm(kCCAlgorithmAES),
            CCPadding(ccNoPadding), iv, key, key.count, nil, 0, 0,
            CCModeOptions(kCCModeOptionCTR_BE), &cryptor)
        precondition(create == kCCSuccess, "CCCryptorCreate CTR failed: \(create)")
        defer { if let c = cryptor { CCCryptorRelease(c) } }
        var out = [UInt8](repeating: 0, count: data.count)
        var moved = 0
        CCCryptorUpdate(cryptor, data, data.count, &out, out.count, &moved)
        return out
    }
}

// Big-endian helpers for 64-bit words (used by AES Key Wrap).
@inline(__always) func beUInt64(_ b: ArraySlice<UInt8>) -> UInt64 {
    var v: UInt64 = 0
    for byte in b { v = (v << 8) | UInt64(byte) }
    return v
}
@inline(__always) func beBytes(_ v: UInt64) -> [UInt8] {
    var out = [UInt8](repeating: 0, count: 8)
    var x = v
    for i in (0..<8).reversed() { out[i] = UInt8(x & 0xff); x >>= 8 }
    return out
}

// Doubling in GF(2^128), polynomial x^128 + x^7 + x^2 + x + 1 (for CMAC subkeys and SIV S2V).
func gfDouble(_ b: [UInt8]) -> [UInt8] {
    var out = [UInt8](repeating: 0, count: 16)
    var carry: UInt8 = 0
    for i in stride(from: 15, through: 0, by: -1) {
        let newCarry = (b[i] >> 7) & 1
        out[i] = (b[i] << 1) | carry
        carry = newCarry
    }
    if carry != 0 { out[15] ^= 0x87 }
    return out
}

func xorBytes(_ a: [UInt8], _ b: [UInt8]) -> [UInt8] {
    zip(a, b).map { $0 ^ $1 }
}
