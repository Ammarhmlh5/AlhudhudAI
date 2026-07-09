package com.mobile_gateway

import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.work.Worker
import androidx.work.WorkerParameters

class WatchdogWorker(context: Context, params: WorkerParameters) : Worker(context, params) {
    override fun doWork(): androidx.work.ListenableWorker.Result {
        val serviceIntent = Intent(applicationContext, GatewayService::class.java)
        
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            applicationContext.startForegroundService(serviceIntent)
        } else {
            applicationContext.startService(serviceIntent)
        }
        
        return androidx.work.ListenableWorker.Result.success()
    }
}
