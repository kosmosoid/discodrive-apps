import Foundation

// Cryptomator-compatible recovery key. Port of recovery.go.
// payload = encKey(32) ‖ macKey(32) ‖ CRC32-IEEE-LE-low16(2) = 66 bytes → 44 words (4096-word dictionary).
enum RecoveryWords {
    static let list: [String] = {
        guard let url = Bundle.module.url(forResource: "4096words_en", withExtension: "txt"),
              let content = try? String(contentsOf: url, encoding: .utf8) else { return [] }
        return content.split(whereSeparator: \.isNewline)
            .map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
    }()
    static let index: [String: Int] = {
        var m = [String: Int](minimumCapacity: 4096)
        for (i, w) in list.enumerated() { m[w] = i }
        return m
    }()
}

func crc32IEEE(_ data: [UInt8]) -> UInt32 {
    var crc: UInt32 = 0xFFFFFFFF
    for b in data {
        crc ^= UInt32(b)
        for _ in 0..<8 { crc = (crc & 1) != 0 ? (crc >> 1) ^ 0xEDB88320 : (crc >> 1) }
    }
    return ~crc
}

func wordEncode(_ data: [UInt8]) -> String {
    precondition(data.count % 3 == 0)
    var out = [String]()
    var i = 0
    while i < data.count {
        let b1 = Int(data[i]), b2 = Int(data[i+1]), b3 = Int(data[i+2])
        out.append(RecoveryWords.list[(b1 << 4) | (b2 >> 4)])
        out.append(RecoveryWords.list[((b2 & 0x0F) << 8) | b3])
        i += 3
    }
    return out.joined(separator: " ")
}

func wordDecode(_ phrase: String) throws -> [UInt8] {
    let words = phrase.split(whereSeparator: \.isWhitespace).map(String.init)
    guard !words.isEmpty, words.count % 2 == 0 else { throw Vault.VaultError.badVaultFile }
    var out = [UInt8](repeating: 0, count: words.count / 2 * 3)
    var i = 0
    while i < words.count {
        guard let w1 = RecoveryWords.index[words[i]], let w2 = RecoveryWords.index[words[i+1]] else {
            throw Vault.VaultError.badVaultFile
        }
        let j = i / 2 * 3
        out[j] = UInt8(0xFF & (w1 >> 4))
        out[j+1] = UInt8((0xF0 & (w1 << 4)) | (0x0F & (w2 >> 8)))
        out[j+2] = UInt8(0xFF & w2)
        i += 2
    }
    return out
}

extension Vault {
    // Returns the vault's 44-word recovery phrase.
    public func recoveryKey() -> String {
        var raw = encKey + macKey
        let crc = crc32IEEE(raw)
        raw.append(UInt8(crc & 0xff))
        raw.append(UInt8((crc >> 8) & 0xff))
        return wordEncode(raw)
    }

    public static func keysFromRecovery(_ phrase: String) throws -> (encKey: [UInt8], macKey: [UInt8]) {
        let payload = try wordDecode(phrase)
        guard payload.count == 66 else { throw VaultError.badVaultFile }
        let raw = Array(payload[0..<64])
        let crc = crc32IEEE(raw)
        guard payload[64] == UInt8(crc & 0xff), payload[65] == UInt8((crc >> 8) & 0xff) else {
            throw VaultError.wrongPassword   // corrupted or invalid phrase
        }
        return (Array(raw[0..<32]), Array(raw[32..<64]))
    }

    // Open a vault using a recovery phrase instead of a password.
    public static func open(source: VaultFileSource, recoveryPhrase: String) async throws -> Vault {
        let (encKey, macKey) = try keysFromRecovery(recoveryPhrase)
        let jwt = String(decoding: try await source.read("vault.cryptomator"), as: UTF8.self)
        try verifyVaultJWT(jwt.trimmingCharacters(in: .whitespacesAndNewlines), encKey: encKey, macKey: macKey)
        return Vault(encKey: encKey, macKey: macKey)
    }
}
