import Foundation

// AES Key Wrap (RFC 3394). Port of daemon/internal/vault/keywrap.go.
enum AESKeyWrap {
    static let defaultIV: [UInt8] = Array(repeating: 0xA6, count: 8)

    enum WrapError: Error { case badLength, badIV }

    static func wrap(kek: [UInt8], plaintext: [UInt8]) throws -> [UInt8] {
        guard plaintext.count % 8 == 0 else { throw WrapError.badLength }
        let n = plaintext.count / 8
        var R: [[UInt8]] = [defaultIV]
        for i in 0..<n { R.append(Array(plaintext[(i*8)..<(i*8+8)])) }
        for j in 0..<6 {
            for i in 1...n {
                let buf = AESPrimitives.ecbEncrypt(key: kek, R[0] + R[i])
                let t = UInt64(n * j + i)
                let a = beUInt64(buf[0..<8]) ^ t
                R[0] = beBytes(a)
                R[i] = Array(buf[8..<16])
            }
        }
        var out = R[0]
        for i in 1...n { out += R[i] }
        return out
    }

    static func unwrap(kek: [UInt8], wrapped: [UInt8]) throws -> [UInt8] {
        guard wrapped.count % 8 == 0, wrapped.count >= 16 else { throw WrapError.badLength }
        let n = wrapped.count / 8 - 1
        var R: [[UInt8]] = [Array(wrapped[0..<8])]
        for i in 1...n { R.append(Array(wrapped[(i*8)..<(i*8+8)])) }
        for j in stride(from: 5, through: 0, by: -1) {
            for i in stride(from: n, through: 1, by: -1) {
                let t = UInt64(n * j + i)
                let a = beUInt64(R[0][0..<8]) ^ t
                let buf = AESPrimitives.ecbDecrypt(key: kek, beBytes(a) + R[i])
                R[0] = Array(buf[0..<8])
                R[i] = Array(buf[8..<16])
            }
        }
        guard R[0] == defaultIV else { throw WrapError.badIV }
        var out = [UInt8]()
        for i in 1...n { out += R[i] }
        return out
    }
}
