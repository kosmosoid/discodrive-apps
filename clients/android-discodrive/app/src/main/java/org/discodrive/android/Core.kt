package org.discodrive.android

import mobile.Browser
import mobile.Mobile
import mobile.Pairing
import mobile.Vault

// Wrapper over the gomobile API (package `mobile`). All calls throw and block — use Dispatchers.IO.
object Core {
    fun pairBegin(server: String, name: String, kind: String, insecure: Boolean): Pairing =
        Mobile.pairBegin(server, name, kind, insecure)

    fun pairAwait(server: String, deviceCode: String, intervalSec: Long, insecure: Boolean): String =
        Mobile.pairAwait(server, deviceCode, intervalSec, insecure)

    fun newBrowser(server: String, token: String, rootDir: String, indexDBPath: String, insecure: Boolean): Browser =
        Mobile.newBrowser(server, token, rootDir, indexDBPath, insecure)

    fun openVault(server: String, token: String, vaultRoot: String, password: String,
                  indexDBPath: String, tmpDir: String, insecure: Boolean): Vault =
        Mobile.openVault(server, token, vaultRoot, password, indexDBPath, tmpDir, insecure)
}
