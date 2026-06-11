import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/web_socket_channel.dart';

typedef TaskCallback =
    Future<void> Function({
      required String executionId,
      required String lambdaId,
      required Uint8List wasmBytes,
      required Uint8List payload,
    });

/// Decodes the typed input envelope sent by the server.
/// The server MUST tag the input type to disambiguate:
///   {"type": "json",   "data": {...}}   → serialize to UTF-8 bytes
///   {"type": "binary", "data": "<b64>"} → decode from base64 bytes
///   {"type": "null"}                    → empty payload
Uint8List _decodeTaskInput(Map<String, dynamic>? input) {
  if (input == null || input['type'] == 'null') return Uint8List(0);
  if (input['type'] == 'binary') {
    return base64Decode(input['data'] as String);
  }
  // Default: treat as JSON → UTF-8 bytes
  return Uint8List.fromList(utf8.encode(jsonEncode(input['data'] ?? {})));
}

/// Hardened WebSocket manager for the MILF node.
/// Handles: JWT auth, auto-reconnect with exponential backoff, periodic heartbeat.
class CloudSync {
  final String serverUrl;
  final String authToken;
  final void Function(String) onLog;
  final void Function(String) onSinkRegistered;
  final TaskCallback onTaskReceived;
  final VoidCallback onDisconnected;

  String? _sinkId;
  WebSocketChannel? _channel;
  bool _intentionalDisconnect = false;
  bool isConnected = false;

  int _retryDelaySecs = 2;
  Timer? _heartbeatTimer;

  CloudSync({
    required this.serverUrl,
    required this.authToken,
    required this.onLog,
    required this.onSinkRegistered,
    required this.onTaskReceived,
    required this.onDisconnected,
  });

  Future<void> connect() async {
    _intentionalDisconnect = false;
    await _register();
  }

  void disconnect() {
    _intentionalDisconnect = true;
    _heartbeatTimer?.cancel();
    _channel?.sink.close();
    isConnected = false;
    onDisconnected();
  }

  // ── Registration ──────────────────────────────────────────────────────────

  Future<void> _register() async {
    try {
      onLog('Registering node with server...');
      final headers = <String, String>{
        'Content-Type': 'application/json',
        if (authToken.isNotEmpty) 'Authorization': 'Bearer $authToken',
      };

      final res = await http.post(
        Uri.parse('$serverUrl/api/v1/sinks/register'),
        headers: headers,
        body: jsonEncode({
          'email': 'node_primary@milf.local',
          'password': 'unused',
          'endpoint': 'ws-node',
        }),
      );

      if (res.statusCode == 201) {
        final data = jsonDecode(res.body);
        _sinkId = data['sink_id'] as String?;
        if (_sinkId == null) {
          onLog('Registration error: sink_id missing in response');
          return;
        }
        onLog('Registered. SinkID: $_sinkId');
        onSinkRegistered(_sinkId!);
        _openWebSocket();
      } else {
        onLog('Registration failed (${res.statusCode}): ${res.body}');
        _scheduleReconnect();
      }
    } catch (e) {
      onLog('Registration error: $e');
      _scheduleReconnect();
    }
  }

  // ── WebSocket ─────────────────────────────────────────────────────────────

  void _openWebSocket() {
    if (_sinkId == null) return;

    final wsBase = serverUrl.replaceFirst(RegExp(r'^http'), 'ws');
    final uri = Uri.parse('$wsBase/api/v1/sinks/ws?sinkId=$_sinkId');
    onLog('Opening WebSocket: $uri');

    _channel = WebSocketChannel.connect(uri);
    isConnected = true;
    _retryDelaySecs = 2; // reset backoff on success

    _channel!.stream.listen(
      _handleMessage,
      onDone: _handleDisconnect,
      onError: (e) {
        onLog('WebSocket error: $e');
        _handleDisconnect();
      },
      cancelOnError: true,
    );

    _startHeartbeat();
  }

  void _handleDisconnect() {
    isConnected = false;
    _heartbeatTimer?.cancel();
    if (_intentionalDisconnect) return;
    onLog('WebSocket disconnected. Reconnecting...');
    onDisconnected();
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_intentionalDisconnect) return;
    onLog('Retrying in ${_retryDelaySecs}s...');
    Future.delayed(Duration(seconds: _retryDelaySecs), _register);
    _retryDelaySecs = (_retryDelaySecs * 2).clamp(2, 60);
  }

  // ── Heartbeat ─────────────────────────────────────────────────────────────

  void _startHeartbeat() {
    _heartbeatTimer?.cancel();
    _sendHeartbeat(); // immediate on connect
    _heartbeatTimer = Timer.periodic(
      const Duration(seconds: 30),
      (_) => _sendHeartbeat(),
    );
  }

  void _sendHeartbeat() {
    if (!isConnected) return;
    _channel!.sink.add(
      jsonEncode({
        'type': 'heartbeat',
        'payload': {
          'sink_id': _sinkId,
          'ram_available_mb': 2048,
          'storage_available_mb': 10240,
        },
      }),
    );
  }

  // ── Message Handling ──────────────────────────────────────────────────────

  Future<void> _handleMessage(dynamic raw) async {
    try {
      final msg = jsonDecode(raw as String) as Map<String, dynamic>;
      if (msg['type'] != 'task_assignment') return;

      final p = msg['payload'] as Map<String, dynamic>;
      final executionId = p['execution_id'] as String;
      final lambdaId = p['lambda_id'] as String;
      final rawInput = p['payload'] as Map<String, dynamic>?;

      onLog('Received task $executionId (lambda: $lambdaId)');
      Uint8List? wasmBytes;

      // 1. Try to use inlined WASM bytes first
      final inlinedB64 = p['wasm_base64'] as String?;
      if (inlinedB64 != null && inlinedB64.isNotEmpty) {
        try {
          wasmBytes = base64Decode(inlinedB64);
          onLog('Using inlined WASM ($executionId)');
        } catch (e) {
          onLog('Failed to decode inlined WASM: $e');
        }
      }

      // 2. Fallback: Download via HTTP if no inlined data or decoding failed
      if (wasmBytes == null) {
        onLog('Downloading WASM for $executionId...');
        wasmBytes = await _downloadWasm(lambdaId);
      }

      if (wasmBytes == null) {
        sendResult(
          executionId,
          success: false,
          error: 'Failed to acquire WASM binary',
        );
        return;
      }

      // 2. Decode typed input
      final payload = _decodeTaskInput(rawInput);

      // 3. Delegate execution to NodeController
      await onTaskReceived(
        executionId: executionId,
        lambdaId: lambdaId,
        wasmBytes: wasmBytes,
        payload: payload,
      );
    } catch (e) {
      onLog('Message handling error: $e');
    }
  }

  Future<Uint8List?> _downloadWasm(String lambdaId) async {
    try {
      String url = '$serverUrl/api/v1/lambdas/$lambdaId/wasm';
      if (Platform.isAndroid) {
        url = url.replaceAll('localhost', '10.0.2.2');
      }
      final res = await http.get(
        Uri.parse(url),
        headers: {
          if (authToken.isNotEmpty) 'Authorization': 'Bearer $authToken',
        },
      );
      if (res.statusCode == 200) return res.bodyBytes;
      onLog('WASM download failed (${res.statusCode})');
    } catch (e) {
      onLog('WASM download error: $e');
    }
    return null;
  }

  // ── Result Reporting ──────────────────────────────────────────────────────

  void sendResult(
    String executionId, {
    required bool success,
    dynamic output,
    String? error,
  }) {
    if (!isConnected) {
      onLog('Cannot send result: not connected');
      return;
    }

    dynamic serializedOutput;
    if (output is Uint8List) {
      serializedOutput = {'data': base64Encode(output)};
    } else if (output is Map) {
      serializedOutput = output;
    } else if (output != null) {
      serializedOutput = {'result': output.toString()};
    }

    _channel!.sink.add(
      jsonEncode({
        'type': 'task_result',
        'payload': {
          'execution_id': executionId,
          'success': success,
          if (serializedOutput != null) 'output': serializedOutput,
          if (error != null) 'error': error,
        },
      }),
    );

    _sendHeartbeat(); // signal ready for next task
  }
}
