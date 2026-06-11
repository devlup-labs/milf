import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;
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
  dynamic output;
  bool? success;
  String? errorMessage;

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

  bool get isConnected =>
      status == NodeStatus.online || status == NodeStatus.executing;

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
    final hms =
        '${timestamp.hour.toString().padLeft(2, '0')}:'
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
    final wasmEvent = WasmEvent(
      executionId: executionId,
      lambdaId: lambdaId,
      wasmSizeBytes: wasmBytes.length,
      payloadSizeBytes: payload.length,
      payloadType: payloadType,
    );
    wasmEvents.insert(0, wasmEvent);
    notifyListeners();

    try {
      // 1. Try to parse JSON and see if it's a simple a,b addition task
      Map<String, dynamic>? jsonPayload;
      try {
        final rawString = utf8.decode(payload);
        _log('RAW PAYLOAD RECEIVED: $rawString');
        jsonPayload = jsonDecode(rawString) as Map<String, dynamic>;
      } catch (e) {
        _log('Failed to parse payload as JSON: $e');
      }

      Future<dynamic> methodCall;
      // Extract optional function name hint (reserved key '_func')
      final String funcName =
          (jsonPayload != null && jsonPayload.containsKey('_func'))
          ? jsonPayload['_func'].toString()
          : 'wasm_main';

      // Filter out reserved keys to get user-supplied parameters
      final paramPayload = jsonPayload != null
          ? Map<String, dynamic>.fromEntries(
              jsonPayload.entries.where((e) => !e.key.startsWith('_')),
            )
          : <String, dynamic>{};

      if (paramPayload.isNotEmpty) {
        // Check if ANY parameter requires string/complex data type handling
        bool hasComplexTypes = false;
        final List<int> intArgs = [];

        for (final value in paramPayload.values) {
          if (value is int) {
            intArgs.add(value);
          } else if (value is num) {
            intArgs.add(value.toInt());
          } else {
            // Any string, object, array, or boolean triggers complex mode
            hasComplexTypes = true;
            break;
          }
        }

        if (hasComplexTypes) {
          // Serialize the whole payload mapping for the C string interface
          final stringPayload = jsonEncode(paramPayload);
          _log('String/JSON task → $funcName Payload: $stringPayload');
          methodCall = _platform.invokeMethod('invokeWasmString', {
            'bytes': wasmBytes,
            'funcName': funcName,
            'payload': stringPayload,
          });
        } else {
          // Pure integer function (original fast path)
          _log(
            'Int-args task (${intArgs.length} params): $intArgs → $funcName',
          );
          methodCall = _platform.invokeMethod('invokeWasm', {
            'bytes': wasmBytes,
            'funcName': funcName,
            'args': Int32List.fromList(intArgs),
          });
        }
      } else {
        // No user parameters — call the function with zero args
        _log('Zero-arg task → $funcName');
        methodCall = _platform.invokeMethod('invokeWasm', {
          'bytes': wasmBytes,
          'funcName': funcName,
          'args': Int32List(0),
        });
      }

      final dynamic result = await methodCall.timeout(
        const Duration(seconds: 30),
        onTimeout: () => throw Exception('Execution timed out after 30s'),
      );

      dynamic finalOutput = result;
      if (result is String && result.startsWith('FILE:')) {
        final fileName = result.substring(5);
        _log('Result is a file reference: $fileName. Uploading to server...');
        try {
          final fileBytes = await _platform.invokeMethod<Uint8List>('readLocalFile', {'name': fileName});
          if (fileBytes != null && fileBytes.isNotEmpty) {
            // Upload file to server via HTTP POST
            var uploadUrl = '$serverUrl/api/v1/files';
            if (Platform.isAndroid) {
              uploadUrl = uploadUrl.replaceAll('localhost', '10.0.2.2');
            }
            final request = http.MultipartRequest('POST', Uri.parse(uploadUrl));
            request.files.add(http.MultipartFile.fromBytes(
              'file',
              fileBytes,
              filename: fileName,
            ));
            final response = await request.send();
            final respBody = await response.stream.bytesToString();
            if (response.statusCode == 201) {
              final respJson = jsonDecode(respBody);
              finalOutput = {
                'file_id': respJson['file_id'],
                'filename': respJson['filename'],
                'download_url': respJson['download_url'],
                'size': respJson['size'],
                'content_type': respJson['content_type'],
              };
              _log('File uploaded! ID: ${respJson['file_id']} (${fileBytes.length} bytes)');
            } else {
              _log('File upload failed (${response.statusCode}): $respBody');
            }
          }
        } catch (e) {
          _log('Failed to upload result file $fileName: $e');
        }
      }

      _sync?.sendResult(executionId, success: true, output: finalOutput);
      _log('Task $executionId succeeded.');
      wasmEvent.success = true;
      wasmEvent.output = finalOutput;
      history.insert(
        0,
        ExecutionRecord(
          executionId: executionId,
          lambdaId: lambdaId,
          success: true,
          message: 'Success',
        ),
      );
      executionSuccess++;
    } on PlatformException catch (e) {
      final msg = e.message ?? 'Native error';
      _sync?.sendResult(executionId, success: false, error: msg);
      _log('Task $executionId FAILED: $msg');
      wasmEvent.success = false;
      wasmEvent.errorMessage = msg;
      history.insert(
        0,
        ExecutionRecord(
          executionId: executionId,
          lambdaId: lambdaId,
          success: false,
          message: msg,
        ),
      );
      executionFailed++;
    } catch (e) {
      _sync?.sendResult(executionId, success: false, error: e.toString());
      _log('Task $executionId ERROR: $e');
      wasmEvent.success = false;
      wasmEvent.errorMessage = e.toString();
      history.insert(
        0,
        ExecutionRecord(
          executionId: executionId,
          lambdaId: lambdaId,
          success: false,
          message: e.toString(),
        ),
      );
      executionFailed++;
    } finally {
      _setStatus(isConnected ? NodeStatus.online : NodeStatus.idle);
      notifyListeners();
    }
  }

  Future<void> testFetchAndPdf() async {
    _setStatus(NodeStatus.executing);
    _log('Testing Fetch & PDF conversion...');

    try {
      // 1. Read test_fetch.wasm from local storage
      final bytes = await _platform.invokeMethod<Uint8List>('readLocalFile', {
        'name': 'test_fetch.wasm',
      });
      if (bytes == null || bytes.isEmpty) {
        throw Exception('Failed to read test_fetch.wasm — did you push it?');
      }

      _log('Read ${bytes.length} bytes of WASM. Invoking...');

      // 2. Clear payload
      final payload = Uint8List(0);

      // 3. Call invokeDataWasm
      final pdfBytes = await _platform
          .invokeMethod<Uint8List>('invokeDataWasm', {
            'bytes': bytes,
            'funcName': 'wasm_main',
            'payload': payload,
            'hostPath': 'test_fetch.wasm',
          });

      if (pdfBytes != null && pdfBytes.length > 5) {
        _log('SUCCESS! Generated ${pdfBytes.length} byte PDF.');
        final header = String.fromCharCodes(pdfBytes.take(4));
        _log('PDF Header: $header');
        executionSuccess++;
      } else {
        _log('Failed or returned invalid data: ${pdfBytes?.length ?? 0} bytes');
        executionFailed++;
      }
    } catch (e) {
      _log('Test Error: $e');
      executionFailed++;
    } finally {
      _setStatus(isConnected ? NodeStatus.online : NodeStatus.idle);
      notifyListeners();
    }
  }
}
