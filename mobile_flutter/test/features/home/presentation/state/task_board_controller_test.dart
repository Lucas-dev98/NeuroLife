import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:mobile_flutter/core/network/api_client.dart';
import 'package:mobile_flutter/features/home/presentation/state/task_board_controller.dart';

void main() {
  group('TaskBoardController', () {
    test('loads tasks from API', () async {
      final controller = TaskBoardController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            expect(request.headers['authorization'], 'Bearer token');
            if (request.url.path == '/api/v1/tasks') {
              return http.Response(
                jsonEncode({
                  'tasks': [
                    {
                      'id': 10,
                      'title': 'Planejar semana',
                      'description': 'Rotina semanal',
                      'category': 'Rotina',
                      'priority': 'high',
                      'due_at': '2026-06-08T10:00:00Z',
                      'completed_at': null,
                      'progress_percent': 50,
                      'checklist': [
                        {
                          'id': 1,
                          'title': 'Definir prioridades',
                          'is_done': true,
                        },
                        {
                          'id': 2,
                          'title': 'Separar horarios',
                          'is_done': false,
                        },
                      ],
                      'created_at': '2026-06-06T10:00:00Z',
                      'updated_at': '2026-06-06T10:30:00Z',
                    },
                  ],
                }),
                200,
              );
            }
            return http.Response('not found', 404);
          }),
          accessTokenProvider: () => 'token',
        ),
      );

      await controller.load();

      expect(controller.errorMessage, isNull);
      expect(controller.tasks, hasLength(1));
      expect(controller.tasks.first.title, 'Planejar semana');
      expect(controller.tasks.first.checklist, hasLength(2));
      expect(controller.tasks.first.progress, closeTo(0.5, 0.01));
    });

    test('creates task and reloads list', () async {
      var created = false;
      var listCalls = 0;

      final controller = TaskBoardController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            if (request.method == 'POST' &&
                request.url.path == '/api/v1/tasks') {
              created = true;
              final body = jsonDecode(request.body) as Map<String, dynamic>;
              expect(body['title'], 'Nova tarefa');
              expect(body['checklist_titles'], ['A', 'B']);
              return http.Response('{}', 201);
            }
            if (request.method == 'GET' &&
                request.url.path == '/api/v1/tasks') {
              listCalls += 1;
              return http.Response(
                jsonEncode({
                  'tasks': [
                    {
                      'id': 77,
                      'title': 'Nova tarefa',
                      'description': '',
                      'category': 'Geral',
                      'priority': 'medium',
                      'due_at': null,
                      'completed_at': null,
                      'progress_percent': 0,
                      'checklist': [],
                      'created_at': '2026-06-06T10:00:00Z',
                      'updated_at': '2026-06-06T10:00:00Z',
                    },
                  ],
                }),
                200,
              );
            }
            return http.Response('not found', 404);
          }),
          accessTokenProvider: () => 'token',
        ),
      );

      await controller.addTask(
        const TaskDraft(
          title: 'Nova tarefa',
          description: '',
          category: 'Geral',
          priority: 'medium',
          dueAt: null,
          checklistTitles: ['A', 'B'],
        ),
      );

      expect(created, isTrue);
      expect(listCalls, 1);
      expect(controller.tasks, hasLength(1));
      expect(controller.tasks.first.id, 77);
    });

    test('toggles checklist item and reloads task list', () async {
      var patchCalled = false;

      final controller = TaskBoardController(
        apiClient: ApiClient(
          client: MockClient((request) async {
            if (request.method == 'GET' &&
                request.url.path == '/api/v1/tasks') {
              return http.Response(
                jsonEncode({
                  'tasks': [
                    {
                      'id': 10,
                      'title': 'Planejar semana',
                      'description': '',
                      'category': 'Rotina',
                      'priority': 'high',
                      'due_at': null,
                      'completed_at': null,
                      'progress_percent': 50,
                      'checklist': [
                        {
                          'id': 1,
                          'title': 'Definir prioridades',
                          'is_done': true,
                        },
                        {
                          'id': 2,
                          'title': 'Separar horarios',
                          'is_done': false,
                        },
                      ],
                      'created_at': '2026-06-06T10:00:00Z',
                      'updated_at': '2026-06-06T10:30:00Z',
                    },
                  ],
                }),
                200,
              );
            }
            if (request.method == 'PATCH' &&
                request.url.path == '/api/v1/tasks/10/checklist/2') {
              patchCalled = true;
              final body = jsonDecode(request.body) as Map<String, dynamic>;
              expect(body['is_done'], isTrue);
              return http.Response('{}', 200);
            }
            return http.Response('not found', 404);
          }),
          accessTokenProvider: () => 'token',
        ),
      );

      await controller.load();
      await controller.toggleChecklistItem(10, 2);

      expect(patchCalled, isTrue);
    });
  });
}
