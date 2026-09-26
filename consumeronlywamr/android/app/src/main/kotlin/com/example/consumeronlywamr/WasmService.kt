package com.example.consumeronlywamr

import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.util.Log

/**
 * WasmService
 *
 * Dedicated background execution service hosting the WebAssembly Micro Runtime (WAMR).
 *
 * Process & Security Architecture:
 * - Runs in an isolated Linux process (`:wasm_engine`) as declared in AndroidManifest.xml.
 * - Enforces a security and fault-tolerance boundary: Any memory violations, panics,
 *   or aborts in untrusted WASM or C++ native code will terminate only this background
 *   process, preventing the main application and Flutter UI from crashing.
 * - Implements [WasmServiceInterface.Stub] to expose RPC/IPC endpoints via Android Binder.
 * - Forwards incoming requests to the native layer (`libnative-lib.so`) via JNI.
 */
class WasmService : Service() {

    /**
     * Service lifecycle initialization:
     * - Initializes global WAMR runtime environment.
     * - Configures host imports and streaming I/O via [HttpStreamer].
     */
    override fun onCreate() {
        super.onCreate()
        // Initialize WASM runtime when service starts
        initWasm()
        
        // Setup JNI bridge for HTTP Streaming
        val streamer = HttpStreamer()
        streamer.setStorageDir(filesDir)
        streamer.bindNative()
        Log.i("WasmService", "HttpStreamer Native bindings initialized with storage at ${filesDir.absolutePath}")
    }

    /**
     * AIDL IPC Binder Implementation:
     * Implements [WasmServiceInterface.Stub] to fulfill remote IPC requests sent by [MainActivity].
     */
    private val binder =
            object : WasmServiceInterface.Stub() {
                /**
                 * Dispatches an exported WASM function by name with integer parameters.
                 */
                override fun invokeWasm(
                        wasmBytes: ByteArray?,
                        funcName: String?,
                        args: IntArray?
                ): Int {
                    if (wasmBytes == null || funcName == null || args == null) return -1
                    return this@WasmService.invokeWasm(wasmBytes, funcName, args)
                }

                /**
                 * Passes string payload into WASM linear memory and captures string return value.
                 */
                override fun invokeWasmString(
                        wasmBytes: ByteArray?,
                        funcName: String?,
                        payload: String?
                ): String {
                    if (wasmBytes == null || funcName == null || payload == null) return "Error: Invalid arguments"
                    return this@WasmService.invokeWasmString(wasmBytes, funcName, payload)
                }

                /**
                 * Executes default entrypoint of the given WASM binary, returning runtime logs.
                 */
                override fun runWasm(wasmBytes: ByteArray?): String {
                    if (wasmBytes == null) return "Error: Null bytes"
                    return this@WasmService.runWasm(wasmBytes)
                }
                
                // just for initial testing no need in the main version 
                override fun wasmAdd(wasmBytes: ByteArray?, a: Int, b: Int): Int {
                    if (wasmBytes == null) return -1
                    return this@WasmService.wasmAdd(wasmBytes, intArrayOf(a, b))
                }

                /**
                 * Universal ABI Dispatcher:
                 * Passes binary payload into WASM linear memory and returns output byte array.
                 */
                override fun invokeDataWasm(
                        wasmBytes: ByteArray?,
                        funcName: String?,
                        payload: ByteArray?,
                        hostPath: String?
                ): ByteArray? {
                    if (wasmBytes == null || funcName == null || payload == null || hostPath == null
                    )
                            return ByteArray(0)
                    return invokeDataWasmNative(wasmBytes, funcName, payload, hostPath)
                }
            }

    /**
     * Returns the Binder interface token to binding clients ([MainActivity]).
     */
    override fun onBind(intent: Intent?): IBinder {
        return binder
    }

    // =========================================================================
    // JNI Native Methods (Implemented in cpp/native-lib.cpp)
    // =========================================================================

    /** Initializes the WAMR runtime engine and memory allocator. */
    external fun initWasm(): Int

    /** Executes module main and returns captured stdout/stderr. */
    external fun runWasm(wasmBytes: ByteArray): String

    /** Legacy smoke-test method: invokes 'add(a, b)' exported symbol. */
    external fun wasmAdd(wasmBytes: ByteArray, args: IntArray): Int

    /** Generic dispatcher: executes funcName with int[] arguments. */
    external fun invokeWasm(wasmBytes: ByteArray, funcName: String, args: IntArray): Int

    /** String dispatcher: passes UTF-8 payload and returns UTF-8 result. */
    external fun invokeWasmString(wasmBytes: ByteArray, funcName: String, payload: String): String

    // Removed incorrect override of invokeDataWasm.
    // It should only be implemented in the Stub (binder) or via direct delegation.

    /** Raw buffer ABI: marshals byte array buffers into WASM linear memory. */
    private external fun invokeDataWasmNative(
            wasmBytes: ByteArray?,
            funcName: String?,
            payload: ByteArray?,
            hostPath: String?
    ): ByteArray?

    companion object {
        init {
            try {
                // Load native C++ library containing WAMR and JNI bindings.
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
