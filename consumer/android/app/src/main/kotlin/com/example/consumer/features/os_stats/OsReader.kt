package com.example.consumer.features.os_stats

import android.app.ActivityManager

import android.content.Context
import android.os.Debug

class OsReader(private val context: Context) {

    /**
     * Returns total PSS (Proportional Set Size) memory for the entire app in KB.
     */
    fun getAppMemory(): Long {
        val am = context.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val memInfo = am.getProcessMemoryInfo(intArrayOf(android.os.Process.myPid()))
        return if (memInfo.isNotEmpty()) {
            memInfo[0].totalPss.toLong()
        } else {
            0L
        }
    }

    /**
     * Returns PSS memory for a specific PID (if accessible).
     */
    fun getProcessMem(pid: Int): Long {
        val am = context.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val memInfo = am.getProcessMemoryInfo(intArrayOf(pid))
        return if (memInfo.isNotEmpty()) {
            memInfo[0].totalPss.toLong()
        } else {
            0L
        }
    }

    fun getProcessCpu(pid: Int): Double {
        // CPU reading is complex on Android 8+ due to /proc restrictions
        // Usually requires reading /proc/stat if accessible or using internal telemetry
        return 0.0
    }

    /**
     * Returns device-wide memory info in KB.
     */
    fun getDeviceMemory(): Map<String, Long> {
        val am = context.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val outInfo = ActivityManager.MemoryInfo()
        am.getMemoryInfo(outInfo)
        return mapOf(
            "avail" to outInfo.availMem / 1024,
            "total" to outInfo.totalMem / 1024,
            "threshold" to outInfo.threshold / 1024,
            "lowMemory" to if (outInfo.lowMemory) 1L else 0L
        )
    }
}
