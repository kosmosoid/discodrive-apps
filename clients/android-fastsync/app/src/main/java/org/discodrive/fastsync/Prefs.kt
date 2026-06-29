package org.discodrive.fastsync

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

class Prefs(context: Context) {
    private val sp = EncryptedSharedPreferences.create(
        context, "fastsync",
        MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    var serverURL: String
        get() = sp.getString("serverURL", "") ?: ""
        set(v) { sp.edit().putString("serverURL", v).apply() }
    var deviceToken: String?
        get() = sp.getString("deviceToken", null)
        set(v) { sp.edit().putString("deviceToken", v).apply() }
    var insecure: Boolean
        get() = sp.getBoolean("insecure", false)
        set(v) { sp.edit().putBoolean("insecure", v).apply() }

    fun clear() { sp.edit().clear().apply() }
}
