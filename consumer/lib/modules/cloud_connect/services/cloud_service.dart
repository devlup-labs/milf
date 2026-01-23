import 'api_client.dart';

class CloudService {
  final ApiClient _apiClient;

  CloudService(this._apiClient);

  /// Checks if the server is reachable and the API key is valid.
  Future<bool> checkConnection() async {
    try {
      final response = await _apiClient.client.get('/health');
      return response.statusCode == 200;
    } catch (e) {
      print('Cloud Connection Failed: $e');
      return false;
    }
  }

  /// Reports node statistics (CPU/RAM/Active Jobs) to the server.
  Future<void> sendHeartbeat(Map<String, dynamic> stats) async {
    try {
      await _apiClient.client.post('/node/heartbeat', data: stats);
    } catch (e) {
      // Silent failure for heartbeats is usually acceptable
      print('Heartbeat failed: $e');
    }
  }
}
