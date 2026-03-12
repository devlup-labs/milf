import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:file_picker/file_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:open_filex/open_filex.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';
import 'cloud_sync.dart'; // Import Cloud Sync

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'WAMR Consumer',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color.fromARGB(255, 166, 204, 235),
        ),
        useMaterial3: true,
      ),
      home: const MyHomePage(title: 'WAMR Sandbox Runner'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});
  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  static const platform = MethodChannel('com.example.consumeronlywamr/wasm');
  String _output = 'Select a WASM file to run...';
  bool _isLoading = false;

  final TextEditingController _funcNameController = TextEditingController(
    text: 'invoke', // Default for test_pdf.wasm
  );
  final TextEditingController _argsController = TextEditingController(
    text: '10, 20',
  );
  final TextEditingController _serverUrlController = TextEditingController(
    text: 'http://10.0.2.2:8080', // Default for Android Emulator to Go server
  );

  CloudSync? _cloudSync;

  @override
  void dispose() {
    _cloudSync?.disconnect();
    super.dispose();
  }

  void _toggleCloudSync() {
    if (_cloudSync != null && _cloudSync!.isConnected) {
      _cloudSync!.disconnect();
      setState(() {
        _cloudSync = null;
        _output += "\nCloud Sync Stopped.";
      });
    } else {
      _cloudSync = CloudSync(
        serverUrl: _serverUrlController.text,
        onLog: (msg) {
          if (mounted) {
             setState(() {
               _output += "\n[Cloud] $msg";
             });
          }
        }
      );
      _cloudSync!.connect();
      setState(() {
        _output = "Cloud Sync Started. Connecting...";
      });
    }
  }

  Future<String?> _logToServer(
    String funcName,
    dynamic input,
    dynamic output, {
    Uint8List? fileBytes,
    String? fileName,
  }) async {
    try {
      final baseUrl = _serverUrlController.text;
      final url = Uri.parse('$baseUrl/log');
      var request = http.MultipartRequest('POST', url);

      request.fields['function_name'] = funcName;
      request.fields['input_data'] = input.toString();
      request.fields['output_data'] = output.toString();

      if (fileBytes != null) {
        request.files.add(
          http.MultipartFile.fromBytes(
            'file',
            fileBytes,
            filename: fileName ?? 'output.bin',
          ),
        );
      }

      var response = await request.send();

      if (response.statusCode == 200) {
        final respBody = await response.stream.bytesToString();
        final data = json.decode(respBody);
        final filePath = data['file_path'];
        if (filePath != null) {
          return '$baseUrl$filePath';
        }
        return "Logged (No file)";
      } else {
        final respBody = await response.stream.bytesToString();
        debugPrint("Failed to log: ${response.statusCode} - $respBody");
      }
    } catch (e) {
      debugPrint("Error logging to server: $e");
    }
    return null;
  }

  Future<void> _invokeGenericWasm() async {
    setState(() {
      _isLoading = true;
      _output = "Preparing execution...";
    });

    try {
      FilePickerResult? pickResult = await FilePicker.platform.pickFiles();
      if (pickResult == null) {
        setState(() => _output = "No file selected.");
        return;
      }

      File file = File(pickResult.files.single.path!);
      Uint8List bytes = await file.readAsBytes();

      String funcName = _funcNameController.text.trim();
      List<int> args = _argsController.text
          .split(',')
          .where((e) => e.trim().isNotEmpty)
          .map((e) => int.tryParse(e.trim()) ?? 0)
          .toList();

      setState(() => _output = "Invoking $funcName(${args.join(', ')})...");

      final dynamic result = await platform.invokeMethod('invokeWasm', {
        'bytes': bytes,
        'funcName': funcName,
        'args': Int32List.fromList(args),
      });

      setState(() {
        _output = "Function: $funcName\nArgs: $args\n\nResult: $result";
      });

      // Log to server
      await _logToServer(funcName, args.toString(), result.toString());
    } on PlatformException catch (e) {
      setState(() => _output = "Error: ${e.message}");
    } catch (e) {
      setState(() => _output = "Other Error: $e");
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _runUniversalByteArrayWasm() async {
    setState(() {
      _isLoading = true;
      _output = "Preparing Universal Byte Array execution...";
    });

    try {
      final wasmPickResult = await FilePicker.platform.pickFiles(
        dialogTitle: 'Select WASM File',
      );
      if (wasmPickResult == null) {
        setState(() => _output = "No WASM file selected.");
        return;
      }
      final wasmFile = File(wasmPickResult.files.single.path!);
      final wasmBytes = await wasmFile.readAsBytes();

      final payloadPickResult = await FilePicker.platform.pickFiles(
        dialogTitle: 'Select Payload/Input File',
      );
      if (payloadPickResult == null) {
        setState(() => _output = "No payload file selected.");
        return;
      }
      final payloadFile = File(payloadPickResult.files.single.path!);
      final payloadBytes = await payloadFile.readAsBytes();

      String funcName = _funcNameController.text.trim();
      if (funcName.isEmpty) funcName = "invoke";

      setState(
        () => _output =
            "Invoking $funcName with ${payloadBytes.length} bytes payload...",
      );

      final dynamic result = await platform.invokeMethod('invokeDataWasm', {
        'bytes': wasmBytes,
        'funcName': funcName,
        'payload': payloadBytes,
      });

      if (result is Uint8List) {
        // Save PDF to temp dir and open it
        // Use internal cache dir to match the native WASI mapping
        final tempDir = await getTemporaryDirectory();
        final pdfFile = File('${tempDir.path}/wasm_output.pdf');
        await pdfFile.writeAsBytes(result);

        setState(() {
          _output =
              "Success! Received ${result.length} bytes.\nSaved to: ${pdfFile.path}\nOpening PDF...";
        });
        debugPrint("Received result: ${result.length} bytes");

        await OpenFilex.open(pdfFile.path);

        // Log to server
        final serverFileUrl = await _logToServer(
          funcName,
          "Universal Byte Array",
          "PDF saved to ${pdfFile.path}",
          fileBytes: result,
          fileName: 'wasm_output.pdf',
        );

        setState(() {
          _output += "\n\n🌐 View on Server: $serverFileUrl";
        });
      } else {
        setState(() {
          _output = "WASM returned unexpected type or result: $result";
        });
        await _logToServer(funcName, "Universal Byte Array", result.toString());
      }
    } on PlatformException catch (e) {
      setState(() => _output = "Error: ${e.message}");
    } catch (e) {
      setState(() => _output = "Other Error: $e");
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<void> _runMultiImageToPdfWasm() async {
    setState(() {
      _isLoading = true;
      _output = "Preparing Multi-Image to PDF execution...";
    });

    try {
      // 1. Select WASM
      final wasmPickResult = await FilePicker.platform.pickFiles(
        dialogTitle: 'Select multi_img_to_pdf.wasm',
      );
      if (wasmPickResult == null) {
        setState(() => _output = "No WASM file selected.");
        return;
      }
      final wasmFile = File(wasmPickResult.files.single.path!);
      final wasmBytes = await wasmFile.readAsBytes();

      // 2. Select Multiple Images
      final payloadPickResult = await FilePicker.platform.pickFiles(
        dialogTitle: 'Select Multiple JPEG Images',
        allowMultiple: true,
        type: FileType.image,
      );
      if (payloadPickResult == null || payloadPickResult.files.isEmpty) {
        setState(() => _output = "No images selected.");
        return;
      }

      // 3. Construct Payload: [Count (4 bytes HTTP)] [Size1 (4 bytes)] [Data1] ...
      int numImages = payloadPickResult.files.length;
      List<int> payloadList = [];

      // Add Count (Little Endian)
      payloadList.addAll([
        numImages & 0xFF,
        (numImages >> 8) & 0xFF,
        (numImages >> 16) & 0xFF,
        (numImages >> 24) & 0xFF,
      ]);

      for (var f in payloadPickResult.files) {
        final imgBytes = await File(f.path!).readAsBytes();
        int size = imgBytes.length;

        // Add Size (Little Endian)
        payloadList.addAll([
          size & 0xFF,
          (size >> 8) & 0xFF,
          (size >> 16) & 0xFF,
          (size >> 24) & 0xFF,
        ]);

        // Add Data
        payloadList.addAll(imgBytes);
      }

      Uint8List payloadBytes = Uint8List.fromList(payloadList);

      String funcName = _funcNameController.text.trim();
      if (funcName.isEmpty) funcName = "invoke";

      setState(
        () => _output =
            "Invoking $funcName with $numImages images (${payloadBytes.length} bytes total)...",
      );

      final dynamic result = await platform.invokeMethod('invokeDataWasm', {
        'bytes': wasmBytes,
        'funcName': funcName,
        'payload': payloadBytes,
      });

      if (result is Uint8List) {
        final tempDir = await getTemporaryDirectory();
        final pdfFile = File('${tempDir.path}/multi_image_output.pdf');
        await pdfFile.writeAsBytes(result);

        setState(() {
          _output =
              "Success! PDF generated (${result.length} bytes).\nSaved to: ${pdfFile.path}\nOpening PDF...";
        });

        await OpenFilex.open(pdfFile.path);

        final serverFileUrl = await _logToServer(
          funcName,
          "Multi-Image Payload ($numImages images)",
          "PDF saved to ${pdfFile.path}",
          fileBytes: result,
          fileName: 'multi_image_output.pdf',
        );

        setState(() {
          _output += "\n\n🌐 View on Server: $serverFileUrl";
        });
      } else {
        setState(() {
          _output = "WASM returned error or unexpected type: $result";
        });
      }
    } on PlatformException catch (e) {
      setState(() => _output = "Error: ${e.message}");
    } catch (e) {
      setState(() => _output = "Other Error: $e");
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: Text(widget.title),
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          children: <Widget>[
            TextField(
              controller: _funcNameController,
              decoration: const InputDecoration(
                labelText: 'WASM Function Name',
                hintText: 'e.g. add, multiply, grayscale',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _argsController,
              decoration: const InputDecoration(
                labelText: 'Arguments (comma separated ints)',
                hintText: 'e.g. 10, 20',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _serverUrlController,
              decoration: const InputDecoration(
                labelText: 'FastAPI Server URL',
                hintText: 'e.g. http://10.0.2.2:8000',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton.icon(
              onPressed: _toggleCloudSync,
              icon: Icon((_cloudSync != null && _cloudSync!.isConnected) ? Icons.cloud_off : Icons.cloud_sync),
              label: Text((_cloudSync != null && _cloudSync!.isConnected) ? 'Disconnect from Cloud' : 'Connect to Cloud (Wait for Tasks)'),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size.fromHeight(50),
                backgroundColor: (_cloudSync != null && _cloudSync!.isConnected) ? Colors.red.shade50 : Colors.blue.shade50,
              ),
            ),
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: _isLoading ? null : _invokeGenericWasm,
              icon: const Icon(Icons.rocket_launch),
              label: const Text('Pick File & Invoke'),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size.fromHeight(50),
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton.icon(
              onPressed: _isLoading ? null : _runUniversalByteArrayWasm,
              icon: const Icon(Icons.memory),
              label: const Text('Invoke Universal Byte Array WASM'),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size.fromHeight(50),
                backgroundColor: Colors.purple.shade50,
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton.icon(
              onPressed: _isLoading ? null : _runMultiImageToPdfWasm,
              icon: const Icon(Icons.picture_as_pdf),
              label: const Text('Invoke Multi-Image to PDF WASM'),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size.fromHeight(50),
                backgroundColor: Colors.orange.shade50,
              ),
            ),
            const SizedBox(height: 30),
            const Divider(),
            const Text(
              'Execution Log:',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 10),
            Expanded(
              child: Container(
                padding: const EdgeInsets.all(12),
                width: double.infinity,
                decoration: BoxDecoration(
                  color: Colors.black.withOpacity(0.05),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: SingleChildScrollView(
                  child: Text(
                    _output,
                    style: const TextStyle(fontFamily: 'monospace'),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
