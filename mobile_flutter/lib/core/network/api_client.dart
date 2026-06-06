import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/app_config.dart';
import 'api_exception.dart';

typedef AccessTokenProvider = String? Function();
typedef RefreshAccessTokenCallback = Future<void> Function();

class ApiClient {
  ApiClient({
    http.Client? client,
    this.accessTokenProvider,
    this.refreshAccessToken,
  }) : _client = client ?? http.Client();

  final http.Client _client;
  final AccessTokenProvider? accessTokenProvider;
  final RefreshAccessTokenCallback? refreshAccessToken;

  Future<http.Response> get(String path, {bool requiresAuth = false}) {
    return request('GET', path, requiresAuth: requiresAuth);
  }

  Future<http.Response> post(
    String path, {
    Map<String, dynamic>? body,
    bool requiresAuth = false,
  }) {
    return request('POST', path, body: body, requiresAuth: requiresAuth);
  }

  Future<http.Response> put(
    String path, {
    Map<String, dynamic>? body,
    bool requiresAuth = false,
  }) {
    return request('PUT', path, body: body, requiresAuth: requiresAuth);
  }

  Future<http.Response> delete(String path, {bool requiresAuth = false}) {
    return request('DELETE', path, requiresAuth: requiresAuth);
  }

  Future<http.Response> request(
    String method,
    String path, {
    Map<String, dynamic>? body,
    bool requiresAuth = false,
  }) async {
    Future<http.Response> send() {
      final headers = <String, String>{};
      if (body != null) {
        headers['Content-Type'] = 'application/json';
      }

      if (requiresAuth) {
        final token = accessTokenProvider?.call();
        if (token == null || token.isEmpty) {
          throw ApiException('Sessao expirada. Faça login novamente.');
        }
        headers['Authorization'] = 'Bearer $token';
      }

      final uri = _uri(path);
      switch (method.toUpperCase()) {
        case 'GET':
          return _client.get(uri, headers: headers);
        case 'POST':
          return _client.post(uri, headers: headers, body: body == null ? null : jsonEncode(body));
        case 'PUT':
          return _client.put(uri, headers: headers, body: body == null ? null : jsonEncode(body));
        case 'DELETE':
          return _client.delete(uri, headers: headers);
        default:
          throw ApiException('Metodo HTTP nao suportado: $method');
      }
    }

    var response = await send();
    if (requiresAuth && response.statusCode == 401 && refreshAccessToken != null) {
      await refreshAccessToken!.call();
      response = await send();
    }
    return response;
  }

  Map<String, dynamic> decodeJsonObject(String source) {
    final decoded = jsonDecode(source);
    if (decoded is! Map<String, dynamic>) {
      throw ApiException('Resposta da API em formato invalido.');
    }
    return decoded;
  }

  Uri _uri(String path) {
    final normalizedBase = AppConfig.apiBaseUrl.endsWith('/')
        ? AppConfig.apiBaseUrl.substring(0, AppConfig.apiBaseUrl.length - 1)
        : AppConfig.apiBaseUrl;
    return Uri.parse('$normalizedBase$path');
  }

  void close() {
    _client.close();
  }
}
