import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'node_controller.dart';

/// Operator settings screen.
/// Allows configuring the server URL and auth token without changing code.
class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  late TextEditingController _urlCtrl;
  late TextEditingController _tokenCtrl;

  @override
  void initState() {
    super.initState();
    final ctrl = context.read<NodeController>();
    _urlCtrl = TextEditingController(text: ctrl.serverUrl);
    _tokenCtrl = TextEditingController(text: ctrl.authToken);
  }

  @override
  void dispose() {
    _urlCtrl.dispose();
    _tokenCtrl.dispose();
    super.dispose();
  }

  void _save() {
    final ctrl = context.read<NodeController>();
    ctrl.configure(url: _urlCtrl.text.trim(), token: _tokenCtrl.text.trim());
    Navigator.pop(context);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Settings saved')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0A0A0F),
      appBar: AppBar(
        backgroundColor: const Color(0xFF0A0A0F),
        title: const Text('Settings', style: TextStyle(color: Colors.white)),
        iconTheme: const IconThemeData(color: Colors.white),
      ),
      body: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Server Configuration',
              style: TextStyle(color: Colors.white54, fontSize: 12, letterSpacing: 1.2),
            ),
            const SizedBox(height: 16),
            _DarkField(
              controller: _urlCtrl,
              label: 'Server URL',
              hint: 'http://10.0.2.2:8080',
              icon: Icons.dns_outlined,
            ),
            const SizedBox(height: 16),
            _DarkField(
              controller: _tokenCtrl,
              label: 'Auth Token (JWT)',
              hint: 'Obtained after Google login',
              icon: Icons.key_outlined,
              obscure: true,
            ),
            const SizedBox(height: 8),
            const Text(
              'Sign in via the web dashboard to get your token, then paste it here.',
              style: TextStyle(color: Colors.white38, fontSize: 11),
            ),
            const Spacer(),
            SizedBox(
              width: double.infinity,
              height: 52,
              child: ElevatedButton(
                onPressed: _save,
                style: ElevatedButton.styleFrom(
                  backgroundColor: const Color(0xFF2563EB),
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                ),
                child: const Text('Save Settings'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DarkField extends StatelessWidget {
  final TextEditingController controller;
  final String label, hint;
  final IconData icon;
  final bool obscure;
  const _DarkField({
    required this.controller,
    required this.label,
    required this.hint,
    required this.icon,
    this.obscure = false,
  });

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      obscureText: obscure,
      style: const TextStyle(color: Colors.white),
      decoration: InputDecoration(
        labelText: label,
        hintText: hint,
        labelStyle: const TextStyle(color: Colors.white54),
        hintStyle: const TextStyle(color: Colors.white30),
        prefixIcon: Icon(icon, color: Colors.white38),
        filled: true,
        fillColor: const Color(0xFF161622),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: Colors.white10),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: Colors.white10),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: Color(0xFF2563EB)),
        ),
      ),
    );
  }
}
