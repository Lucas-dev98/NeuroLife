import 'package:flutter/foundation.dart';

abstract final class AppConfig {
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: kIsWeb ? 'http://localhost:18080' : 'http://10.0.2.2:18080',
  );
}
