package org.discodrive.fastsync

import mobile.Client
import mobile.Mobile
import mobile.Pairing

// Wrapper over the gomobile-generated Kfmobile API (package `mobile`). All methods throw
// Exception. They block — call from Dispatchers.IO.
object SyncCore {
    fun pairBegin(server: String, name: String, kind: String, insecure: Boolean): Pairing =
        Mobile.pairBegin(server, name, kind, insecure)

    fun pairAwait(server: String, deviceCode: String, intervalSec: Long, insecure: Boolean): String =
        Mobile.pairAwait(server, deviceCode, intervalSec, insecure)

    fun newClient(server: String, token: String, syncDir: String, dbPath: String, insecure: Boolean): Client =
        Mobile.new_(server, token, syncDir, dbPath, insecure)
}
