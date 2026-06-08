import 'package:flutter/foundation.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/api_exception.dart';

class TaskChecklistItem {
  const TaskChecklistItem({
    required this.id,
    required this.title,
    required this.isDone,
  });

  final int id;
  final String title;
  final bool isDone;

  TaskChecklistItem copyWith({String? title, bool? isDone}) {
    return TaskChecklistItem(
      id: id,
      title: title ?? this.title,
      isDone: isDone ?? this.isDone,
    );
  }
}

class TaskItem {
  const TaskItem({
    required this.id,
    required this.title,
    required this.description,
    required this.category,
    required this.priority,
    required this.dueAt,
    required this.checklist,
    required this.createdAt,
    required this.updatedAt,
    required this.progressPercent,
    required this.dependsOnTaskId,
    required this.isBlocked,
  });

  final int id;
  final String title;
  final String description;
  final String category;
  final String priority;
  final DateTime? dueAt;
  final List<TaskChecklistItem> checklist;
  final DateTime createdAt;
  final DateTime updatedAt;
  final int progressPercent;
  final int? dependsOnTaskId;
  final bool isBlocked;

  double get progress {
    if (checklist.isEmpty) {
      return 0;
    }
    return progressPercent / 100;
  }

  bool get isCompleted =>
      checklist.isNotEmpty && checklist.every((item) => item.isDone);
}

class TaskDraft {
  const TaskDraft({
    required this.title,
    required this.description,
    required this.category,
    required this.priority,
    required this.dueAt,
    required this.checklistTitles,
    required this.dependsOnTaskId,
  });

  final String title;
  final String description;
  final String category;
  final String priority;
  final DateTime? dueAt;
  final List<String> checklistTitles;
  final int? dependsOnTaskId;
}

class TaskBoardController extends ChangeNotifier {
  TaskBoardController({required this.apiClient});

  final ApiClient apiClient;

  bool _isLoading = false;
  String? _errorMessage;
  List<TaskItem> _tasks = const [];

  bool get isLoading => _isLoading;
  String? get errorMessage => _errorMessage;
  List<TaskItem> get tasks => _tasks;

  Future<void> load() async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final response = await apiClient.get(
        '/api/v1/tasks?page=1&limit=100',
        requiresAuth: true,
      );
      _requireSuccess(response.statusCode, action: 'carregar tarefas');

      final data = apiClient.decodeJsonObject(response.body);
      final items = data['tasks'] as List<dynamic>? ?? const [];
      _tasks = items.whereType<Map>().map((item) => _mapTask(item)).toList();
    } catch (error) {
      _errorMessage = error.toString();
      _tasks = const [];
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> addTask(TaskDraft draft) async {
    final response = await apiClient.post(
      '/api/v1/tasks',
      requiresAuth: true,
      body: _taskPayload(draft),
    );
    _requireSuccess(response.statusCode, action: 'criar tarefa');
    await load();
  }

  Future<void> updateTask(int taskId, TaskDraft draft) async {
    final response = await apiClient.put(
      '/api/v1/tasks/$taskId',
      requiresAuth: true,
      body: _taskPayload(draft),
    );
    _requireSuccess(response.statusCode, action: 'atualizar tarefa');
    await load();
  }

  Future<void> deleteTask(int taskId) async {
    final response = await apiClient.delete(
      '/api/v1/tasks/$taskId',
      requiresAuth: true,
    );
    if (response.statusCode != 204 &&
        (response.statusCode < 200 || response.statusCode >= 300)) {
      _requireSuccess(response.statusCode, action: 'excluir tarefa');
    }
    await load();
  }

  Future<void> toggleChecklistItem(int taskId, int checklistItemId) async {
    final task = taskById(taskId);
    if (task == null) {
      throw ApiException('Tarefa nao encontrada no estado local.');
    }

    final item = task.checklist.firstWhere(
      (candidate) => candidate.id == checklistItemId,
      orElse: () => throw ApiException('Item de checklist nao encontrado.'),
    );

    final response = await apiClient.patch(
      '/api/v1/tasks/$taskId/checklist/$checklistItemId',
      requiresAuth: true,
      body: {'is_done': !item.isDone},
    );
    _requireSuccess(response.statusCode, action: 'atualizar item do checklist');
    await load();
  }

  Future<void> addChecklistItem(int taskId, String title) async {
    final response = await apiClient.post(
      '/api/v1/tasks/$taskId/checklist',
      requiresAuth: true,
      body: {'title': title.trim()},
    );
    _requireSuccess(response.statusCode, action: 'adicionar item no checklist');
    await load();
  }

  Future<void> deleteChecklistItem(int taskId, int checklistItemId) async {
    final response = await apiClient.delete(
      '/api/v1/tasks/$taskId/checklist/$checklistItemId',
      requiresAuth: true,
    );
    _requireSuccess(response.statusCode, action: 'remover item do checklist');
    await load();
  }

  TaskItem? taskById(int taskId) {
    for (final task in _tasks) {
      if (task.id == taskId) {
        return task;
      }
    }
    return null;
  }

  TaskItem _mapTask(Map raw) {
    final checklistRaw = raw['checklist'] as List<dynamic>? ?? const [];
    final checklist = checklistRaw
        .whereType<Map>()
        .map(
          (item) => TaskChecklistItem(
            id: (item['id'] as num?)?.toInt() ?? 0,
            title: item['title']?.toString() ?? '',
            isDone: item['is_done'] == true,
          ),
        )
        .toList();

    return TaskItem(
      id: (raw['id'] as num?)?.toInt() ?? 0,
      title: raw['title']?.toString() ?? '',
      description: raw['description']?.toString() ?? '',
      category: raw['category']?.toString() ?? '',
      priority: raw['priority']?.toString() ?? 'medium',
      dueAt: DateTime.tryParse(raw['due_at']?.toString() ?? ''),
      checklist: checklist,
      createdAt:
          DateTime.tryParse(raw['created_at']?.toString() ?? '') ??
          DateTime.now().toUtc(),
      updatedAt:
          DateTime.tryParse(raw['updated_at']?.toString() ?? '') ??
          DateTime.now().toUtc(),
      progressPercent: (raw['progress_percent'] as num?)?.toInt() ?? 0,
      dependsOnTaskId: (raw['depends_on_task_id'] as num?)?.toInt(),
      isBlocked: raw['is_blocked'] == true,
    );
  }

  Map<String, dynamic> _taskPayload(TaskDraft draft) {
    return {
      'title': draft.title.trim(),
      'description': draft.description.trim(),
      'category': draft.category.trim(),
      'priority': draft.priority,
      'due_at': draft.dueAt?.toUtc().toIso8601String(),
      'checklist_titles': draft.checklistTitles,
      'depends_on_task_id': draft.dependsOnTaskId,
    };
  }

  void _requireSuccess(int statusCode, {required String action}) {
    if (statusCode < 200 || statusCode >= 300) {
      throw ApiException(
        'Falha ao $action ($statusCode).',
        statusCode: statusCode,
      );
    }
  }
}
