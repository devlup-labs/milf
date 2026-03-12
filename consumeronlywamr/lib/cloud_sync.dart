import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:path_provider/path_provider.dart';
import 'package:flutter/services.dart';

class CloudSync {
  static const platform = MethodChannel('com.example.consumeronlywamr/wasm');
  
  final String serverUrl;
  String? sinkId;
  WebSocketChannel? _channel;
  bool isConnected = false;
  final Function(String) onLog;

  CloudSync({required this.serverUrl, required this.onLog});

  Future<void> connect() async {
    try {
      // 1. Register Sink to get ID
      onLog("Registering sink with $serverUrl...");
      final regRes = await http.post(
        Uri.parse('$serverUrl/api/v1/sinks/register'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({
          'email': 'node_${DateTime.now().millisecondsSinceEpoch}@milf.local',
          'password': 'password123',
          'endpoint': 'ws-connected-node'
        }),
      );

      if (regRes.statusCode == 201) {
        final data = jsonDecode(regRes.body);
        sinkId = data['sink_id'];
        onLog("Registered! SinkID: $sinkId");
      } else {
        onLog("Failed to register: ${regRes.body}");
        return;
      }

      // 2. Connect WebSocket
      final wsUrl = serverUrl.replaceFirst('http', 'ws');
      final uri = Uri.parse('$wsUrl/api/v1/sinks/ws?sinkId=$sinkId');
      
      onLog("Connecting to WebSocket: $uri");
      _channel = WebSocketChannel.connect(uri);
      isConnected = true;

      // 3. Listen for incoming Tasks
      _channel!.stream.listen((message) {
        _handleMessage(message);
      }, onDone: () {
        isConnected = false;
        onLog("WebSocket disconnected.");
      }, onError: (error) {
        isConnected = false;
        onLog("WebSocket error: $error");
      });

      // Send initial heartbeat
      _sendHeartbeat();

    } catch (e) {
      isConnected = false;
      onLog("Connection error: $e");
    }
  }

  void disconnect() {
    _channel?.sink.close();
    isConnected = false;
    onLog("Disconnected from server.");
  }

  void _sendHeartbeat() {
    if (!isConnected) return;
    final msg = {
      'type': 'heartbeat',
      'payload': {
        'sink_id': sinkId,
        'ram_available_mb': 2048,
        'storage_available_mb': 10240,
      }
    };
    _channel!.sink.add(jsonEncode(msg));
  }

  Future<void> _handleMessage(dynamic message) async {
    try {
      final msg = jsonDecode(message);
      if (msg['type'] == 'task_assignment') {
        final payload = msg['payload'];
        final executionId = payload['execution_id'];
        final lambdaId = payload['lambda_id']; // Using Lambda ID to fetch the WASM
        final input = payload['input'];

        onLog("Received Task $executionId! LambdaID: $lambdaId");

        // 1. Download WASM from Central Server
        // Make sure lambdaId is sent in payload to fetch binary correctly
        final wasmBytes = await _downloadFile('$serverUrl/api/v1/lambdas/$lambdaId/wasm');
        if (wasmBytes == null) {
          _sendResult(executionId, false, error: "Failed to download WASM");
          return;
        }

        // 2. Prepare payload (convert input to bytes depending on WASM expectations)
        // For simplicity, converting stringified JSON input to bytes, or if it's base64 data, decode it.
        Uint8List taskPayload;
        if (input != null && input['data'] != null) {
           taskPayload = base64Decode(input['data']);
        } else {
           taskPayload = Uint8List.fromList(utf8.encode(jsonEncode(input)));
        }

        onLog("Executing WASM...");
        
        // 3. Invoke WASM via Platform Channel
        final dynamic result = await platform.invokeMethod('invokeDataWasm', {
          'bytes': wasmBytes,
          'funcName': 'invoke', // Default generic entrypoint
          'payload': taskPayload,
        });

        // 4. Send Result
        if (result is Uint8List) {
          // Send back as base64 string
          final b64Result = base64Encode(result);
          _sendResult(executionId, true, output: {'data': b64Result});
        } else {
          _sendResult(executionId, true, output: {'result': result.toString()});
        }

      }
    } catch (e) {
      onLog("Error handling message: $e");
    }
  }

  Future<Uint8List?> _downloadFile(String url) async {
    try {
      // Handle local test urls mapping to host
      String fetchUrl = url;
      if (fetchUrl.contains('localhost') && Platform.isAndroid) {
         fetchUrl = fetchUrl.replaceAll('localhost', '10.0.2.2');
      }
      final response = await http.get(Uri.parse(fetchUrl));
      if (response.statusCode == 200) {
        return response.bodyBytes;
      }
      onLog("Failed to download: ${response.statusCode}");
    } catch (e) {
      onLog("Download error: $e");
    }
    return null;
  }

  void _sendResult(String executionId, bool success, {dynamic output, String? error}) {
    if (!isConnected) return;
    
    final payload = {
      'execution_id': executionId,
      'success': success,
      'output': output,
      'error': error,
    };

    final msg = {
      'type': 'task_result',
      'payload': payload
    };

    _channel!.sink.add(jsonEncode(msg));
    onLog("Sent result for $executionId. Success: $success");
    
    // Send heartbeat to indicate ready
    _sendHeartbeat();
  }
}
