package com.mobile_gateway

import android.app.*
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat

class GatewayService : Service() {
    private val CHANNEL_ID = "ALHudhudAIGatewayService"

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("بوابة الهدهد")
            .setContentText("خدمة البوابة تعمل بنجاح في الخلفية...")
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setOngoing(true)
            .build()

        startForeground(1, notification)
        scheduleWatchdog()

        // Keep service alive
        return START_STICKY
    }

    private fun scheduleWatchdog() {
        val watchdogRequest = androidx.work.PeriodicWorkRequestBuilder<WatchdogWorker>(
            15, java.util.concurrent.TimeUnit.MINUTES
        ).build()

        androidx.work.WorkManager.getInstance(this).enqueueUniquePeriodicWork(
            "GatewayWatchdog",
            androidx.work.ExistingPeriodicWorkPolicy.KEEP,
            watchdogRequest
        )
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val serviceChannel = NotificationChannel(
                CHANNEL_ID,
                "ALHudhudAI Gateway Service Channel",
                NotificationManager.IMPORTANCE_DEFAULT
            )
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(serviceChannel)
        }
    }
}
