import Foundation
import CryptoKit

// scrypt (RFC 7914) for vault KEK derivation. Salsa20/8 + BlockMix + ROMix + PBKDF2-HMAC-SHA256.
enum Scrypt {
    static func derive(password: [UInt8], salt: [UInt8], n: Int, r: Int, p: Int, dkLen: Int) -> [UInt8] {
        let mfLen = 128 * r
        var b = pbkdf2C1(password: password, salt: salt, dkLen: p * mfLen)
        for i in 0..<p {
            var block = Array(b[(i * mfLen)..<((i + 1) * mfLen)])
            roMix(&block, n: n, r: r)
            b.replaceSubrange((i * mfLen)..<((i + 1) * mfLen), with: block)
        }
        return pbkdf2C1(password: password, salt: b, dkLen: dkLen)
    }

    // PBKDF2-HMAC-SHA256 with iteration count c=1 (as required by scrypt): T_i = HMAC(P, S ‖ BE32(i)).
    static func pbkdf2C1(password: [UInt8], salt: [UInt8], dkLen: Int) -> [UInt8] {
        let key = SymmetricKey(data: Data(password))
        var out = [UInt8](); out.reserveCapacity(dkLen)
        var i: UInt32 = 1
        while out.count < dkLen {
            var msg = salt
            msg.append(contentsOf: [UInt8((i >> 24) & 0xff), UInt8((i >> 16) & 0xff),
                                    UInt8((i >> 8) & 0xff), UInt8(i & 0xff)])
            out.append(contentsOf: HMAC<SHA256>.authenticationCode(for: msg, using: key))
            i += 1
        }
        return Array(out[0..<dkLen])
    }

    @inline(__always) private static func rotl(_ a: UInt32, _ b: UInt32) -> UInt32 {
        (a << b) | (a >> (32 &- b))
    }

    // Salsa20/8 core over a 64-byte block (in place).
    static func salsa20_8(_ input: inout [UInt8]) {
        var x = [UInt32](repeating: 0, count: 16)
        for i in 0..<16 {
            x[i] = UInt32(input[i*4]) | (UInt32(input[i*4+1]) << 8)
                 | (UInt32(input[i*4+2]) << 16) | (UInt32(input[i*4+3]) << 24)
        }
        let orig = x
        for _ in 0..<4 {
            x[ 4] ^= rotl(x[ 0] &+ x[12], 7);  x[ 8] ^= rotl(x[ 4] &+ x[ 0], 9)
            x[12] ^= rotl(x[ 8] &+ x[ 4],13);  x[ 0] ^= rotl(x[12] &+ x[ 8],18)
            x[ 9] ^= rotl(x[ 5] &+ x[ 1], 7);  x[13] ^= rotl(x[ 9] &+ x[ 5], 9)
            x[ 1] ^= rotl(x[13] &+ x[ 9],13);  x[ 5] ^= rotl(x[ 1] &+ x[13],18)
            x[14] ^= rotl(x[10] &+ x[ 6], 7);  x[ 2] ^= rotl(x[14] &+ x[10], 9)
            x[ 6] ^= rotl(x[ 2] &+ x[14],13);  x[10] ^= rotl(x[ 6] &+ x[ 2],18)
            x[ 3] ^= rotl(x[15] &+ x[11], 7);  x[ 7] ^= rotl(x[ 3] &+ x[15], 9)
            x[11] ^= rotl(x[ 7] &+ x[ 3],13);  x[15] ^= rotl(x[11] &+ x[ 7],18)
            x[ 1] ^= rotl(x[ 0] &+ x[ 3], 7);  x[ 2] ^= rotl(x[ 1] &+ x[ 0], 9)
            x[ 3] ^= rotl(x[ 2] &+ x[ 1],13);  x[ 0] ^= rotl(x[ 3] &+ x[ 2],18)
            x[ 6] ^= rotl(x[ 5] &+ x[ 4], 7);  x[ 7] ^= rotl(x[ 6] &+ x[ 5], 9)
            x[ 4] ^= rotl(x[ 7] &+ x[ 6],13);  x[ 5] ^= rotl(x[ 4] &+ x[ 7],18)
            x[11] ^= rotl(x[10] &+ x[ 9], 7);  x[ 8] ^= rotl(x[11] &+ x[10], 9)
            x[ 9] ^= rotl(x[ 8] &+ x[11],13);  x[10] ^= rotl(x[ 9] &+ x[ 8],18)
            x[12] ^= rotl(x[15] &+ x[14], 7);  x[13] ^= rotl(x[12] &+ x[15], 9)
            x[14] ^= rotl(x[13] &+ x[12],13);  x[15] ^= rotl(x[14] &+ x[13],18)
        }
        for i in 0..<16 {
            let v = x[i] &+ orig[i]
            input[i*4] = UInt8(v & 0xff); input[i*4+1] = UInt8((v >> 8) & 0xff)
            input[i*4+2] = UInt8((v >> 16) & 0xff); input[i*4+3] = UInt8((v >> 24) & 0xff)
        }
    }

    // BlockMix: B (128r bytes = 2r blocks of 64) → out of the same length. No allocations in the loop.
    static func blockMixInto(_ out: inout [UInt8], _ b: [UInt8], r: Int) {
        var x = [UInt8](repeating: 0, count: 64)
        let lastOff = (2*r - 1) * 64
        for k in 0..<64 { x[k] = b[lastOff + k] }
        for i in 0..<(2*r) {
            let off = i * 64
            for k in 0..<64 { x[k] ^= b[off + k] }
            salsa20_8(&x)
            let destOff = ((i % 2 == 0) ? (i / 2) : (r + i / 2)) * 64
            for k in 0..<64 { out[destOff + k] = x[k] }
        }
    }

    // ROMix: N*128r bytes of memory, two passes. Index-based lookups, buffers are reused.
    static func roMix(_ b: inout [UInt8], n: Int, r: Int) {
        let blockLen = 128 * r
        var v = [UInt8](repeating: 0, count: n * blockLen)
        var x = b
        var tmp = [UInt8](repeating: 0, count: blockLen)
        for i in 0..<n {
            let vOff = i * blockLen
            for k in 0..<blockLen { v[vOff + k] = x[k] }
            blockMixInto(&tmp, x, r: r)
            swap(&x, &tmp)
        }
        for _ in 0..<n {
            let off = (2*r - 1) * 64
            var j: UInt64 = 0
            for k in 0..<8 { j |= UInt64(x[off + k]) << (8 * k) }
            let vOff = Int(j % UInt64(n)) * blockLen
            for k in 0..<blockLen { x[k] ^= v[vOff + k] }
            blockMixInto(&tmp, x, r: r)
            swap(&x, &tmp)
        }
        b = x
    }
}
