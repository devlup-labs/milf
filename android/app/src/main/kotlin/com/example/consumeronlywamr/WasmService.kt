package com.example.consumeronlywamr

import android.app.Service
import android.content.Intent
import android.os.IBinder

// Controller

class WasmService : Service() {

    override fun onCreate() {
        super.onCreate()
        // Initialize WASM runtime when service starts
        initWasm()
    }

    private val binder =
            object : WasmServiceInterface.Stub() {
                override fun invokeWasm(
                        wasmBytes: ByteArray?,
                        funcName: String?,
                        args: IntArray?
                ): Int {
                    if (wasmBytes == null || funcName == null || args == null) return -1
                    return this@WasmService.invokeWasm(wasmBytes, funcName, args)
                }

                override fun runWasm(wasmBytes: ByteArray?): String {
                    if (wasmBytes == null) return "Error: Null bytes"
                    return this@WasmService.runWasm(wasmBytes)
                }

                override fun wasmAdd(wasmBytes: ByteArray?, a: Int, b: Int): Int {
                    if (wasmBytes == null) return -1
                    return this@WasmService.wasmAdd(wasmBytes, intArrayOf(a, b))
                }
            }

    override fun onBind(intent: Intent?): IBinder {
        return binder
    }

    // JNI Native methods
    external fun initWasm(): Int
    external fun runWasm(wasmBytes: ByteArray): String
    external fun wasmAdd(wasmBytes: ByteArray, args: IntArray): Int
    external fun invokeWasm(wasmBytes: ByteArray, funcName: String, args: IntArray): Int

    companion object {
        init {
            try {
                System.loadLibrary("native-lib")
            } catch (e: UnsatisfiedLinkError) {
                // Determine if this is a crash or just missing lib
                e.printStackTrace()
                // Since this is static init, throwing exception crashes the app (or service)
                // catching it might allow service to start but then methods fail.
                // But at least we might see logs or service stays alive to report error?
                // No, better to let it crash but maybe we can log to a file or special place?
                // Standard logcat will catch it.
                // But let's rethrow to be sure, but log explicitly first.
                throw e
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }
    }
}
