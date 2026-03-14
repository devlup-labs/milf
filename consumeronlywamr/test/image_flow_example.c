/**
 * COMPLETE IMAGE GRAYSCALE EXAMPLE - How Data Flows
 *
 * This demonstrates the ENTIRE flow from Flutter to WASM and back
 * WITHOUT needing image libraries in C++/WAMR!
 */

// ============================================================================
// PART 1: Flutter/Dart Side (lib/image_processor.dart)
// ============================================================================
/*
import 'dart:typed_data';
import 'package:image/image.dart' as img;  // <-- Image library ONLY here!
import 'package:flutter/services.dart';

class ImageProcessor {
  static const platform = MethodChannel('com.example.consumeronlywamr/wasm');

  /// Convert image to grayscale using WASM
  Future<Uint8List> convertToGrayscale(Uint8List imageBytes) async {
    // 1. Decode image to get raw RGB bytes
    img.Image? image = img.decodeImage(imageBytes);
    if (image == null) throw Exception('Invalid image');

    // 2. Extract raw RGB data (what WASM needs)
    Uint8List rgbBytes = Uint8List(image.width * image.height * 3);
    int index = 0;

    for (int y = 0; y < image.height; y++) {
      for (int x = 0; x < image.width; x++) {
        img.Pixel pixel = image.getPixel(x, y);
        rgbBytes[index++] = pixel.r.toInt();
        rgbBytes[index++] = pixel.g.toInt();
        rgbBytes[index++] = pixel.b.toInt();
      }
    }

    // 3. Load WASM file (grayscale.wasm)
    Uint8List wasmBytes = await rootBundle.load('assets/grayscale.wasm')
        .then((data) => data.buffer.asUint8List());

    // 4. Call native WAMR to execute
    final result = await platform.invokeMethod('processImage', {
      'wasmBytes': wasmBytes,
      'imageData': rgbBytes,
      'width': image.width,
      'height': image.height,
    });

    // 5. Result is grayscale bytes - convert back to image
    Uint8List grayBytes = result as Uint8List;

    // Create grayscale image
    img.Image grayImage = img.Image(image.width, image.height);
    int grayIndex = 0;

    for (int y = 0; y < image.height; y++) {
      for (int x = 0; x < image.width; x++) {
        int gray = grayBytes[grayIndex++];
        grayImage.setPixelRgb(x, y, gray, gray, gray);
      }
    }

    // 6. Encode back to PNG/JPEG
    return Uint8List.fromList(img.encodePng(grayImage));
  }
}
*/

// ============================================================================
// PART 2: Native C++ Side (native-lib.cpp)
// ============================================================================
/*
#include "wasm_export.h"
#include <jni.h>
#include <android/log.h>

// NO IMAGE LIBRARIES IMPORTED HERE! ✅

extern "C" JNIEXPORT jbyteArray JNICALL
Java_com_example_consumeronlywamr_WasmService_processImage(
    JNIEnv *env, jobject,
    jbyteArray wasmBytes,
    jbyteArray imageData,
    jint width,
    jint height) {

    // 1. Load WASM module (same as before)
    jsize wasm_len = env->GetArrayLength(wasmBytes);
    jbyte* wasm_buffer = env->GetByteArrayElements(wasmBytes, NULL);

    char error_buf[128];
    wasm_module_t module = wasm_runtime_load(
        (uint8_t*)wasm_buffer, wasm_len, error_buf, sizeof(error_buf));

    if (!module) {
        env->ReleaseByteArrayElements(wasmBytes, wasm_buffer, JNI_ABORT);
        return NULL;
    }

    // 2. Instantiate module
    wasm_module_inst_t module_inst = wasm_runtime_instantiate(
        module, 8 * 1024 * 1024, 256 * 1024 * 1024, error_buf,
sizeof(error_buf));

    if (!module_inst) {
        wasm_runtime_unload(module);
        env->ReleaseByteArrayElements(wasmBytes, wasm_buffer, JNI_ABORT);
        return NULL;
    }

    // 3. Get image data bytes
    jsize data_len = env->GetArrayLength(imageData);
    jbyte* data_buffer = env->GetByteArrayElements(imageData, NULL);

    // 4. Allocate memory in WASM for input
    wasm_exec_env_t exec_env = wasm_runtime_create_exec_env(module_inst, 8192);

    uint32_t input_offset = wasm_runtime_module_malloc(
        module_inst, data_len, (void**)&input_ptr);

    // Copy image data into WASM memory
    memcpy(input_ptr, data_buffer, data_len);

    // 5. Allocate output buffer
    uint32_t output_size = width * height;  // Grayscale = 1 byte per pixel
    uint32_t output_offset = wasm_runtime_module_malloc(
        module_inst, output_size, (void**)&output_ptr);

    // 6. Call WASM function
    wasm_function_inst_t func = wasm_runtime_lookup_function(
        module_inst, "convert_to_grayscale");

    if (func) {
        uint32_t argv[4];
        argv[0] = input_offset;   // RGB data pointer
        argv[1] = width;
        argv[2] = height;
        argv[3] = output_offset;  // Output pointer

        bool success = wasm_runtime_call_wasm(exec_env, func, 4, argv);

        if (success) {
            // 7. Copy result back to Java
            jbyteArray result = env->NewByteArray(output_size);
            env->SetByteArrayRegion(result, 0, output_size, (jbyte*)output_ptr);

            // Cleanup
            wasm_runtime_module_free(module_inst, input_offset);
            wasm_runtime_module_free(module_inst, output_offset);
            wasm_runtime_destroy_exec_env(exec_env);
            wasm_runtime_deinstantiate(module_inst);
            wasm_runtime_unload(module);
            env->ReleaseByteArrayElements(wasmBytes, wasm_buffer, JNI_ABORT);
            env->ReleaseByteArrayElements(imageData, data_buffer, JNI_ABORT);

            return result;  // <-- Grayscale bytes returned!
        }
    }

    // Cleanup on error
    wasm_runtime_module_free(module_inst, input_offset);
    wasm_runtime_module_free(module_inst, output_offset);
    wasm_runtime_destroy_exec_env(exec_env);
    wasm_runtime_deinstantiate(module_inst);
    wasm_runtime_unload(module);
    env->ReleaseByteArrayElements(wasmBytes, wasm_buffer, JNI_ABORT);
    env->ReleaseByteArrayElements(imageData, data_buffer, JNI_ABORT);

    return NULL;
}
*/

// ============================================================================
// PART 3: WASM Side (grayscale.c) - Already created!
// ============================================================================
// See test/grayscale.c - it's completely self-contained!
// No external dependencies, just pure C math

// ============================================================================
// SUMMARY: Where Image Libraries Are Needed
// ============================================================================

/*
┌─────────────────────────────────────────────────────────────┐
│                         FLUTTER/DART                         │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ image library: Decode PNG/JPEG → Raw RGB bytes       │  │
│  │ image library: Encode Raw bytes → PNG/JPEG           │  │
│  └───────────────────────────────────────────────────────┘  │
│                            ↓ Raw RGB bytes                   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      NATIVE C++ (WAMR)                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ NO image libraries needed! ✅                         │  │
│  │ Just: Load WASM, Execute, Return bytes                │  │
│  └───────────────────────────────────────────────────────┘  │
│                            ↓ Execution                       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                      WASM (grayscale.c)                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ NO image libraries needed! ✅                         │  │
│  │ Just: Math operations on RGB bytes                    │  │
│  │ Formula: Gray = 0.299*R + 0.587*G + 0.114*B          │  │
│  └───────────────────────────────────────────────────────┘  │
│                            ↓ Grayscale bytes                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
                    Back to Flutter for encoding
*/

// ============================================================================
// KEY INSIGHT
// ============================================================================
/*
The WASM module treats the image as just an array of numbers!

Input:  [255, 0, 0, 0, 255, 0, ...]  (RGB bytes)
            ↓
Process: Math operations
            ↓
Output: [76, 150, 29, ...]  (Grayscale bytes)

It doesn't care that these numbers represent an image!
This is why it's a SINGLE BINARY - no dependencies!
*/

// ============================================================================
// COMPILE TO WASM
// ============================================================================
/*
# Just compile the grayscale.c file:
wasi-sdk/bin/clang \
  --target=wasm32-wasi \
  -O2 \
  -o grayscale.wasm \
  grayscale.c

# Result: Single ~2-3KB WASM file
# Contains ALL the logic, no external dependencies!
*/
