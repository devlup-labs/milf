import 'dart:io';
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

  Future<void> _pickAndRunWasm() async {
    setState(() {
      _isLoading = true;
      _output = "Picking file...";
    });

    try {
      FilePickerResult? result = await FilePicker.platform.pickFiles();

      if (result != null) {
        File file = File(result.files.single.path!);
        Uint8List bytes = await file.readAsBytes();

        setState(() {
          _output = "Executing WASM (${bytes.length} bytes)...";
        });

        final String resultString = await platform.invokeMethod('runWasm', {
          'bytes': bytes,
        });

        setState(() {
          _output = "Execution Result:\n$resultString";
        });
      } else {
        setState(() {
          _output = "File picker canceled.";
        });
      }
    } on PlatformException catch (e) {
      setState(() {
        _output = "Failed to run WASM: '${e.message}'.";
      });
    } catch (e) {
      setState(() {
        _output = "Error: $e";
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
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
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            ElevatedButton.icon(
              onPressed: _isLoading ? null : _pickAndRunWasm,
              icon: const Icon(Icons.file_upload),
              label: const Text('Pick & Run .wasm File'),
              style: ElevatedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 20),
                textStyle: const TextStyle(fontSize: 18),
              ),
            ),
            const SizedBox(height: 30),
            Expanded(
              child: Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: Colors.grey[200],
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: Colors.grey),
                ),
                child: SingleChildScrollView(
                  child: Text(
                    _output,
                    style: const TextStyle(fontFamily: 'Courier', fontSize: 14),
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
