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
            if (call.method == "runWasm") {
                val wasmBytes = call.argument<ByteArray>("bytes")
                if (wasmBytes != null && isBound && wasmService != null) {
                    // Run on background thread to not block UI
                    Executors.newSingleThreadExecutor().execute {
                        try {
                            val output = wasmService?.runWasm(wasmBytes)
                            runOnUiThread { result.success(output) }
                        } catch (e: Exception) {
                            runOnUiThread { result.error("EXECUTION_ERROR", e.toString(), null) }
                        }
                    }
                } else {
                    if (!isBound) {
                        result.error("SERVICE_NOT_BOUND", "WasmService is not bound yet", null)
                    } else {
                        result.error("INVALID_ARGUMENT", "Bytes are null", null)
                    }
                }
            } else {
                result.notImplemented()
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
