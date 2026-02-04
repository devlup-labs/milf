# Wasm on Android: Complete Guide

This guide documents the architecture, implementation details, and reconstruction checkpoints for running WebAssembly (Wasm) on Android using WAMR (WebAssembly Micro Runtime) in an isolated process.

## 🚀 One-Command Execution
Once the project is set up, the entire build system (Gradle + CMake + Flutter) handles everything.
To run the app:

```bash
flutter run
```

To compile a new Wasm binary (e.g., `add.wasm`):
```bash
/Users/adarsh/Library/Android/sdk/ndk/28.2.13676358/toolchains/llvm/prebuilt/darwin-x86_64/bin/clang \
    --target=wasm32 -O3 -nostdlib \
    -Wl,--no-entry -Wl,--export=app_main -Wl,--export=add -Wl,--allow-undefined \
    -o test/addf.wasm test/add.c
```

---

## 📍 Project Checkpoints (Reconstruction Guide)
To recreate this project from scratch, follow these checkpoints:

1.  **Project Setup**: Create a standard Flutter app (`flutter create .`).
2.  **WAMR Integration**:
    *   Clone `wasm-micro-runtime` into `android/app/src/main/cpp/wamr`.
    *   Create `CMakeLists.txt` to compile WAMR core (`iwasm`) and your native bridge (`native-lib.cpp`).
3.  **Android Service (Isolation)**:
    *   Define `WasmServiceInterface.aidl` for IPC (Inter-Process Communication).
    *   Create `WasmService.kt` and set `android:isolatedProcess="true"` in `AndroidManifest.xml`.
4.  **JNI Bridge**:
    *   Implement `native-lib.cpp` to initialize the Wasm runtime and execute modules.
5.  **Flutter UI**:
    *   Use `MethodChannel` in `MainActivity.kt` to communicate between Flutter and the Android Service.

---

## 🧠 Code Deep Dive (Learning)

### 1. The Entry Point: `MainActivity.kt`
**Role:** The Coordinator. It bridges Flutter (Dart) and the Android Service.

```kotlin
// 1. Service Connection: Handles the Binder connection to our isolated service.
private val connection = object : ServiceConnection {
    override fun onServiceConnected(className: ComponentName, service: IBinder) {
        // Converts the raw Binder object into our AIDL interface
        wasmService = WasmServiceInterface.Stub.asInterface(service)
        isBound = true
    }
    // ...
}

// 2. Flutter Engine Config: Sets up the MethodChannel.
override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
    // START the isolated service so it's ready.
    val intent = Intent(this, WasmService::class.java)
    bindService(intent, connection, Context.BIND_AUTO_CREATE)

    // Listen for messages from Dart ("runWasm")
    MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler { call, result ->
        if (call.method == "runWasm") {
            // EXECUTE on a background thread (network/disk ops shouldn't block UI)
            Executors.newSingleThreadExecutor().execute {
                 // CALL the isolated service via AIDL
                 val output = wasmService?.runWasm(bytes)
                 // RETURN result to Flutter Main Thread
                 runOnUiThread { result.success(output) }
            }
        }
    }
}
```

### 2. The Sandbox: `WasmService.kt`
**Role:** The Bodyguard. It runs in a separate process (`isolatedProcess="true"`). If Wasm crashes here, the main app survives.

```kotlin
class WasmService : Service() {
    // 1. Static Initialization: Loads the C++ library when the class is loaded.
    companion object {
        init {
            System.loadLibrary("native-lib") // Loads libnative-lib.so
        }
    }

    // 2. Lifecycle: Initializes the WAMR runtime once when service is created.
    override fun onCreate() {
        initWasm() // Native call to wasm_runtime_full_init()
    }

    // 3. The Binder: Implementation of the AIDL interface.
    private val binder = object : WasmServiceInterface.Stub() {
        override fun runWasm(bytes: ByteArray?): String {
            // DELEGATE to native JNI function
            return this@WasmService.runWasm(bytes!!)
        }
        // ...
    }
}
```

### 3. The Engine: `native-lib.cpp`
**Role:** The Translator. Converts Java data to C, runs WAMR, and converts results back.

```cpp
// 1. Initialization
extern "C" JNIEXPORT jint JNICALL Java_..._initWasm(...) {
    RuntimeInitArgs init_args;
    // ... configure memory allocator ...
    return wasm_runtime_full_init(&init_args); // Boot WAMR
}

// 2. Execution Logic (simplified)
extern "C" JNIEXPORT jstring JNICALL Java_..._runWasm(..., jbyteArray wasmBytes) {
    // A. Load Wasm bytes from Java array
    jbyte *buffer = env->GetByteArrayElements(wasmBytes, NULL);

    // B. Load Module
    module = wasm_runtime_load(buffer, len, error_buf, sizeof(error_buf));

    // C. Instantiate (Create memory/stack)
    module_inst = wasm_runtime_instantiate(module, stack_size, heap_size, ...);

    // D. Lookup Entry Function
    // We check for "app_main" (our 0-arg wrapper), then "main", etc.
    func = wasm_runtime_lookup_function(module_inst, "app_main");

    // E. Execute
    // We pass 'argv' to capture the return value!
    uint32_t argv[2] = {0};
    if (wasm_runtime_call_wasm(exec_env, func, 0, argv)) {
        // F. Success: Extract result from stack (argv[0])
        std::string res = "Result: " + std::to_string(argv[0]);
        return env->NewStringUTF(res.c_str());
    } else {
        // G. Failure: Get exception message
        const char *ex = wasm_runtime_get_exception(module_inst);
        return env->NewStringUTF(ex);
    }
}
```

### 4. The Blueprint: `CMakeLists.txt`
**Role:** The Architect. Tells the compiler how to build the native library.

```cmake
# 1. Configuration flags (Performance vs Features)
set(WAMR_BUILD_INTERP 1)        # Enable Interpreter
set(WAMR_BUILD_FAST_INTERP 1)   # Faster Interpreter
# set(WAMR_ENABLE_LIBC_WASI 1)  # (Currently commented out for stability testing)

# 2. Includes: Tell compiler where to find headers (.h files)
include_directories(${WAMR_ROOT_DIR}/core/iwasm/include)
# ...

# 3. Sources: Tell compiler which .c files to compile
set(WAMR_CORE_SRC ${COMMON_SRC} ${INTERP_SRC} ...)

# 4. Target: Build 'native-lib' as a Shared Library (.so)
add_library(native-lib SHARED native-lib.cpp ${WAMR_CORE_SRC})

# 5. Link: Connect standard Android libraries
target_link_libraries(native-lib log m dl)
```
