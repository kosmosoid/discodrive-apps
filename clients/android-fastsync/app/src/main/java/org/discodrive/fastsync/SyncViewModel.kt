package org.discodrive.fastsync

import android.app.Application
import android.os.Build
import android.os.Environment
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import mobile.Client
import java.io.File

data class UiState(
    val paired: Boolean = false,
    val working: Boolean = false,
    val state: String = "idle",
    val lastSyncUnix: Long = 0,
    val lastError: String? = null,
    val pendingUserCode: String? = null,
)

class SyncViewModel(app: Application) : AndroidViewModel(app) {
    private val prefs = Prefs(app)
    private var client: Client? = null

    private val _ui = MutableStateFlow(UiState())
    val ui: StateFlow<UiState> = _ui.asStateFlow()

    val syncDir: File = File(Environment.getExternalStorageDirectory(), "DiscoDriveFastSync/Sync")
    private val stateDbPath: String get() = File(getApplication<Application>().filesDir, "state.db").path

    init { refreshAfterPermission() }

    fun hasStoragePermission(): Boolean = Environment.isExternalStorageManager()

    // Open the client when we have permission + a saved token (called on launch and on returning
    // from the All-files-access settings screen).
    fun refreshAfterPermission() {
        val token = prefs.deviceToken
        if (client == null && token != null && prefs.serverURL.isNotEmpty() && hasStoragePermission()) {
            openClient(prefs.serverURL, token, prefs.insecure)
        }
    }

    private fun openClient(server: String, token: String, insecure: Boolean) {
        try {
            syncDir.mkdirs()
            client = SyncCore.newClient(server, token, syncDir.path, stateDbPath, insecure)
            _ui.value = _ui.value.copy(paired = true)
        } catch (e: Exception) {
            _ui.value = _ui.value.copy(lastError = e.message)
        }
    }

    // openUrl is invoked (on the main thread) after PairBegin so the UI can open the browser
    // at the verification URL before PairAwait blocks.
    fun pair(server: String, insecure: Boolean, openUrl: (String) -> Unit) {
        viewModelScope.launch {
            _ui.value = _ui.value.copy(working = true, lastError = null)
            try {
                val p = withContext(Dispatchers.IO) {
                    SyncCore.pairBegin(server, Build.MODEL, "android", insecure)
                }
                _ui.value = _ui.value.copy(pendingUserCode = p.userCode)
                openUrl(p.verificationURL)
                val token = withContext(Dispatchers.IO) {
                    SyncCore.pairAwait(server, p.deviceCode, p.intervalSeconds, insecure)
                }
                prefs.serverURL = server; prefs.insecure = insecure; prefs.deviceToken = token
                withContext(Dispatchers.IO) { openClient(server, token, insecure) }
                SyncWorker.schedule(getApplication())
            } catch (e: Exception) {
                _ui.value = _ui.value.copy(lastError = e.message)
            } finally {
                _ui.value = _ui.value.copy(working = false, pendingUserCode = null)
            }
        }
    }

    fun syncNow() {
        val c = client ?: return
        viewModelScope.launch {
            _ui.value = _ui.value.copy(working = true, lastError = null, state = "syncing")
            try { withContext(Dispatchers.IO) { c.syncOnce() } }
            catch (e: Exception) { _ui.value = _ui.value.copy(lastError = e.message) }
            val st = withContext(Dispatchers.IO) { c.status() }
            _ui.value = _ui.value.copy(
                working = false, state = st.state, lastSyncUnix = st.lastSyncUnix,
                lastError = if (st.lastError.isNotEmpty()) st.lastError else _ui.value.lastError
            )
        }
    }

    fun unpair() {
        try { client?.close() } catch (_: Exception) {}
        client = null
        prefs.clear()
        SyncWorker.cancel(getApplication())
        _ui.value = UiState()
    }
}
