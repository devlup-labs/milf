import 'package:dio/dio.dart';
import '../../../config/env.dart';

class ApiClient {
  late final Dio _dio;

  ApiClient() {
    _dio = Dio(
      BaseOptions(
        baseUrl: Env.serverUrl,
        connectTimeout: const Duration(seconds: 10),
        receiveTimeout: const Duration(seconds: 30),
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
      ),
    );

    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          // Attach API Key to every request
          options.headers['x-api-key'] = Env.nodeApiKey;
          options.headers['x-node-id'] = Env.nodeId;
          return handler.next(options);
        },
        onError: (DioException e, handler) {
          // Global error handling logging
          print('[API Error] ${e.requestOptions.path}: ${e.message}');
          return handler.next(e);
        },
      ),
    );
  }

  Dio get client => _dio;
}
