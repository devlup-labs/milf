package com.example.consumeronlywamr

import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.ServiceConnection
import android.os.IBinder
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.util.concurrent.Executors

class MainActivity : FlutterActivity() {
    private val CHANNEL = "com.example.consumeronlywamr/wasm"
    private var wasmService: WasmServiceInterface? = null
    private var isBound = false

    private val connection =
            object : ServiceConnection {
                override fun onServiceConnected(className: ComponentName, service: IBinder) {
                    wasmService = WasmServiceInterface.Stub.asInterface(service)
                    isBound = true
                }

                override fun onServiceDisconnected(arg0: ComponentName) {
                    wasmService = null
                    isBound = false
                }
            }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Bind to the isolated service
        val intent = Intent(this, WasmService::class.java)
        bindService(intent, connection, Context.BIND_AUTO_CREATE)

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler {
                call,
                result ->
            when (call.method) {
                "runWasm" -> {
                    val wasmBytes = call.argument<ByteArray>("bytes")
                    if (wasmBytes != null && isBound && wasmService != null) {
                        Executors.newSingleThreadExecutor().execute {
                            try {
                                val output = wasmService?.runWasm(wasmBytes)
                                runOnUiThread { result.success(output) }
                            } catch (e: Exception) {
                                runOnUiThread {
                                    result.error("EXECUTION_ERROR", e.toString(), null)
                                }
                            }
                        }
                    } else {
                        result.error("ERROR", "Service not bound or null bytes", null)
                    }
                }
                "invokeWasm" -> {
                    val bytes = call.argument<ByteArray>("bytes")
                    val func = call.argument<String>("funcName")
                    val args = call.argument<IntArray>("args")

                    if (bytes != null &&
                                    func != null &&
                                    args != null &&
                                    isBound &&
                                    wasmService != null
                    ) {
                        Executors.newSingleThreadExecutor().execute {
                            try {
                                val res = wasmService?.invokeWasm(bytes, func, args)
                                runOnUiThread { result.success(res) }
                            } catch (e: Exception) {
                                runOnUiThread { result.error("INVOKE_ERROR", e.toString(), null) }
                            }
                        }
                    } else {
                        result.error("INVALID_ARGS", "Missing arguments for invokeWasm", null)
                    }
                }
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        if (isBound) {
            unbindService(connection)
            isBound = false
        }
    }
}
