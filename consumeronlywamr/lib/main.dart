import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:flutter_background/flutter_background.dart';
import 'node_controller.dart';
import 'node_screen.dart';
import 'settings_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  const androidConfig = FlutterBackgroundAndroidConfig(
    notificationTitle: "MILF Node Active",
    notificationText: "Background execution is active for running WebAssembly lambdas.",
    notificationImportance: AndroidNotificationImportance.normal,
    notificationIcon: AndroidResource(name: 'ic_launcher', defType: 'mipmap'),
    enableWifiLock: true,
  );

  await FlutterBackground.initialize(androidConfig: androidConfig);

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
