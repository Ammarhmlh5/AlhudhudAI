package com.mobile_gateway

import android.app.Activity
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.telephony.SmsManager
import com.facebook.react.bridge.*

class SmsModule(reactContext: ReactApplicationContext) : ReactContextBaseJavaModule(reactContext) {

    private val statusReceiver = SmsStatusReceiver(reactContext)

    init {
        val intentFilter = android.content.IntentFilter().apply {
            addAction("SMS_SENT")
            addAction("SMS_DELIVERED")
        }
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.TIRAMISU) {
            reactContext.registerReceiver(statusReceiver, intentFilter, Context.RECEIVER_EXPORTED)
        } else {
            reactContext.registerReceiver(statusReceiver, intentFilter)
        }
    }

    override fun getName(): String {
        return "ALGatewaySmsModule"
    }

    @ReactMethod
    fun sendSMS(id: String, phoneNumber: String, message: String, promise: Promise) {
        try {
            val smsManager: SmsManager = reactApplicationContext.getSystemService(SmsManager::class.java)
            
            val sentIntent = PendingIntent.getBroadcast(
                reactApplicationContext, 
                id.hashCode(), 
                Intent("SMS_SENT").putExtra("id", id), 
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )

            val deliveredIntent = PendingIntent.getBroadcast(
                reactApplicationContext, 
                id.hashCode(), 
                Intent("SMS_DELIVERED").putExtra("id", id), 
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )

            // Send multi-part if needed, but for gateway usually short
            val parts = smsManager.divideMessage(message)
            if (parts.size > 1) {
                val sentIntents = ArrayList<PendingIntent>()
                val deliveryIntents = ArrayList<PendingIntent>()
                for (i in parts.indices) {
                    sentIntents.add(sentIntent)
                    deliveryIntents.add(deliveredIntent)
                }
                smsManager.sendMultipartTextMessage(phoneNumber, null, parts, sentIntents, deliveryIntents)
            } else {
                smsManager.sendTextMessage(phoneNumber, null, message, sentIntent, deliveredIntent)
            }

            promise.resolve(true)
        } catch (e: Exception) {
            promise.reject("SMS_ERROR", e.message)
        }
    }

    @ReactMethod
    fun addListener(eventName: String) {
        // Required for NativeEventEmitter
    }

    @ReactMethod
    fun removeListeners(count: Int) {
        // Required for NativeEventEmitter
    }
}
