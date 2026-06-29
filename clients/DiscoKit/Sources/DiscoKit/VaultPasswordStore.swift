import Foundation
import Security
import LocalAuthentication

public enum BiometryKind: Sendable {
    case none, touchID, faceID, opticID

    public var displayName: String? {
        switch self {
        case .faceID:  return "Face ID"
        case .touchID: return "Touch ID"
        case .opticID: return "Optic ID"
        case .none:    return nil
        }
    }
    public var sfSymbol: String {
        switch self { case .faceID: return "faceid"; case .opticID: return "opticid"; default: return "touchid" }
    }
}

// Stores vault passwords in the Keychain protected by biometrics (Face ID / Touch ID,
// falling back to the device passcode). The key is a stable vault identifier (server path).
public enum VaultPasswordStore {
    private static let service = "org.discodrive.vaultpw"

    public enum VaultPWError: Error { case cancelled, failed(OSStatus) }

    // Returns the available biometry kind (for button labels and icons).
    public static func biometry() -> BiometryKind {
        let ctx = LAContext()
        var err: NSError?
        guard ctx.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &err) else { return .none }
        switch ctx.biometryType {
        case .touchID: return .touchID
        case .faceID:  return .faceID
        case .opticID: return .opticID
        default:       return .none
        }
    }

    // Returns true if a password is already saved for the vault.
    public static func hasPassword(forVault key: String) -> Bool {
        SecItemCopyMatching(baseQuery(key) as CFDictionary, nil) != errSecItemNotFound
    }

    // Save a password (plain Keychain item, no access-control → no special entitlements needed).
    // Biometric gate is applied via LAContext at read time.
    @discardableResult
    public static func save(password: String, forVault key: String) -> OSStatus {
        delete(forVault: key)
        var q = baseQuery(key)
        q[kSecValueData as String] = Data(password.utf8)
        // Available only after first unlock, and never synced or migrated off this device.
        q[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        return SecItemAdd(q as CFDictionary, nil)
    }

    // Retrieve a password: first authenticate via Face ID / Touch ID / passcode (LAContext), then read.
    public static func loadPassword(forVault key: String, reason: String) async throws -> String {
        let ctx = LAContext()
        let ok = try await ctx.evaluatePolicy(.deviceOwnerAuthentication, localizedReason: reason)
        guard ok else { throw VaultPWError.cancelled }
        var q = baseQuery(key)
        q[kSecReturnData as String] = true
        var out: CFTypeRef?
        let status = SecItemCopyMatching(q as CFDictionary, &out)
        guard status == errSecSuccess, let data = out as? Data, let pw = String(data: data, encoding: .utf8) else {
            throw VaultPWError.failed(status)
        }
        return pw
    }

    private static func baseQuery(_ key: String) -> [String: Any] {
        [kSecClass as String: kSecClassGenericPassword,
         kSecAttrService as String: service,
         kSecAttrAccount as String: key]
    }

    public static func delete(forVault key: String) {
        SecItemDelete([
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ] as CFDictionary)
    }
}
