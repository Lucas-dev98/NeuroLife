import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:mobile_flutter/core/network/api_client.dart';
import 'package:mobile_flutter/features/home/presentation/state/home_controller.dart';

void main() {
  group('HomeController', () {
    test('loads profile, events and gamification data', () async {
      final controller = HomeController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            expect(request.headers['authorization'], 'Bearer test-token');

            switch (request.url.path) {
              case '/api/v1/profile':
                return http.Response(
                  jsonEncode({'id': 12, 'email': 'leo@example.com', 'name': 'Leo'}),
                  200,
                );
              case '/api/v1/gamification/summary':
                return http.Response(
                  jsonEncode({
                    'xp': 140,
                    'level': 2,
                    'current_streak': 4,
                    'longest_streak': 8,
                    'last_activity_day': '2026-06-06',
                  }),
                  200,
                );
              case '/api/v1/gamification/achievements':
                return http.Response(
                  jsonEncode({
                    'achievements': [
                      {
                        'key': 'first-complete',
                        'title': 'Primeira entrega',
                        'description': 'Voce concluiu seu primeiro evento.',
                        'unlocked_at': '2026-06-06T10:00:00Z',
                      },
                    ],
                  }),
                  200,
                );
              case '/api/v1/events':
                return http.Response(
                  jsonEncode({
                    'events': [
                      {
                        'id': 99,
                        'title': 'Consulta',
                        'description': 'Revisao semanal',
                        'start_at': '2026-06-07T10:00:00Z',
                        'end_at': '2026-06-07T11:00:00Z',
                        'is_all_day': false,
                        'completed_at': null,
                      },
                    ],
                    'pagination': {
                      'page': 1,
                      'limit': 5,
                      'total': 1,
                      'total_pages': 1,
                    },
                  }),
                  200,
                );
              case '/api/v1/preferences':
                return http.Response(
                  jsonEncode({
                    'reminder_intensity': 'high',
                    'push_enabled': true,
                    'email_enabled': false,
                    'whatsapp_enabled': true,
                  }),
                  200,
                );
              default:
                return http.Response('not found', 404);
            }
          }),
          accessTokenProvider: () => 'test-token',
        ),
      );

      await controller.load();

      expect(controller.isLoading, isFalse);
      expect(controller.errorMessage, isNull);
      expect(controller.profile?.name, 'Leo');
      expect(controller.summary?.level, 2);
      expect(controller.events, hasLength(1));
      expect(controller.events.first.title, 'Consulta');
      expect(controller.achievements, hasLength(1));
      expect(controller.achievements.first.key, 'first-complete');
      expect(controller.preferences.reminderIntensity, 'high');
      expect(controller.preferences.whatsappEnabled, isTrue);
    });

    test('stores error message when api fails', () async {
      final controller = HomeController(
        apiClient: ApiClient(
          client: MockClient((request) async => http.Response('{"error":"boom"}', 500)),
          accessTokenProvider: () => 'test-token',
        ),
      );

      await controller.load();

      expect(controller.isLoading, isFalse);
      expect(controller.profile, isNull);
      expect(controller.summary, isNull);
      expect(controller.events, isEmpty);
      expect(controller.errorMessage, contains('Falha ao carregar painel (500)'));
    });

    test('creates event and reloads dashboard data', () async {
      var createCalled = false;
      var eventsCalls = 0;

      final controller = HomeController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            switch (request.url.path) {
              case '/api/v1/events':
                if (request.method == 'POST') {
                  createCalled = true;
                  return http.Response(jsonEncode({'id': 55}), 201);
                }
                eventsCalls += 1;
                return http.Response(
                  jsonEncode({
                    'events': [
                      {
                        'id': 55,
                        'title': 'Novo evento',
                        'description': 'Criado via app',
                        'start_at': '2026-06-07T10:00:00Z',
                        'end_at': '2026-06-07T11:00:00Z',
                        'is_all_day': false,
                        'completed_at': null,
                      },
                    ],
                    'pagination': {'page': 1, 'limit': 5, 'total': 1, 'total_pages': 1},
                  }),
                  200,
                );
              case '/api/v1/profile':
                return http.Response(jsonEncode({'id': 1, 'email': 'a@a.com', 'name': 'A'}), 200);
              case '/api/v1/gamification/summary':
                return http.Response(jsonEncode({'xp': 0, 'level': 1, 'current_streak': 0, 'longest_streak': 0}), 200);
              case '/api/v1/gamification/achievements':
                return http.Response(jsonEncode({'achievements': []}), 200);
              case '/api/v1/preferences':
                return http.Response(
                  jsonEncode({
                    'reminder_intensity': 'medium',
                    'push_enabled': true,
                    'email_enabled': true,
                    'whatsapp_enabled': false,
                  }),
                  200,
                );
              default:
                return http.Response('not found', 404);
            }
          }),
          accessTokenProvider: () => 'test-token',
        ),
      );

      await controller.createEvent(
        EventDraft(
          title: 'Novo evento',
          description: 'Criado via app',
          startAt: DateTime.utc(2026, 6, 7, 10),
          endAt: DateTime.utc(2026, 6, 7, 11),
        ),
      );

      expect(createCalled, isTrue);
      expect(eventsCalls, 1);
      expect(controller.events, hasLength(1));
      expect(controller.events.first.id, 55);
    });

    test('completes event and reloads summary', () async {
      var completeCalled = false;

      final controller = HomeController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            if (request.url.path == '/api/v1/events/99/complete') {
              completeCalled = true;
              return http.Response(jsonEncode({'awarded_xp': 50}), 200);
            }
            if (request.url.path == '/api/v1/profile') {
              return http.Response(jsonEncode({'id': 1, 'email': 'a@a.com', 'name': 'A'}), 200);
            }
            if (request.url.path == '/api/v1/gamification/summary') {
              return http.Response(jsonEncode({'xp': 50, 'level': 1, 'current_streak': 1, 'longest_streak': 1}), 200);
            }
            if (request.url.path == '/api/v1/gamification/achievements') {
              return http.Response(jsonEncode({'achievements': []}), 200);
            }
            if (request.url.path == '/api/v1/preferences') {
              return http.Response(
                jsonEncode({
                  'reminder_intensity': 'medium',
                  'push_enabled': true,
                  'email_enabled': true,
                  'whatsapp_enabled': false,
                }),
                200,
              );
            }
            if (request.url.path == '/api/v1/events') {
              return http.Response(
                jsonEncode({
                  'events': [
                    {
                      'id': 99,
                      'title': 'Consulta',
                      'description': 'Revisao',
                      'start_at': '2026-06-07T10:00:00Z',
                      'end_at': '2026-06-07T11:00:00Z',
                      'is_all_day': false,
                      'completed_at': '2026-06-07T10:30:00Z',
                    },
                  ],
                  'pagination': {'page': 1, 'limit': 5, 'total': 1, 'total_pages': 1},
                }),
                200,
              );
            }
            return http.Response('not found', 404);
          }),
          accessTokenProvider: () => 'test-token',
        ),
      );

      await controller.completeEvent(99);

      expect(completeCalled, isTrue);
      expect(controller.summary?.xp, 50);
      expect(controller.events.first.isCompleted, isTrue);
    });

    test('updates profile and preferences locally', () async {
      final controller = HomeController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            if (request.method == 'GET' && request.url.path == '/api/v1/profile') {
              return http.Response(jsonEncode({'id': 7, 'email': 'user@example.com', 'name': 'User'}), 200);
            }
            if (request.method == 'GET' && request.url.path == '/api/v1/gamification/summary') {
              return http.Response(jsonEncode({'xp': 0, 'level': 1, 'current_streak': 0, 'longest_streak': 0}), 200);
            }
            if (request.method == 'GET' && request.url.path == '/api/v1/gamification/achievements') {
              return http.Response(jsonEncode({'achievements': []}), 200);
            }
            if (request.method == 'GET' && request.url.path == '/api/v1/preferences') {
              return http.Response(
                jsonEncode({
                  'reminder_intensity': 'medium',
                  'push_enabled': true,
                  'email_enabled': true,
                  'whatsapp_enabled': false,
                }),
                200,
              );
            }
            if (request.method == 'GET' && request.url.path == '/api/v1/events') {
              return http.Response(jsonEncode({'events': [], 'pagination': {'page': 1, 'limit': 5, 'total': 0, 'total_pages': 0}}), 200);
            }
            if (request.method == 'PUT' && request.url.path == '/api/v1/profile') {
              final body = jsonDecode(request.body) as Map<String, dynamic>;
              expect(body['name'], 'Nova Pessoa');
              return http.Response(jsonEncode({'name': 'Nova Pessoa'}), 200);
            }
            if (request.method == 'PUT' && request.url.path == '/api/v1/preferences') {
              final body = jsonDecode(request.body) as Map<String, dynamic>;
              expect(body['reminder_intensity'], 'low');
              expect(body['push_enabled'], isFalse);
              expect(body['email_enabled'], isTrue);
              expect(body['whatsapp_enabled'], isFalse);
              return http.Response(
                jsonEncode({
                  'reminder_intensity': 'low',
                  'push_enabled': false,
                  'email_enabled': true,
                  'whatsapp_enabled': false,
                }),
                200,
              );
            }
            return http.Response('not found', 404);
          }),
          accessTokenProvider: () => 'test-token',
        ),
      );

      await controller.load();
      await controller.updateProfile('Nova Pessoa');
      await controller.updatePreferences(
        const HomePreferences(
          reminderIntensity: 'low',
          pushEnabled: false,
          emailEnabled: true,
          whatsappEnabled: false,
        ),
      );

      expect(controller.profile?.name, 'Nova Pessoa');
      expect(controller.preferences.reminderIntensity, 'low');
      expect(controller.preferences.pushEnabled, isFalse);
    });
  });
}
