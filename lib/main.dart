import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:file_picker/file_picker.dart';

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
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
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
    text: 'add',
  );
  final TextEditingController _argsController = TextEditingController(
    text: '10, 20',
  );

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
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: _isLoading ? null : _invokeGenericWasm,
              icon: const Icon(Icons.rocket_launch),
              label: const Text('Pick File & Invoke'),
              style: ElevatedButton.styleFrom(
                minimumSize: const Size.fromHeight(50),
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
