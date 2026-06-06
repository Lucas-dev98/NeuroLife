import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/api_exception.dart';

class AuthUser {
  const AuthUser({
    required this.id,
    required this.email,
    required this.name,
  });

  final int id;
  final String email;
  final String name;
}

class PasswordResetRequestResult {
  const PasswordResetRequestResult({
    required this.message,
    this.resetToken,
    this.expiresAt,
  });

  final String message;
  final String? resetToken;
  final DateTime? expiresAt;
}

class AuthController extends ChangeNotifier {
  AuthController({http.Client? client})
      : _apiClient = ApiClient(client: client) {
    _authenticatedApiClient = ApiClient(
      client: client,
      accessTokenProvider: () => _accessToken,
      refreshAccessToken: _refreshTokens,
    );
  }

  static const _accessTokenKey = 'nl_access_token';
  static const _refreshTokenKey = 'nl_refresh_token';

  late final ApiClient _authenticatedApiClient;
  final ApiClient _apiClient;

  bool _isInitializing = true;
  AuthUser? _user;
  String? _accessToken;
  String? _refreshToken;

  bool get isInitializing => _isInitializing;
  bool get isAuthenticated => _accessToken != null && _refreshToken != null;
  AuthUser? get user => _user;
  ApiClient get apiClient => _authenticatedApiClient;

  Future<void> initialize() async {
    _isInitializing = true;
    notifyListeners();

    final prefs = await SharedPreferences.getInstance();
    _accessToken = prefs.getString(_accessTokenKey);
    _refreshToken = prefs.getString(_refreshTokenKey);

    if (_refreshToken == null || _refreshToken!.isEmpty) {
      await _clearSession(prefs: prefs);
      _finishInitialization();
      return;
    }

    try {
      if (_accessToken == null || _accessToken!.isEmpty) {
        await _refreshTokens();
      }
      await _loadProfile();
    } catch (_) {
      try {
        await _refreshTokens();
        await _loadProfile();
      } catch (_) {
        await _clearSession(prefs: prefs);
      }
    }

    _finishInitialization();
  }

  Future<void> login({required String email, required String password}) async {
    final body = {
      'email': email.trim(),
      'password': password,
    };

    final response = await _post('/api/v1/auth/login', body: body);
    final data = _decodeJson(response.body);
    await _consumeAuthPayload(data);
  }

  Future<void> register({
    required String name,
    required String email,
    required String password,
  }) async {
    final body = {
      'name': name.trim(),
      'email': email.trim(),
      'password': password,
    };

    final response = await _post('/api/v1/auth/register', body: body, expectedStatus: 201);
    final data = _decodeJson(response.body);
    await _consumeAuthPayload(data);
  }

  Future<PasswordResetRequestResult> requestPasswordReset(String email) async {
    final response = await _post(
      '/api/v1/auth/forgot-password',
      body: {'email': email.trim()},
      expectedStatus: 200,
      throwOnUnexpectedStatus: false,
    );

    if (response.statusCode >= 200 && response.statusCode < 300) {
      final data = _decodeJson(response.body);
      return PasswordResetRequestResult(
        message: data['message']?.toString() ?? 'Solicitacao processada.',
        resetToken: data['reset_token']?.toString(),
        expiresAt: _parseDateTime(data['expires_at']?.toString()),
      );
    }

    if (response.statusCode == 404 || response.statusCode == 405) {
      throw ApiException('Fluxo de recuperacao ainda nao foi implementado no backend.', statusCode: response.statusCode);
    }

    _throwFromResponse(response);
    throw ApiException('Falha ao solicitar redefinicao de senha.', statusCode: response.statusCode);
  }

  Future<void> resetPassword({required String token, required String password}) async {
    final response = await _post(
      '/api/v1/auth/reset-password',
      body: {
        'token': token.trim(),
        'password': password,
      },
    );
    final data = _decodeJson(response.body);
    await _consumeAuthPayload(data);
  }

  Future<void> logout() async {
    await _clearSession();
    notifyListeners();
  }

  Future<void> _consumeAuthPayload(Map<String, dynamic> payload) async {
    final userMap = payload['user'];
    final tokensMap = payload['tokens'];

    if (userMap is! Map || tokensMap is! Map) {
      throw ApiException('Resposta de autenticacao invalida.');
    }

    final accessToken = tokensMap['access_token']?.toString() ?? '';
    final refreshToken = tokensMap['refresh_token']?.toString() ?? '';

    if (accessToken.isEmpty || refreshToken.isEmpty) {
      throw ApiException('Tokens ausentes na resposta de autenticacao.');
    }

    _accessToken = accessToken;
    _refreshToken = refreshToken;
    _user = AuthUser(
      id: (userMap['id'] as num?)?.toInt() ?? 0,
      email: userMap['email']?.toString() ?? '',
      name: userMap['name']?.toString() ?? '',
    );

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_accessTokenKey, _accessToken!);
    await prefs.setString(_refreshTokenKey, _refreshToken!);
    notifyListeners();
  }

  Future<void> _loadProfile() async {
    final response = await _authorizedGet('/api/v1/profile');
    final data = _decodeJson(response.body);

    _user = AuthUser(
      id: (data['id'] as num?)?.toInt() ?? 0,
      email: data['email']?.toString() ?? '',
      name: data['name']?.toString() ?? '',
    );
    notifyListeners();
  }

  Future<http.Response> _authorizedGet(String path) async {
	    final response = await _authenticatedApiClient.get(path, requiresAuth: true);

	    if (response.statusCode < 200 || response.statusCode >= 300) {
	      _throwFromResponse(response);
	    }

	    return response;
  }

  Future<void> _refreshTokens() async {
    final currentRefresh = _refreshToken;
    if (currentRefresh == null || currentRefresh.isEmpty) {
      throw ApiException('Sessao expirada. Faça login novamente.');
    }

    final response = await _post('/api/v1/auth/refresh', body: {'refresh_token': currentRefresh});
    final data = _decodeJson(response.body);
    final tokensMap = data['tokens'];

    if (tokensMap is! Map) {
      throw ApiException('Resposta de refresh invalida.');
    }

    final newAccessToken = tokensMap['access_token']?.toString() ?? '';
    final newRefreshToken = tokensMap['refresh_token']?.toString() ?? '';
    if (newAccessToken.isEmpty || newRefreshToken.isEmpty) {
      throw ApiException('Tokens ausentes na resposta de refresh.');
    }

    _accessToken = newAccessToken;
    _refreshToken = newRefreshToken;

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_accessTokenKey, _accessToken!);
    await prefs.setString(_refreshTokenKey, _refreshToken!);
    notifyListeners();
  }

  Future<http.Response> _post(
    String path, {
    required Map<String, dynamic> body,
    int expectedStatus = 200,
    bool throwOnUnexpectedStatus = true,
  }) async {
    final response = await _apiClient.post(path, body: body);

    final isExpected = response.statusCode == expectedStatus ||
        (expectedStatus == 200 && response.statusCode >= 200 && response.statusCode < 300);

    if (!isExpected && throwOnUnexpectedStatus) {
      _throwFromResponse(response);
    }

    return response;
  }

  Map<String, dynamic> _decodeJson(String source) {
    return _apiClient.decodeJsonObject(source);
  }

  void _throwFromResponse(http.Response response) {
    String? message;
    try {
      final data = _decodeJson(response.body);
      message = data['error']?.toString();
    } catch (_) {
      // Falls back to generic message below when response is not parseable.
    }

    if (message != null && message.isNotEmpty) {
      throw ApiException(message, statusCode: response.statusCode);
    }

    throw ApiException(
      'Falha na comunicacao com a API (${response.statusCode}).',
      statusCode: response.statusCode,
    );
  }

  DateTime? _parseDateTime(String? rawValue) {
    if (rawValue == null || rawValue.isEmpty) return null;
    return DateTime.tryParse(rawValue);
  }

  Future<void> _clearSession({SharedPreferences? prefs}) async {
    final p = prefs ?? await SharedPreferences.getInstance();
    _accessToken = null;
    _refreshToken = null;
    _user = null;
    await p.remove(_accessTokenKey);
    await p.remove(_refreshTokenKey);
  }

  void _finishInitialization() {
    _isInitializing = false;
    notifyListeners();
  }

  @override
  void dispose() {
    _apiClient.close();
    super.dispose();
  }
}
