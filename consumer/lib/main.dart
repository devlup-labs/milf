import 'dart:async';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
import 'config/env.dart';
import 'config/service_locator.dart';
import 'modules/native_bridge/native_bridge_module.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  bool envLoaded = false;
  try {
    await Env.init();
    envLoaded = true;
  } catch (e) {
    debugPrint("Warning: Failed to load .env file: $e");
  }

  await setupServiceLocator();

  runApp(ConsumerApp(envLoaded: envLoaded));
}

class ConsumerApp extends StatelessWidget {
  final bool envLoaded;
  const ConsumerApp({super.key, required this.envLoaded});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Consumer Node Pro',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.cyan,
          brightness: Brightness.dark,
        ),
        useMaterial3: true,
      ),
      home: ConsumerHomePage(envLoaded: envLoaded),
    );
  }
}

class WasmJob {
  final String fileName;
  String status;
  String? output;
  String? path;
  int? memoryUsed; // Simulated per-function memory

  WasmJob({required this.fileName, this.status = 'Pending'});
}

class ConsumerHomePage extends StatefulWidget {
  final bool envLoaded;
  const ConsumerHomePage({super.key, required this.envLoaded});

  @override
  State<ConsumerHomePage> createState() => _ConsumerHomePageState();
}

class _ConsumerHomePageState extends State<ConsumerHomePage> {
  final List<WasmJob> _jobs = [];
  bool _isProcessing = false;
  final NativeBridge _bridge = NativeBridge();
  int _activeJobsCount = 0;
  static const int maxConcurrentJobs = 3;

  // Stats
  double _currentRam = 0.0;
  double _deviceAvail = 0.0;
  double _deviceTotal = 0.0;
  bool _lowMemory = false;
  int _perJobMemoryLimit = 50; // Default limit 50MB
  Timer? _statsTimer;

  @override
  void initState() {
    super.initState();
    _startStatsPolling();
  }

  @override
  void dispose() {
    _statsTimer?.cancel();
    super.dispose();
  }

  void _startStatsPolling() {
    _statsTimer = Timer.periodic(const Duration(seconds: 1), (timer) async {
      final stats = await _bridge.getOSResources();
      if (stats != null) {
        setState(() {
          _currentRam =
              (stats['mem'] as int).toDouble() / 1024.0; // Convert to MB
          _deviceAvail =
              (stats['deviceAvail'] as int? ?? 0).toDouble() / 1024.0;
          _deviceTotal =
              (stats['deviceTotal'] as int? ?? 0).toDouble() / 1024.0;
          _lowMemory = stats['lowMemory'] as bool? ?? false;
        });
      }
    });
  }

  Future<void> _pickAndProcessFiles() async {
    final result = await FilePicker.platform.pickFiles(
      allowMultiple: true,
      type: FileType.custom,
      allowedExtensions: ['c'],
    );

    if (result != null) {
      setState(() {
        _jobs.addAll(result.files.map((f) => WasmJob(fileName: f.name)));
      });
      _processJobs();
    }
  }

  Future<void> _processJobs() async {
    if (_isProcessing) return;
    setState(() => _isProcessing = true);

    const double ramLimit = 500.0; // 500 MB Limit

    // We use a while loop to act as a scheduler
    while (_jobs.any((j) => j.status == 'Pending')) {
      // Check RAM pressure and active jobs before spawning more
      if (_currentRam < ramLimit && _activeJobsCount < maxConcurrentJobs) {
        final job = _jobs.firstWhere((j) => j.status == 'Pending');

        // Launch job but DON'T await it here so we can spawn the next one simultaneously
        unawaited(_runSingleJob(job));

        // Small delay
        await Future.delayed(const Duration(milliseconds: 100));
      } else {
        // Wait for resources to be released before checking again
        await Future.delayed(const Duration(milliseconds: 200));
        // Force update RAM stats immediately to be more reactive
        final stats = await _bridge.getOSResources();
        if (stats != null) {
          setState(() {
            _currentRam = (stats['mem'] as int).toDouble() / 1024.0;
            _deviceAvail =
                (stats['deviceAvail'] as int? ?? 0).toDouble() / 1024.0;
            _deviceTotal =
                (stats['deviceTotal'] as int? ?? 0).toDouble() / 1024.0;
            _lowMemory = stats['lowMemory'] as bool? ?? false;
          });
        }
      }
    }

    // Wait until all started jobs are actually finished
    while (_jobs.any(
      (j) => j.status == 'Compiling...' || j.status == 'Running...',
    )) {
      await Future.delayed(const Duration(milliseconds: 500));
    }

    setState(() => _isProcessing = false);
  }

  Future<void> _runSingleJob(WasmJob job) async {
    setState(() {
      job.status = 'Compiling...';
      _activeJobsCount++;
    });

    // Simulating C -> WASM Compile
    await Future.delayed(const Duration(seconds: 2));

    if (!mounted) {
      _activeJobsCount--;
      return;
    }
    setState(() => job.status = 'Running...');

    final dummyWasm = Uint8List.fromList([
      0x00,
      0x61,
      0x73,
      0x6d,
      0x01,
      0x00,
      0x00,
      0x00,
    ]);
    final result = await _bridge.runWasmTest(dummyWasm, {
      "file": job.fileName,
      "mode": "concurrent_execution",
    }, _perJobMemoryLimit);

    if (!mounted) {
      _activeJobsCount--;
      return;
    }
    setState(() {
      job.status = (result?['content']?.startsWith('Error') ?? true)
          ? 'Failed'
          : 'Completed';
      job.output = result?['content'];
      job.path = result?['path'];
      job.memoryUsed = result?['memDelta'] as int?;
      _activeJobsCount--;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('MILF Consumer Stats'),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 16.0),
            child: Center(
              child: Text(
                'RAM: ${_currentRam.toStringAsFixed(1)} MB',
                style: const TextStyle(
                  fontWeight: FontWeight.bold,
                  color: Colors.cyanAccent,
                ),
              ),
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          _buildSystemStatus(),
          const Divider(height: 1),
          Expanded(child: _buildJobsList()),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: _isProcessing ? null : _pickAndProcessFiles,
        icon: const Icon(Icons.add),
        label: const Text('Upload C Files'),
        backgroundColor: Colors.cyan,
      ),
    );
  }

  Widget _buildSystemStatus() {
    return Container(
      padding: const EdgeInsets.all(16),
      color: Colors.black26,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Active Node Telemetry',
                style: TextStyle(
                  fontSize: 12,
                  color: Colors.cyanAccent.withOpacity(0.7),
                ),
              ),
              Text(
                'UPTIME: ${(DateTime.now().second % 60)}s',
                style: const TextStyle(fontSize: 10, color: Colors.grey),
              ),
            ],
          ),
          const SizedBox(height: 8),
          LinearProgressIndicator(
            value: (_currentRam / 500).clamp(0, 1), // 500MB total scale
            color: _currentRam > 450 ? Colors.red : Colors.greenAccent,
            backgroundColor: Colors.white10,
          ),
          const SizedBox(height: 4),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Live App RAM Usage',
                style: TextStyle(fontSize: 12, color: Colors.grey[400]),
              ),
              Text(
                '${_currentRam.toStringAsFixed(2)} MB',
                style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'PER-JOB RAM LIMIT: ${_perJobMemoryLimit} MB',
                      style: const TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                        color: Colors.orangeAccent,
                      ),
                    ),
                    Slider(
                      value: _perJobMemoryLimit.toDouble(),
                      min: 10.0,
                      max: 200.0,
                      divisions: 19,
                      label: '$_perJobMemoryLimit MB',
                      onChanged: _isProcessing
                          ? null
                          : (v) =>
                                setState(() => _perJobMemoryLimit = v.toInt()),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 16),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    'DEVICE TOTAL: ${_deviceTotal.toStringAsFixed(0)} MB',
                    style: const TextStyle(fontSize: 9, color: Colors.grey),
                  ),
                  Text(
                    'DEVICE AVAIL: ${_deviceAvail.toStringAsFixed(0)} MB',
                    style: TextStyle(
                      fontSize: 10,
                      color: _lowMemory ? Colors.red : Colors.greenAccent,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildJobsList() {
    if (_jobs.isEmpty) {
      return const Center(
        child: Text(
          'No active jobs. Upload C files to start.',
          style: TextStyle(color: Colors.grey),
        ),
      );
    }

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(
            children: [
              Text(
                'ACTIVE JOBS: ${_jobs.length}',
                style: const TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.bold,
                  color: Colors.grey,
                ),
              ),
              const Spacer(),
              Text(
                'COMPLETED: ${_jobs.where((j) => j.status == 'Completed').length}',
                style: const TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.bold,
                  color: Colors.cyan,
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: ListView.builder(
            itemCount: _jobs.length,
            itemBuilder: (context, index) {
              final job = _jobs[index];
              final lastLine =
                  job.output
                      ?.split('\n')
                      .where((s) => s.contains('Result:'))
                      .firstOrNull
                      ?.trim() ??
                  '';

              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                child: ExpansionTile(
                  leading: _getStatusIcon(job.status),
                  title: Text(
                    job.fileName,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Status: ${job.status}',
                        style: TextStyle(
                          color: _getStatusColor(job.status),
                          fontSize: 11,
                        ),
                      ),
                      if (lastLine.isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.only(top: 2.0),
                          child: Text(
                            'Quick Result: $lastLine',
                            style: const TextStyle(
                              fontSize: 10,
                              color: Colors.greenAccent,
                              fontFamily: 'monospace',
                            ),
                          ),
                        ),
                    ],
                  ),
                  children: [
                    if (job.output != null)
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.all(12),
                        margin: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: Colors.black,
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              'FUNCTION OUTPUT',
                              style: TextStyle(
                                fontSize: 10,
                                color: Colors.cyan,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Text(
                              job.output!,
                              style: const TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 11,
                                color: Colors.greenAccent,
                              ),
                            ),
                            const SizedBox(height: 8),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              children: [
                                const Text(
                                  'MEMORY CONSUMPTION',
                                  style: TextStyle(
                                    fontSize: 10,
                                    color: Colors.cyan,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                                Text(
                                  '${job.memoryUsed ?? 0} KB',
                                  style: const TextStyle(
                                    fontSize: 10,
                                    color: Colors.greenAccent,
                                    fontFamily: 'monospace',
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 8),
                            const Text(
                              'STORED AT',
                              style: TextStyle(
                                fontSize: 10,
                                color: Colors.grey,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            Text(
                              job.path!,
                              style: const TextStyle(
                                fontSize: 9,
                                color: Colors.grey,
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _getStatusIcon(String status) {
    switch (status) {
      case 'Pending':
        return const Icon(Icons.hourglass_empty, color: Colors.grey);
      case 'Compiling...':
        return const SizedBox(
          width: 24,
          height: 24,
          child: CircularProgressIndicator(strokeWidth: 2),
        );
      case 'Running...':
        return const Icon(Icons.play_arrow, color: Colors.green);
      case 'Completed':
        return const Icon(Icons.check_circle, color: Colors.cyanAccent);
      default:
        return const Icon(Icons.error, color: Colors.red);
    }
  }

  Color _getStatusColor(String status) {
    if (status.contains('Compiling')) return Colors.orange;
    if (status.contains('Running')) return Colors.green;
    if (status == 'Completed') return Colors.cyanAccent;
    if (status == 'Failed') return Colors.redAccent;
    return Colors.grey;
  }
}
