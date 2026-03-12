import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'node_controller.dart';

/// Production node status dashboard.
/// Purely observes NodeController — no logic here.
class NodeScreen extends StatelessWidget {
  const NodeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<NodeController>(
      builder: (context, ctrl, _) => Scaffold(
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
        ),
        body: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _StatusCard(ctrl: ctrl),
              const SizedBox(height: 16),
              _StatsRow(ctrl: ctrl),
              const SizedBox(height: 16),
              const Text(
                'Execution Log',
                style: TextStyle(
                  color: Colors.white70,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 1.2,
                ),
              ),
              const SizedBox(height: 8),
              Expanded(child: _LogView(log: ctrl.logBuffer)),
              const SizedBox(height: 16),
              _ConnectButton(ctrl: ctrl),
            ],
          ),
        ),
      ),
    );
  }
}

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
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: const Color(0xFF161622),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Row(
        children: [
          Icon(icon, color: color, size: 28),
          const SizedBox(width: 16),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                label,
                style: TextStyle(
                  color: color,
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                ),
              ),
              if (ctrl.sinkId != null)
                Text(
                  'ID: ${ctrl.sinkId}',
                  style: const TextStyle(color: Colors.white54, fontSize: 12),
                ),
              Text(
                ctrl.serverUrl,
                style: const TextStyle(color: Colors.white38, fontSize: 11),
              ),
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
            color: Colors.greenAccent,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: _StatChip(
            label: 'Failed',
            value: '${ctrl.executionFailed}',
            color: Colors.redAccent,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: _StatChip(
            label: 'Total',
            value: '${ctrl.executionSuccess + ctrl.executionFailed}',
            color: Colors.white70,
          ),
        ),
      ],
    );
  }
}

class _StatChip extends StatelessWidget {
  final String label, value;
  final Color color;
  const _StatChip({required this.label, required this.value, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 12),
      decoration: BoxDecoration(
        color: const Color(0xFF161622),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(
        children: [
          Text(value, style: TextStyle(color: color, fontSize: 22, fontWeight: FontWeight.bold)),
          const SizedBox(height: 4),
          Text(label, style: const TextStyle(color: Colors.white54, fontSize: 11)),
        ],
      ),
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
    // Auto-scroll to bottom when new log lines arrive
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
          backgroundColor: isOn ? Colors.red.shade900 : const Color(0xFF2563EB),
          foregroundColor: Colors.white,
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
        ),
      ),
    );
  }
}
