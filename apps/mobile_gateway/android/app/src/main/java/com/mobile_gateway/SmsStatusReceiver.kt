package com.mobile_gateway

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.modules.core.DeviceEventManagerModule
import com.facebook.react.bridge.Arguments

class SmsStatusReceiver(private val reactContext: ReactApplicationContext) : BroadcastReceiver() {

    override fun onReceive(context: Context?, intent: Intent?) {
        val id = intent?.getStringExtra("id") ?: return
        val action = intent?.action ?: return

        val params = Arguments.createMap()
        params.putString("id", id)

        if (action == "SMS_SENT") {
            val resultCode = intent.getIntExtra("resultCode", Activity.RESULT_OK)
            if (resultCode == Activity.RESULT_OK) {
                params.putString("status", "SUCCESS")
            } else {
                val reason = when (resultCode) {
                    android.telephony.SmsManager.RESULT_ERROR_GENERIC_FAILURE -> "Generic Failure (Check Balance/Signal)"
                    android.telephony.SmsManager.RESULT_ERROR_NO_SERVICE -> "No Service"
                    android.telephony.SmsManager.RESULT_ERROR_NULL_PDU -> "Null PDU"
                    android.telephony.SmsManager.RESULT_ERROR_RADIO_OFF -> "Airplane Mode/Radio Off"
                    else -> "Unknown Error (Code: $resultCode)"
                }
                params.putString("status", reason)
            }
            reactContext
                .getJSModule(DeviceEventManagerModule.RCTDeviceEventEmitter::class.java)
                .emit("onSmsSent", params)
        } else if (action == "SMS_DELIVERED") {
            params.putString("status", "SUCCESS")
            reactContext
                .getJSModule(DeviceEventManagerModule.RCTDeviceEventEmitter::class.java)
                .emit("onSmsDelivered", params)
        }
    }
}
