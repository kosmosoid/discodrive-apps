import Foundation

// AES-CMAC (RFC 4493) — required for S2V in AES-SIV (vault name encryption).
enum AESCMAC {
    static func mac(key: [UInt8], _ message: [UInt8]) -> [UInt8] {
        let zero = [UInt8](repeating: 0, count: 16)
        // Subkeys K1, K2.
        let L = AESPrimitives.ecbEncrypt(key: key, zero)
        let K1 = gfDouble(L)
        let K2 = gfDouble(K1)

        let n = max(1, (message.count + 15) / 16)
        let lastComplete = message.count > 0 && message.count % 16 == 0

        // Last block: pad and XOR with K1 (complete block) or K2 (incomplete/empty).
        var lastBlock: [UInt8]
        let lastStart = (n - 1) * 16
        if lastComplete {
            lastBlock = xorBytes(Array(message[lastStart..<lastStart + 16]), K1)
        } else {
            var mLast = Array(message[lastStart..<message.count])
            mLast.append(0x80)
            while mLast.count < 16 { mLast.append(0x00) }
            lastBlock = xorBytes(mLast, K2)
        }

        // CBC-MAC pass.
        var x = zero
        for i in 0..<(n - 1) {
            let block = Array(message[(i * 16)..<(i * 16 + 16)])
            x = AESPrimitives.ecbEncrypt(key: key, xorBytes(x, block))
        }
        x = AESPrimitives.ecbEncrypt(key: key, xorBytes(x, lastBlock))
        return x
    }
}
