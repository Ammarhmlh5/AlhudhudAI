package com.mobile_gateway

import android.content.Intent
import com.facebook.react.bridge.*

class GatewayModule(reactContext: ReactApplicationContext) : ReactContextBaseJavaModule(reactContext) {

    override fun getName(): String {
        return "GatewayModule"
    }

    @ReactMethod
    fun startService(promise: Promise) {
        try {
            val intent = Intent(reactApplicationContext, GatewayService::class.java)
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
                reactApplicationContext.startForegroundService(intent)
            } else {
                reactApplicationContext.startService(intent)
            }
            promise.resolve(true)
        } catch (e: Exception) {
            promise.reject("SERVICE_ERROR", e.message)
        }
    }

    @ReactMethod
    fun stopService(promise: Promise) {
        try {
            val intent = Intent(reactApplicationContext, GatewayService::class.java)
            reactApplicationContext.stopService(intent)
            promise.resolve(true)
        } catch (e: Exception) {
            promise.reject("SERVICE_ERROR", e.message)
        }
    }

    @ReactMethod
    fun getSimInfo(promise: Promise) {
        try {
            val sm = reactApplicationContext.getSystemService(android.content.Context.TELEPHONY_SUBSCRIPTION_SERVICE) as android.telephony.SubscriptionManager
            val simList = WritableNativeArray()
            
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.LOLLIPOP_MR1) {
                val activeSubscriptions = sm.activeSubscriptionInfoList
                if (activeSubscriptions != null) {
                    for (info in activeSubscriptions) {
                        val simInfo = WritableNativeMap()
                        simInfo.putInt("subscriptionId", info.subscriptionId)
                        simInfo.putString("carrierName", info.carrierName.toString())
                        simInfo.putString("displayName", info.displayName.toString())
                        simInfo.putInt("slotIndex", info.simSlotIndex)
                        simInfo.putString("number", info.number ?: "Hidden")
                        simList.pushMap(simInfo)
                    }
                }
            }
            promise.resolve(simList)
        } catch (e: Exception) {
            promise.reject("SIM_ERROR", e.message)
        }
    }

    @ReactMethod
    fun requestBatteryOptimization(promise: Promise) {
        try {
            val intent = Intent()
            val packageName = reactApplicationContext.packageName
            val pm = reactApplicationContext.getSystemService(android.content.Context.POWER_SERVICE) as android.os.PowerManager
            
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.M) {
                if (!pm.isIgnoringBatteryOptimizations(packageName)) {
                    intent.action = android.provider.Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS
                    intent.data = android.net.Uri.parse("package:$packageName")
                    intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                    reactApplicationContext.startActivity(intent)
                }
            }
            promise.resolve(true)
        } catch (e: Exception) {
            promise.reject("BATTERY_ERROR", e.message)
        }
    }
}
