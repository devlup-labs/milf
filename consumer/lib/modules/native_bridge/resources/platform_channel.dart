import 'package:flutter/services.dart';

class NativeBridge {
  static const MethodChannel _channel = MethodChannel(
    'com.example.consumer/bridge',
  );

  Future<void> reqProcessAllocation(Map<String, dynamic> jobMetadata) async {
    try {
      await _channel.invokeMethod('reqProcessAllocation', jobMetadata);
    } on PlatformException catch (e) {
      // Handle error
      print("Failed to allocate process: '${e.message}'.");
    }
  }

  Future<void> sendFuncJob(
    String jobId,
    String funcId,
    Uint8List wasmBinary,

    Map<String, dynamic> inputPayload,
    Map<String, dynamic> metadata,
  ) async {
    try {
      await _channel.invokeMethod('sendFuncJob', {
        'jobId': jobId,
        'funcId': funcId,
        'wasmBinary': wasmBinary,
        'inputPayload': inputPayload,
        'metadata': metadata,
      });
    } on PlatformException catch (e) {
      print("Failed to send func job: '${e.message}'.");
    }
  }

  Future<Map<String, dynamic>?> getOSResources() async {
    try {
      final result = await _channel.invokeMethod<Map<dynamic, dynamic>>(
        'getOSResources',
      );
      return result?.cast<String, dynamic>();
    } on PlatformException catch (e) {
      print("Failed to get OS resources: '${e.message}'.");
      return null;
    }
  }

  Future<Map<String, dynamic>?> runWasmTest(
    Uint8List wasmBinary,
    Map<String, dynamic> input,
    int memoryLimitMB,
  ) async {
    try {
      final result = await _channel.invokeMapMethod<String, dynamic>(
        'runWasmTest',
        {
          'wasmBinary': wasmBinary,
          'input': input,
          'memoryLimitMB': memoryLimitMB,
        },
      );
      return result;
    } on PlatformException catch (e) {
      print("Failed to run WASM test: '${e.message}'.");
      return null;
    }
  }
}
