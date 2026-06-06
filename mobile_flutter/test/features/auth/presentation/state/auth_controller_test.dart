import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:mobile_flutter/features/auth/presentation/state/auth_controller.dart';

void main() {
  group('AuthController', () {
    test('login stores tokens and user', () async {
      SharedPreferences.setMockInitialValues({});

      final controller = AuthController(
        client: MockClient((request) async {
          expect(request.url.path, '/api/v1/auth/login');
          return http.Response(
            jsonEncode({
              'user': {'id': 7, 'email': 'ana@example.com', 'name': 'Ana'},
              'tokens': {'access_token': 'access-1', 'refresh_token': 'refresh-1'},
            }),
            200,
            headers: {'content-type': 'application/json'},
          );
        }),
      );

      await controller.login(email: 'ana@example.com', password: '12345678');

      final prefs = await SharedPreferences.getInstance();
      expect(controller.isAuthenticated, isTrue);
      expect(controller.user?.name, 'Ana');
      expect(prefs.getString('nl_access_token'), 'access-1');
      expect(prefs.getString('nl_refresh_token'), 'refresh-1');

      controller.dispose();
    });

    test('initialize refreshes tokens and loads profile', () async {
      SharedPreferences.setMockInitialValues({
        'nl_refresh_token': 'refresh-old',
      });

      final controller = AuthController(
        client: MockClient((request) async {
          if (request.url.path == '/api/v1/auth/refresh') {
            return http.Response(
              jsonEncode({
                'tokens': {'access_token': 'access-2', 'refresh_token': 'refresh-2'},
              }),
              200,
              headers: {'content-type': 'application/json'},
            );
          }

          if (request.url.path == '/api/v1/profile') {
            expect(request.headers['authorization'], 'Bearer access-2');
            return http.Response(
              jsonEncode({'id': 9, 'email': 'bia@example.com', 'name': 'Bia'}),
              200,
              headers: {'content-type': 'application/json'},
            );
          }

          return http.Response('not found', 404);
        }),
      );

      await controller.initialize();

      final prefs = await SharedPreferences.getInstance();
      expect(controller.isAuthenticated, isTrue);
      expect(controller.user?.email, 'bia@example.com');
      expect(prefs.getString('nl_access_token'), 'access-2');
      expect(prefs.getString('nl_refresh_token'), 'refresh-2');

      controller.dispose();
    });

    test('logout clears persisted session', () async {
      SharedPreferences.setMockInitialValues({
        'nl_access_token': 'access-old',
        'nl_refresh_token': 'refresh-old',
      });

      final controller = AuthController(
        client: MockClient((request) async => http.Response('{}', 200)),
      );

      await controller.logout();

      final prefs = await SharedPreferences.getInstance();
      expect(controller.isAuthenticated, isFalse);
      expect(prefs.getString('nl_access_token'), isNull);
      expect(prefs.getString('nl_refresh_token'), isNull);

      controller.dispose();
    });
  });
}
