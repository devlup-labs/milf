import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'cloud_sync.dart';

/// Tracks a single WASM execution for the history log.
class ExecutionRecord {
  final String executionId;
  final String lambdaId;
  final bool success;
  final String message;
  final DateTime timestamp;

  ExecutionRecord({
    required this.executionId,
    required this.lambdaId,
    required this.success,
    required this.message,
  }) : timestamp = DateTime.now();
}

/// Tracks an incoming WASM dispatch event for the WASM Info tab.
class WasmEvent {
  final String executionId;
  final String lambdaId;
  final int wasmSizeBytes;
  final int payloadSizeBytes;
  final String payloadType;
  final DateTime timestamp;

  WasmEvent({
    required this.executionId,
    required this.lambdaId,
    required this.wasmSizeBytes,
    required this.payloadSizeBytes,
    required this.payloadType,
  }) : timestamp = DateTime.now();
}

/// The single source of truth for all app state.
/// The UI only observes this — it never has direct logic.
enum NodeStatus { idle, connecting, online, executing, error }

class NodeController extends ChangeNotifier {
  static const _platform = MethodChannel('com.example.consumeronlywamr/wasm');

  // ── Configuration ─────────────────────────────────────────────────────────
  String serverUrl = 'http://10.0.2.2:8080';
  String authToken = '';

  // ── Reactive State ────────────────────────────────────────────────────────
  NodeStatus status = NodeStatus.idle;
  String? sinkId;
  String logBuffer = 'Node ready. Press Connect to start.\n';
  int executionSuccess = 0;
  int executionFailed = 0;
  List<ExecutionRecord> history = [];
  List<WasmEvent> wasmEvents = [];

  // ── Internal ──────────────────────────────────────────────────────────────
  CloudSync? _sync;

  bool get isConnected => status == NodeStatus.online || status == NodeStatus.executing;

  // ── Public API ────────────────────────────────────────────────────────────

  void configure({String? url, String? token}) {
    if (url != null) serverUrl = url;
    if (token != null) authToken = token;
    notifyListeners();
  }

  void connect() {
    if (isConnected) return;
    _setStatus(NodeStatus.connecting);
    _log('Connecting to $serverUrl...');

    _sync = CloudSync(
      serverUrl: serverUrl,
      authToken: authToken,
      onLog: _log,
      onSinkRegistered: (id) {
        sinkId = id;
        _setStatus(NodeStatus.online);
        _log('Online. SinkID: $id');
        notifyListeners();
      },
      onTaskReceived: _executeTask,
      onDisconnected: () {
        _setStatus(NodeStatus.error);
        _log('Disconnected from server.');
        notifyListeners();
      },
    );
    _sync!.connect();
  }

  void disconnect() {
    _sync?.disconnect();
    _sync = null;
    sinkId = null;
    _setStatus(NodeStatus.idle);
    _log('Disconnected.');
  }

  // ── Private ───────────────────────────────────────────────────────────────

  void _setStatus(NodeStatus s) {
    status = s;
    notifyListeners();
  }

  void _log(String msg) {
    final timestamp = DateTime.now();
    final hms = '${timestamp.hour.toString().padLeft(2, '0')}:'
        '${timestamp.minute.toString().padLeft(2, '0')}:'
        '${timestamp.second.toString().padLeft(2, '0')}';
    logBuffer += '[$hms] $msg\n';
    notifyListeners();
  }

  /// Core execution loop: called by CloudSync when a task_assignment arrives.
  Future<void> _executeTask({
    required String executionId,
    required String lambdaId,
    required Uint8List wasmBytes,
    required Uint8List payload,
  }) async {
    _setStatus(NodeStatus.executing);
    _log('Executing task $executionId (lambda: $lambdaId)...');

    // Record incoming WASM event for the WASM Info tab
    final payloadType = payload.isEmpty ? 'null' : 'binary';
    wasmEvents.insert(0, WasmEvent(
      executionId: executionId,
      lambdaId: lambdaId,
      wasmSizeBytes: wasmBytes.length,
      payloadSizeBytes: payload.length,
      payloadType: payloadType,
    ));
    notifyListeners();

    try {
      final dynamic result = await _platform
          .invokeMethod('invokeDataWasm', {
            'bytes': wasmBytes,
            'funcName': 'invoke',
            'payload': payload,
          })
          .timeout(
            const Duration(seconds: 30),
            onTimeout: () => throw Exception('Execution timed out after 30s'),
          );

      _sync?.sendResult(executionId, success: true, output: result);
      _log('Task $executionId succeeded.');
      history.insert(0, ExecutionRecord(
        executionId: executionId,
        lambdaId: lambdaId,
        success: true,
        message: 'Success',
      ));
      executionSuccess++;
    } on PlatformException catch (e) {
      final msg = e.message ?? 'Native error';
      _sync?.sendResult(executionId, success: false, error: msg);
      _log('Task $executionId FAILED: $msg');
      history.insert(0, ExecutionRecord(
        executionId: executionId,
        lambdaId: lambdaId,
        success: false,
        message: msg,
      ));
      executionFailed++;
    } catch (e) {
      _sync?.sendResult(executionId, success: false, error: e.toString());
      _log('Task $executionId ERROR: $e');
      history.insert(0, ExecutionRecord(
        executionId: executionId,
        lambdaId: lambdaId,
        success: false,
        message: e.toString(),
      ));
      executionFailed++;
    } finally {
      _setStatus(isConnected ? NodeStatus.online : NodeStatus.idle);
      notifyListeners();
    }
  }
}
