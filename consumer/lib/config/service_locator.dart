import 'package:get_it/get_it.dart';

import '../modules/cloud_connect/services/api_client.dart';
import '../modules/cloud_connect/services/cloud_service.dart';

final getIt = GetIt.instance;

Future<void> setupServiceLocator() async {
  // Cloud Connect
  getIt.registerLazySingleton<ApiClient>(() => ApiClient());
  getIt.registerLazySingleton<CloudService>(
    () => CloudService(getIt<ApiClient>()),
  );
}
