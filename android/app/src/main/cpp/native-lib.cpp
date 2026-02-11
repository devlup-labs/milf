#include "execution_monitor.h"
#include "memory_tracker.h"
#include "wasm_export.h"
#include "wasm_runtime.h"
#include <android/log.h>
#include <jni.h>
#include <string>

#define LOG_TAG "native-lib"
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, LOG_TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)

extern "C" JNIEXPORT jint JNICALL JNI_OnLoad(JavaVM *vm, void *reserved) {
  LOGI("JNI_OnLoad called");
  LOGI("sizeof(WASMModuleInstance) = %zu", sizeof(WASMModuleInstance));
  LOGI("sizeof(WASMModuleInstanceExtra) = %zu",
       sizeof(WASMModuleInstanceExtra));
  LOGI("sizeof(WASMModuleInstanceExtraCommon) = %zu",
       sizeof(WASMModuleInstanceExtraCommon));
  return JNI_VERSION_1_6;
}

// REMOVED: static char global_heap_buf[512 * 1024];
// Now using system allocator for dynamic memory!

extern "C" JNIEXPORT jint JNICALL
Java_com_example_consumeronlywamr_WasmService_initWasm(JNIEnv *env,
                                                       jobject /* this */) {

  LOGI("Initializing WAMR with dynamic memory allocation");

  RuntimeInitArgs init_args;
  memset(&init_args, 0, sizeof(RuntimeInitArgs));

  // USE SYSTEM ALLOCATOR (not pool-based)
  init_args.mem_alloc_type = Alloc_With_System_Allocator;
  init_args.max_thread_num = 4;

  // Initialize memory tracking
  MemoryTracker::Initialize();

  if (!wasm_runtime_full_init(&init_args)) {
    LOGE("Init runtime environment failed.");
    return -1;
  }

  LOGI("Init runtime environment success.");
  LOGI("  Max heap: %zu MB", MemoryTracker::GetMaxHeap() / (1024 * 1024));
  LOGI("  Max stack: %zu MB", MemoryTracker::GetMaxStack() / (1024 * 1024));
  return 0;
}

extern "C" JNIEXPORT jstring JNICALL
Java_com_example_consumeronlywamr_WasmService_runWasm(JNIEnv *env,
                                                      jobject /* this */,
                                                      jbyteArray wasmBytes) {

  // 1. Get bytes from jbyteArray
  jsize len = env->GetArrayLength(wasmBytes);
  jbyte *buffer = env->GetByteArrayElements(wasmBytes, NULL);

  // 2. Load module (parse)
  char error_buf[128];
  wasm_module_t module =
      wasm_runtime_load((uint8_t *)buffer, len, error_buf, sizeof(error_buf));
  if (!module) {
    env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);
    LOGE("Load wasm module failed. error: %s", error_buf);
    std::string error_msg = "Load failed: " + std::string(error_buf);
    return env->NewStringUTF(error_msg.c_str());
  }

  // 2.5 Set WASI parameters
  wasm_runtime_set_wasi_args(module, NULL, 0, NULL, 0, NULL, 0, NULL, 0);

  // Check if we're near memory limit before instantiation
  if (MemoryTracker::IsNearLimit()) {
    LOGE("Cannot instantiate: too close to memory limit");
    wasm_runtime_unload(module);
    env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);
    return env->NewStringUTF("Error: Memory limit reached (RSS > 1.2GB)");
  }

  // 3. Instantiate module with 512MB heap, 16MB stack
  size_t stack_size = MemoryTracker::GetMaxStack(); // 16MB
  size_t heap_size = MemoryTracker::GetMaxHeap();   // 512MB

  LOGI("Instantiating module: heap=%zu MB, stack=%zu MB",
       heap_size / (1024 * 1024), stack_size / (1024 * 1024));

  // Record allocation for tracking
  MemoryTracker::RecordAllocation(heap_size + stack_size);

  // Start execution monitoring
  ExecutionMonitor::StartExecution("app_main", len, heap_size, stack_size);

  wasm_module_inst_t module_inst = wasm_runtime_instantiate(
      module, stack_size, heap_size, error_buf, sizeof(error_buf));
  if (!module_inst) {
    wasm_runtime_unload(module);
    env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);
    LOGE("Instantiate wasm module failed. error: %s", error_buf);
    std::string error_msg = "Instantiation failed: " + std::string(error_buf);
    return env->NewStringUTF(error_msg.c_str());
  }

  // 4. Create execution environment
  wasm_exec_env_t exec_env = wasm_runtime_create_exec_env(module_inst, 8192);
  if (!exec_env) {
    wasm_runtime_deinstantiate(module_inst);
    wasm_runtime_unload(module);
    env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);
    LOGE("Create exec env failed.");
    return env->NewStringUTF("Create exec env failed");
  }

  // 5. Lookup and call entry function (assuming "_start" for now, or finding
  // main) For this example, we'll try to execute default start function or a
  // main function If it's a command model (wasi), _start is usually the entry.
  // If it's a reactor, we might look for specific exports.
  // Let's assume standalone command for now.

  // Check if there is a start function (unlikely for simple calc, but standard
  // for CLI) If no explicit start, we can look for "main" or just let it be.
  // But for "running file", usually implies _start.

  // NOTE: wasm_application_execute_main handles finding _start or main.
  // But since we are using pure runtime API, we should use
  // `wasm_runtime_call_wasm`. However, `wasm_application_execute_main` is part
  // of app-framework which we excluded. So we manually look for _start.

  // 5. Lookup and call entry function : TODO may be differ for other model can
  // we think of some alternatieve Answer: Use 'wasm_application_execute_main'
  // for auto-detection in WASI. For reactors, use dynamic dispatch or metadata
  // sections to map functions to entry points.

  wasm_function_inst_t func =
      wasm_runtime_lookup_function(module_inst, "app_main");
  if (!func) {
    func = wasm_runtime_lookup_function(module_inst, "_start");
  }
  if (!func) {
    func = wasm_runtime_lookup_function(module_inst, "main");
  }
  if (!func) {
    func = wasm_runtime_lookup_function(module_inst, "add");
  }

  const char *result_msg = "Execution success";
  // THINK : this is the reason why add two number works if any function is not
  // required two number as input then what would happen ? Answer: WAMR throws
  // 'invalid argument count' if parameters don't match the signature. We must
  // ensure the argument count in 'call_wasm' matches the WASM export.
  if (func) {
    // Prepare argv buffer for return value (assuming 1 result)
    uint32_t argv[2] = {0};

    // Pass 0 args. Results will be in argv[0]
    if (!wasm_runtime_call_wasm(exec_env, func, 0, argv)) {
      const char *exception = wasm_runtime_get_exception(module_inst);
      LOGE("WASM execution failed: %s",
           exception ? exception : "Unknown error");
      // Allocate string to ensure it persists (though exception ptr usually
      // valid until deinstantiate) For safety in this scope we'll just
      // constructing a std::string to return. But we can't easily change
      // result_msg type here without refactoring. Actually result_msg is const
      // char*. Let's use env->NewStringUTF directly here.
      std::string ex_msg = "Execution failed: " +
                           std::string(exception ? exception : "Unknown error");
      // Cleanup before returning
      wasm_runtime_destroy_exec_env(exec_env);
      wasm_runtime_deinstantiate(module_inst);
      wasm_runtime_unload(module);
      env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);
      return env->NewStringUTF(ex_msg.c_str());
    } else {
      // Assume result is an int (30)
      std::string success_msg =
          "Execution Success! Result: " + std::to_string(argv[0]);
      // Safe to return a new string, but we need to create it before cleanup?
      // No, we create Java string at the end.
      // Let's change result_msg logic slightly.
      // We will return immediately here to avoid the logic below which uses
      // result_msg const char*

      wasm_runtime_destroy_exec_env(exec_env);
      wasm_runtime_deinstantiate(module_inst);
      wasm_runtime_unload(module);
      env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);
      return env->NewStringUTF(success_msg.c_str());
    }
  } else {
    LOGE("No _start or main function found.");
    result_msg = "No entry point found";
  }

  // End execution monitoring
  ExecutionMonitor::Metrics metrics = ExecutionMonitor::EndExecution(0);

  // Cleanup
  wasm_runtime_destroy_exec_env(exec_env);
  wasm_runtime_deinstantiate(module_inst);
  wasm_runtime_unload(module);
  env->ReleaseByteArrayElements(wasmBytes, buffer, JNI_ABORT);

  // Record deallocation
  MemoryTracker::RecordDeallocation(heap_size + stack_size);

  return env->NewStringUTF(result_msg);
}
// Explaination for below lines:
// Answer: This function demonstrates calling specific WASM exports with
// parameters passed from Kotlin, allowing direct interaction with module logic
// beyond a default entry point.

extern "C" JNIEXPORT jint JNICALL
Java_com_example_consumeronlywamr_WasmService_wasmAdd(JNIEnv *env, jobject,
                                                      jbyteArray wasmBytes,
                                                      jint a, jint b) {
  char *buffer = NULL;
  char error_buf[128];
  wasm_module_t module = NULL;
  wasm_module_inst_t module_inst = NULL;
  wasm_exec_env_t exec_env = NULL;
  jint result = -1;

  // 1. Load Wasm bytes : TODO : We have to implement the shared memory concept
  // Answer: Using Direct ByteBuffers or 'mmap' allows the runtime to access
  // memory directly, eliminating the overhead of copying byte arrays between
  // layers.
  jsize length = env->GetArrayLength(wasmBytes);
  buffer = (char *)malloc(length);
  if (!buffer) {
    LOGE("Failed to allocate buffer");
    return -1;
  }
  env->GetByteArrayRegion(wasmBytes, 0, length, (jbyte *)buffer); //

  // 2. Load Module // check for the reuired type
  // Answer: WAMR validates headers automatically. For production, check module
  // types (Interp vs AOT) and verify signatures before loading to ensure
  // security.
  module = wasm_runtime_load((uint8_t *)buffer, length, error_buf,
                             sizeof(error_buf));
  if (!module) {
    LOGE("Load failed: %s", error_buf);
    free(buffer);
    return -1;
  }

  // 2.5 Set WASI parameters (needed for Networking)
  const char *addr_pool[] = {"0.0.0.0/0"};
  const char *ns_lookup_pool[] = {"*"};
  wasm_runtime_set_wasi_addr_pool(module, addr_pool, 1);
  wasm_runtime_set_wasi_ns_lookup_pool(module, ns_lookup_pool, 1);
  wasm_runtime_set_wasi_args(module, NULL, 0, NULL, 0, NULL, 0, NULL, 0);

  // 3. Instantiate // Check : memory size testing, Later On policy checker or
  // health checker implement karenge Answer: Enforce quotas here (CPU, Memory).
  // Stack/heap sizes should be tuned; small values save RAM but complex modules
  // require more space.
  module_inst = wasm_runtime_instantiate(module, 8192, 8192, error_buf,
                                         sizeof(error_buf));
  if (!module_inst) {
    LOGE("Instantiation failed: %s", error_buf);
    wasm_runtime_unload(module);
    free(buffer);
    return -1;
  }

  // 4. Create Execution Environment
  exec_env = wasm_runtime_create_exec_env(module_inst, 8192);
  if (!exec_env) {
    LOGE("Exec env creation failed");
    wasm_runtime_deinstantiate(module_inst);
    wasm_runtime_unload(module);
    free(buffer);
    return -1;
  }

  // 5. Lookup 'add' function // Temporary passing input
  // Answer: Mapping JNI arguments to 'argv' demonstrates specific function
  // invocation. This can be evolved into a generic bridge for any exported
  // function name.
  wasm_function_inst_t func = wasm_runtime_lookup_function(module_inst, "add");
  if (func) {
    // 6. Call with arguments
    uint32_t argv[2];
    argv[0] = a;
    argv[1] = b;

    if (wasm_runtime_call_wasm(exec_env, func, 2, argv)) {
      result = (jint)argv[0];
      LOGI("wasmAdd: %d + %d = %d", a, b, result);
    } else {
      LOGE("wasmAdd execution failed: %s",
           wasm_runtime_get_exception(module_inst));
    }
  } else {
    LOGE("Function 'add' not found");
  }

  // Cleanup
  if (exec_env)
    wasm_runtime_destroy_exec_env(exec_env);
  if (module_inst)
    wasm_runtime_deinstantiate(module_inst);
  if (module)
    wasm_runtime_unload(module);
  free(buffer);

  return result;
}
extern "C" JNIEXPORT jint JNICALL
Java_com_example_consumeronlywamr_WasmService_invokeWasm(JNIEnv *env, jobject,
                                                         jbyteArray wasmBytes,
                                                         jstring funcName,
                                                         jintArray args) {

  const char *nativeFuncName = env->GetStringUTFChars(funcName, NULL);
  jint *nativeArgs = env->GetIntArrayElements(args, NULL);
  jsize argCount = env->GetArrayLength(args);

  char *buffer = NULL;
  char error_buf[128];
  wasm_module_t module = NULL;
  wasm_module_inst_t module_inst = NULL;
  wasm_exec_env_t exec_env = NULL;
  jint result = -1;

  // 1. Load Bytes
  jsize length = env->GetArrayLength(wasmBytes);
  buffer = (char *)malloc(length);
  if (!buffer) {
    LOGE("Failed to allocate buffer");
    goto cleanup;
  }
  env->GetByteArrayRegion(wasmBytes, 0, length, (jbyte *)buffer);

  // 2. Load Module
  module = wasm_runtime_load((uint8_t *)buffer, length, error_buf,
                             sizeof(error_buf));
  if (!module) {
    LOGE("Load failed: %s", error_buf);
    goto cleanup;
  }

  // 3. Instantiate
  module_inst = wasm_runtime_instantiate(module, 8192, 8192, error_buf,
                                         sizeof(error_buf));
  if (!module_inst) {
    LOGE("Instantiation failed: %s", error_buf);
    goto cleanup;
  }

  // 4. Create Exec Env
  exec_env = wasm_runtime_create_exec_env(module_inst, 8192);
  if (!exec_env) {
    LOGE("Exec env creation failed");
    goto cleanup;
  }

  // 5. Dynamic Lookup
  {
    wasm_function_inst_t func =
        wasm_runtime_lookup_function(module_inst, nativeFuncName);
    if (func) {
      // Create argv buffer. WAMR needs space for both args and result.
      // Since we expect 1 result, we need max(argCount, 1).
      uint32_t *argv =
          (uint32_t *)malloc(sizeof(uint32_t) * (argCount > 0 ? argCount : 1));
      for (int i = 0; i < argCount; i++) {
        argv[i] = (uint32_t)nativeArgs[i];
      }

      if (wasm_runtime_call_wasm(exec_env, func, argCount, argv)) {
        result = (jint)argv[0];
        LOGI("invokeWasm: %s executed. Result: %d", nativeFuncName, result);
      } else {
        LOGE("Execution failed: %s", wasm_runtime_get_exception(module_inst));
      }
      free(argv);
    } else {
      LOGE("Function '%s' not found in WASM module", nativeFuncName);
    }
  }

cleanup:
  if (exec_env)
    wasm_runtime_destroy_exec_env(exec_env);
  if (module_inst)
    wasm_runtime_deinstantiate(module_inst);
  if (module)
    wasm_runtime_unload(module);
  if (buffer)
    free(buffer);

  env->ReleaseStringUTFChars(funcName, nativeFuncName);
  env->ReleaseIntArrayElements(args, nativeArgs, JNI_ABORT);

  return result;
}
