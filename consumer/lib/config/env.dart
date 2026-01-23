import 'package:flutter_dotenv/flutter_dotenv.dart';

class Env {
  static String get serverUrl =>
      dotenv.env['SERVER_URL'] ?? 'http://localhost:3000';
  static String get nodeApiKey => dotenv.env['NODE_API_KEY'] ?? '';
  static String get nodeId => dotenv.env['NODE_ID'] ?? 'unknown_node';

  static Future<void> init() async {
    await dotenv.load(fileName: ".env");
  }
}
