package org.discodrive.fastsync

import android.content.Context
import android.os.Environment
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import java.io.File
import java.util.concurrent.TimeUnit

// Opportunistic periodic background sync. WorkManager's minimum period is 15 min; the OS may
// defer it (Doze, OEM limits). Builds its own client from saved prefs, runs one SyncOnce.
class SyncWorker(ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    override suspend fun doWork(): Result {
        val prefs = Prefs(applicationContext)
        val token = prefs.deviceToken ?: return Result.success()
        if (!Environment.isExternalStorageManager()) return Result.success()
        return try {
            val dir = File(Environment.getExternalStorageDirectory(), "DiscoDriveFastSync/Sync").apply { mkdirs() }
            val db = File(applicationContext.filesDir, "state.db").path
            val client = SyncCore.newClient(prefs.serverURL, token, dir.path, db, prefs.insecure)
            client.syncOnce()
            client.close()
            Result.success()
        } catch (e: Exception) {
            Result.retry()
        }
    }

    companion object {
        private const val NAME = "fastsync-periodic"
        fun schedule(ctx: Context) {
            val req = PeriodicWorkRequestBuilder<SyncWorker>(20, TimeUnit.MINUTES).build()
            WorkManager.getInstance(ctx)
                .enqueueUniquePeriodicWork(NAME, ExistingPeriodicWorkPolicy.KEEP, req)
        }
        fun cancel(ctx: Context) { WorkManager.getInstance(ctx).cancelUniqueWork(NAME) }
    }
}
