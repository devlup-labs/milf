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

/**
 * MainActivity
 *
 * Serves as the primary entry point and UI coordinator for the application.
 *
 * Architecture Role:
 * - Runs in the default application process (`com.example.consumeronlywamr`).
 * - Acts as an IPC (Inter-Process Communication) gateway bridging Flutter's Dart layer
 *   and the isolated native WASM execution service (`:wasm_engine`).
 * - Uses [MethodChannel] to receive execution commands from Dart.
 * - Forwards requests across the Android Binder IPC interface ([WasmServiceInterface])
 *   to [WasmService].
 * - Executes native/IPC work asynchronously using background threads to ensure the
 *   Flutter UI thread remains responsive, marshaling responses back via [runOnUiThread].
 */
class MainActivity : FlutterActivity() {
    private val CHANNEL = "com.example.consumeronlywamr/wasm"
    private var wasmService: WasmServiceInterface? = null
    private var isBound = false

    /**
     * IPC ServiceConnection callback monitor.
     * Manages the Binder lifecycle connecting this Activity to [WasmService].
     */
    private val connection =
            object : ServiceConnection {
                /**
                 * Invoked when the IPC Binder connection to [WasmService] has been established.
                 * Unmarshals the raw [IBinder] into the type-safe [WasmServiceInterface] proxy.
                 */
                override fun onServiceConnected(className: ComponentName, service: IBinder) {
                    wasmService = WasmServiceInterface.Stub.asInterface(service)
                    isBound = true
                }

                /**
                 * Invoked if the remote service process crashes or is killed by the OS.
                 * Invalidates the local proxy reference.
                 */
                override fun onServiceDisconnected(arg0: ComponentName) {
                    wasmService = null
                    isBound = false
                }
            }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        // Bind to the isolated background service hosting the WAMR runtime.
        val intent = Intent(this, WasmService::class.java)
        bindService(intent, connection, Context.BIND_AUTO_CREATE)

        // MethodChannel handler: Listens for method invocations dispatched from Flutter Dart.
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler {
                call,
                result ->
            when (call.method) {
                // Execute WASM binary with generic entrypoint (e.g. main/app_main) returning String log/output.
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
                // Invokes an exported WASM function by symbol name, passing primitive integer parameters.
                "invokeWasm" -> {
                    val bytes = call.argument<ByteArray>("bytes")
                    val func = call.argument<String>("funcName")
                    val args = call.argument<IntArray>("args")  //TODO: need to have the change the type or research on the aws way of taking the input 

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
                // Invokes an exported WASM function passing a UTF-8 string payload (e.g., JSON) and returns string result.
                "invokeWasmString" -> {
                    val bytes = call.argument<ByteArray>("bytes")
                    val func = call.argument<String>("funcName")
                    val payload = call.argument<String>("payload")

                    if (bytes != null &&
                        func != null &&
                        payload != null &&
                        isBound &&
                        wasmService != null
                    ) {
                        Executors.newSingleThreadExecutor().execute {
                            try {
                                val res = wasmService?.invokeWasmString(bytes, func, payload)
                                runOnUiThread { result.success(res) }
                            } catch (e: Exception) {
                                runOnUiThread { result.error("INVOKE_STRING_ERROR", e.toString(), null) }
                            }
                        }
                    } else {
                        result.error("INVALID_ARGS", "Missing arguments for invokeWasmString", null)
                    }
                }
                // Reads local file bytes from internal storage (filesDir) to provide assets/data to Flutter.
                "readLocalFile" -> {
                    val name = call.argument<String>("name")
                    if (name != null) {
                        try {
                            val file = java.io.File(filesDir, name)
                            if (file.exists()) {
                                result.success(file.readBytes())
                            } else {
                                result.error("FILE_NOT_FOUND", "File $name not found in $filesDir", null)
                            }
                        } catch (e: Exception) {
                            result.error("READ_ERROR", e.toString(), null)
                        }
                    } else {
                        result.error("INVALID_ARGS", "Missing file name", null)
                    }
                }
                // Advanced ABI Dispatcher: Passes raw binary payload into WASM linear memory and receives output bytes.
                "invokeDataWasm" -> {
                    val bytes = call.argument<ByteArray>("bytes")
                    val func = call.argument<String>("funcName") // TODO: think about the convinient way for this or can be use something as lambda_handler
                    val payload = call.argument<ByteArray>("payload")
                    val hostPath = applicationContext.cacheDir.absolutePath

                    if (bytes != null &&
                                    func != null &&
                                    payload != null &&
                                    isBound &&
                                    wasmService != null
                    ) {
                        Executors.newSingleThreadExecutor().execute {
                            try {
                                val res =
                                        wasmService?.invokeDataWasm(bytes, func, payload, hostPath)
                                runOnUiThread { result.success(res) }
                            } catch (e: Exception) {
                                runOnUiThread {
                                    result.error("INVOKE_DATA_ERROR", e.toString(), null)
                                }
                            }
                        }
                    } else {
                        result.error(
                                "INVALID_ARGS",
                                "Missing arguments or service not bound for invokeDataWasm",
                                null
                        )
                    }
                }
                // Retrieves hardware/OS unique Android ID for node identification in the distributed network.
                "getDeviceId" -> {
                    try {
                        val androidId = android.provider.Settings.Secure.getString(
                            contentResolver,
                            android.provider.Settings.Secure.ANDROID_ID
                        )
                        result.success(androidId ?: "unknown_device")
                    } catch (e: Exception) {
                        result.error("DEVICE_ID_ERROR", e.toString(), null)
                    }
                }
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

    /**
     * Cleans up IPC bindings when Activity is destroyed to prevent service leaks.
     */
    override fun onDestroy() {
        super.onDestroy()
        if (isBound) {
            unbindService(connection)
            isBound = false
        }
    }
}
