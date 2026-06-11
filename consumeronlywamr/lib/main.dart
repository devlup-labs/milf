import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'node_controller.dart';
import 'node_screen.dart';
import 'settings_screen.dart';

void main() {
  runApp(
    ChangeNotifierProvider(
      create: (_) => NodeController(),
      child: const MilfNodeApp(),
    ),
  );
}

class MilfNodeApp extends StatelessWidget {
  const MilfNodeApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MILF Node',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF2563EB),
          brightness: Brightness.dark,
        ),
        useMaterial3: true,
      ),
      routes: {
        '/': (_) => const NodeScreen(),
        '/settings': (_) => const SettingsScreen(),
      },
    );
  }
}
