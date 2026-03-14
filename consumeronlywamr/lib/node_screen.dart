import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'node_controller.dart';

/// Production node status dashboard with two tabs:
///   • Log    — live execution log
///   • WASM   — incoming WASM / payload inspector
class NodeScreen extends StatelessWidget {
  const NodeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NodeController>(
      builder: (context, ctrl, _) => DefaultTabController(
        length: 2,
        child: Scaffold(
          backgroundColor: const Color(0xFF0A0A0F),
          appBar: AppBar(
            backgroundColor: const Color(0xFF0A0A0F),
            title: const Text(
              'MILF Node',
              style: TextStyle(color: Colors.white, fontWeight: FontWeight.w600),
            ),
            actions: [
              IconButton(
                icon: const Icon(Icons.settings, color: Colors.white70),
                onPressed: () => Navigator.pushNamed(context, '/settings'),
              ),
            ],
            bottom: const TabBar(
              indicatorColor: Color(0xFF2563EB),
              labelColor: Colors.white,
              unselectedLabelColor: Colors.white38,
              tabs: [
                Tab(icon: Icon(Icons.terminal, size: 18), text: 'Log'),
                Tab(icon: Icon(Icons.memory, size: 18), text: 'WASM Info'),
              ],
            ),
          ),
          body: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
                child: Column(
                  children: [
                    _StatusCard(ctrl: ctrl),
                    const SizedBox(height: 12),
                    _StatsRow(ctrl: ctrl),
                    const SizedBox(height: 12),
                  ],
                ),
              ),
              Expanded(
                child: TabBarView(
                  children: [
                    _LogTab(ctrl: ctrl),
                    _WasmInfoTab(ctrl: ctrl),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(16),
                child: _ConnectButton(ctrl: ctrl),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Status & Stats ────────────────────────────────────────────────────────────

class _StatusCard extends StatelessWidget {
  final NodeController ctrl;
  const _StatusCard({required this.ctrl});

  @override
  Widget build(BuildContext context) {
    final (color, label, icon) = switch (ctrl.status) {
      NodeStatus.idle => (Colors.grey, 'Idle', Icons.circle_outlined),
      NodeStatus.connecting => (Colors.amber, 'Connecting...', Icons.sync),
      NodeStatus.online => (Colors.greenAccent, 'Online', Icons.circle),
      NodeStatus.executing => (Colors.blue, 'Executing', Icons.memory),
      NodeStatus.error => (Colors.redAccent, 'Error', Icons.error_outline),
    };

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFF161622),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Row(
        children: [
          Icon(icon, color: color, size: 26),
          const SizedBox(width: 14),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label,
                  style: TextStyle(
                      color: color, fontSize: 16, fontWeight: FontWeight.bold)),
              if (ctrl.sinkId != null)
                Text('ID: ${ctrl.sinkId}',
                    style:
                        const TextStyle(color: Colors.white54, fontSize: 11)),
              Text(ctrl.serverUrl,
                  style:
                      const TextStyle(color: Colors.white38, fontSize: 10)),
            ],
          ),
        ],
      ),
    );
  }
}

class _StatsRow extends StatelessWidget {
  final NodeController ctrl;
  const _StatsRow({required this.ctrl});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
            child: _StatChip(
                label: 'Success',
                value: '${ctrl.executionSuccess}',
                color: Colors.greenAccent)),
        const SizedBox(width: 10),
        Expanded(
            child: _StatChip(
                label: 'Failed',
                value: '${ctrl.executionFailed}',
                color: Colors.redAccent)),
        const SizedBox(width: 10),
        Expanded(
            child: _StatChip(
                label: 'Total',
                value: '${ctrl.executionSuccess + ctrl.executionFailed}',
                color: Colors.white70)),
      ],
    );
  }
}

class _StatChip extends StatelessWidget {
  final String label, value;
  final Color color;
  const _StatChip(
      {required this.label, required this.value, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFF161622),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(
        children: [
          Text(value,
              style: TextStyle(
                  color: color,
                  fontSize: 20,
                  fontWeight: FontWeight.bold)),
          const SizedBox(height: 2),
          Text(label,
              style:
                  const TextStyle(color: Colors.white54, fontSize: 10)),
        ],
      ),
    );
  }
}

// ── Tab 1 — Log ───────────────────────────────────────────────────────────────

class _LogTab extends StatelessWidget {
  final NodeController ctrl;
  const _LogTab({required this.ctrl});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: _LogView(log: ctrl.logBuffer),
    );
  }
}

class _LogView extends StatefulWidget {
  final String log;
  const _LogView({required this.log});

  @override
  State<_LogView> createState() => _LogViewState();
}

class _LogViewState extends State<_LogView> {
  final ScrollController _scroll = ScrollController();

  @override
  void didUpdateWidget(_LogView old) {
    super.didUpdateWidget(old);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xFF0D0D14),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: Colors.white10),
      ),
      child: SingleChildScrollView(
        controller: _scroll,
        child: Text(
          widget.log,
          style: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 11,
            color: Colors.white60,
            height: 1.6,
          ),
        ),
      ),
    );
  }
}

// ── Tab 2 — WASM Info ─────────────────────────────────────────────────────────

class _WasmInfoTab extends StatelessWidget {
  final NodeController ctrl;
  const _WasmInfoTab({required this.ctrl});

  @override
  Widget build(BuildContext context) {
    final events = ctrl.wasmEvents;

    if (events.isEmpty) {
      return const Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.inbox_outlined, color: Colors.white24, size: 48),
            SizedBox(height: 12),
            Text('No WASM tasks received yet',
                style: TextStyle(color: Colors.white38, fontSize: 13)),
          ],
        ),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      itemCount: events.length,
      itemBuilder: (context, i) => _WasmEventCard(event: events[i]),
    );
  }
}

class _WasmEventCard extends StatelessWidget {
  final WasmEvent event;
  const _WasmEventCard({required this.event});

  String _fmt(DateTime dt) =>
      '${dt.hour.toString().padLeft(2, '0')}:'
      '${dt.minute.toString().padLeft(2, '0')}:'
      '${dt.second.toString().padLeft(2, '0')}';

  String _size(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(2)} MB';
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(top: 10),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF161622),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFF2563EB).withValues(alpha: 0.25)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.memory, color: Color(0xFF2563EB), size: 16),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  event.lambdaId,
                  style: const TextStyle(
                      color: Colors.white, fontSize: 12, fontFamily: 'monospace'),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Text(_fmt(event.timestamp),
                  style:
                      const TextStyle(color: Colors.white38, fontSize: 10)),
            ],
          ),
          const SizedBox(height: 10),
          Wrap(
            spacing: 8,
            runSpacing: 6,
            children: [
              _Badge(
                label: 'WASM',
                value: _size(event.wasmSizeBytes),
                icon: Icons.insert_drive_file_outlined,
                color: Colors.blueAccent,
              ),
              _Badge(
                label: 'Payload',
                value: _size(event.payloadSizeBytes),
                icon: Icons.data_object,
                color: Colors.purpleAccent,
              ),
              _Badge(
                label: 'Type',
                value: event.payloadType,
                icon: Icons.label_outline,
                color: event.payloadType == 'null'
                    ? Colors.white38
                    : Colors.greenAccent,
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            'exec: ${event.executionId}',
            style: const TextStyle(
                color: Colors.white30, fontSize: 9, fontFamily: 'monospace'),
          ),
          if (event.success != null) ...[
            const SizedBox(height: 12),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: event.success!
                    ? Colors.greenAccent.withValues(alpha: 0.05)
                    : Colors.redAccent.withValues(alpha: 0.05),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: event.success!
                      ? Colors.greenAccent.withValues(alpha: 0.3)
                      : Colors.redAccent.withValues(alpha: 0.3),
                ),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    event.success! ? 'OUTPUT' : 'ERROR',
                    style: TextStyle(
                      color: event.success! ? Colors.greenAccent : Colors.redAccent,
                      fontSize: 10,
                      fontWeight: FontWeight.bold,
                      letterSpacing: 1.2,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    event.success! ? '${event.output}' : '${event.errorMessage}',
                    style: TextStyle(
                      color: event.success! ? Colors.white70 : Colors.redAccent.shade100,
                      fontSize: 12,
                      fontFamily: 'monospace',
                    ),
                  ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _Badge extends StatelessWidget {
  final String label, value;
  final IconData icon;
  final Color color;
  const _Badge(
      {required this.label,
      required this.value,
      required this.icon,
      required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: color, size: 12),
          const SizedBox(width: 5),
          Text(label,
              style: TextStyle(
                  color: color.withValues(alpha: 0.7),
                  fontSize: 10,
                  fontWeight: FontWeight.w600)),
          const SizedBox(width: 4),
          Text(value,
              style: TextStyle(color: color, fontSize: 11, fontWeight: FontWeight.bold)),
        ],
      ),
    );
  }
}

// ── Connect Button ────────────────────────────────────────────────────────────

class _ConnectButton extends StatelessWidget {
  final NodeController ctrl;
  const _ConnectButton({required this.ctrl});

  @override
  Widget build(BuildContext context) {
    final isOn = ctrl.isConnected;
    return SizedBox(
      width: double.infinity,
      height: 52,
      child: ElevatedButton.icon(
        onPressed: ctrl.status == NodeStatus.connecting
            ? null
            : (isOn ? ctrl.disconnect : ctrl.connect),
        icon: Icon(isOn ? Icons.cloud_off : Icons.cloud_sync),
        label: Text(isOn ? 'Disconnect' : 'Connect to Server'),
        style: ElevatedButton.styleFrom(
          backgroundColor:
              isOn ? Colors.red.shade900 : const Color(0xFF2563EB),
          foregroundColor: Colors.white,
          shape:
              RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
      ),
    );
  }
}
