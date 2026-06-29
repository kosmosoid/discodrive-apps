import XCTest
@testable import DiscoKit

func hexBytes(_ s: String) -> [UInt8] {
    var out = [UInt8](); var idx = s.startIndex
    while idx < s.endIndex {
        let next = s.index(idx, offsetBy: 2)
        out.append(UInt8(s[idx..<next], radix: 16)!)
        idx = next
    }
    return out
}

final class VaultCryptoTests: XCTestCase {
    // RFC 4493 — official AES-CMAC test vectors (key 2b7e1516…).
    func testAESCMACRFC4493() {
        let k = hexBytes("2b7e151628aed2a6abf7158809cf4f3c")
        let msg = hexBytes("6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e5130c81c46a35ce411e5fbc1191a0a52eff69f2445df4f9b17ad2b417be66c3710")
        // Example 1: empty message.
        XCTAssertEqual(AESCMAC.mac(key: k, []), hexBytes("bb1d6929e95937287fa37d129b756746"))
        // Example 2: 16 bytes.
        XCTAssertEqual(AESCMAC.mac(key: k, Array(msg[0..<16])), hexBytes("070a16b46b4d4144f79bdd9dd04a287c"))
        // Example 3: 40 bytes.
        XCTAssertEqual(AESCMAC.mac(key: k, Array(msg[0..<40])), hexBytes("dfa66747de9ae63030ca32611497c827"))
        // Example 4: 64 bytes.
        XCTAssertEqual(AESCMAC.mac(key: k, Array(msg[0..<64])), hexBytes("51f0bebf7e3b9d92fc49741779363cfe"))
    }

    // RFC 5297 — deterministic AES-SIV example (one AD component).
    func testAESSIVRFC5297() throws {
        let macKey = hexBytes("fffefdfcfbfaf9f8f7f6f5f4f3f2f1f0")   // S2V key (left half)
        let encKey = hexBytes("f0f1f2f3f4f5f6f7f8f9fafbfcfdfeff")   // CTR key (right half)
        let ad = hexBytes("101112131415161718191a1b1c1d1e1f2021222324252627")
        let pt = hexBytes("112233445566778899aabbccddee")
        let want = hexBytes("85632d07c6e8f37f950acd320a2ecc9340c02b9690c4dc04daef7f6afe5c")
        let ct = AESSIV.encrypt(macKey: macKey, encKey: encKey, plaintext: pt, aad: ad)
        XCTAssertEqual(ct, want)
        XCTAssertEqual(try AESSIV.decrypt(macKey: macKey, encKey: encKey, ciphertext: want, aad: ad), pt)
        // Corrupt the tag → authentication failure.
        var bad = want; bad[0] ^= 1
        XCTAssertThrowsError(try AESSIV.decrypt(macKey: macKey, encKey: encKey, ciphertext: bad, aad: ad))
    }

    // RFC 7914 — scrypt("", "", N=16, r=1, p=1, dkLen=64) reference vector.
    func testScryptRFC7914() {
        let dk = Scrypt.derive(password: [], salt: [], n: 16, r: 1, p: 1, dkLen: 64)
        let want = hexBytes(
            "77d6576238657b203b19ca42c18a0497f16b4844e3074ae8dfdffa3fede21442" +
            "fcd0069ded0948f8326a753a0fc81f17e8d3e0fb2e0d3628cf35e20c38d18906")
        XCTAssertEqual(dk, want)
    }

    // RFC 3394 — 256-bit KEK wrapping 256-bit key data.
    func testAESKeyWrapRFC3394() throws {
        let kek = Array<UInt8>(0x00...0x1f)
        let pt: [UInt8] = [
            0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
            0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
            0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
            0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
        ]
        let wantCT: [UInt8] = [
            0x28, 0xc9, 0xf4, 0x04, 0xc4, 0xb8, 0x10, 0xf4,
            0xcb, 0xcc, 0xb3, 0x5c, 0xfb, 0x87, 0xf8, 0x26,
            0x3f, 0x57, 0x86, 0xe2, 0xd8, 0x0e, 0xd3, 0x26,
            0xcb, 0xc7, 0xf0, 0xe7, 0x1a, 0x99, 0xf4, 0x3b,
            0xfb, 0x98, 0x8b, 0x9b, 0x7a, 0x02, 0xdd, 0x21,
        ]
        XCTAssertEqual(try AESKeyWrap.wrap(kek: kek, plaintext: pt), wantCT)
        XCTAssertEqual(try AESKeyWrap.unwrap(kek: kek, wrapped: wantCT), pt)
        XCTAssertThrowsError(try AESKeyWrap.unwrap(kek: kek, wrapped: Array(repeating: 0, count: 40)))
    }
}
