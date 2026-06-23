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

class TaskReminderPreview {
  const TaskReminderPreview({
    required this.id,
    required this.triggerAt,
    required this.offsetMinutes,
    required this.channels,
    required this.status,
  });

  final int id;
  final DateTime triggerAt;
  final int offsetMinutes;
  final List<String> channels;
  final String status;
}

class TaskAiSuggestion {
  const TaskAiSuggestion({
    required this.id,
    required this.userId,
    required this.taskId,
    required this.titleInput,
    required this.context,
    required this.source,
    required this.subtasks,
    required this.applied,
    required this.replaceExisting,
    required this.createdAt,
  });

  final int id;
  final int userId;
  final int? taskId;
  final String titleInput;
  final String context;
  final String source;
  final List<String> subtasks;
  final bool applied;
  final bool replaceExisting;
  final DateTime createdAt;
}

class TaskAiSuggestionPage {
  const TaskAiSuggestionPage({
    required this.items,
    required this.page,
    required this.limit,
    required this.total,
    required this.totalPages,
    required this.taskId,
  });

  final List<TaskAiSuggestion> items;
  final int page;
  final int limit;
  final int total;
  final int totalPages;
  final int? taskId;
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
    required this.nextReminders,
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
  final List<TaskReminderPreview> nextReminders;

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

  Future<TaskAiSuggestionPage> loadAiSuggestions({
    int? taskId,
    int page = 1,
    int limit = 20,
  }) async {
    final params = <String, String>{
      'page': page.toString(),
      'limit': limit.toString(),
    };
    if (taskId != null) {
      params['task_id'] = taskId.toString();
    }

    final response = await apiClient.get(
      '/api/v1/tasks/ai-suggestions?${Uri(queryParameters: params).query}',
      requiresAuth: true,
    );
    _requireSuccess(response.statusCode, action: 'carregar historico de IA');

    final data = apiClient.decodeJsonObject(response.body);
    final itemsRaw = data['items'] as List<dynamic>? ?? const [];
    final items = itemsRaw.whereType<Map>().map(_mapAiSuggestion).toList();
    final pagination = data['pagination'] as Map? ?? const {};

    return TaskAiSuggestionPage(
      items: items,
      page: (pagination['page'] as num?)?.toInt() ?? page,
      limit: (pagination['limit'] as num?)?.toInt() ?? limit,
      total: (pagination['total'] as num?)?.toInt() ?? items.length,
      totalPages: (pagination['total_pages'] as num?)?.toInt() ?? 1,
      taskId: (data['task_id'] as num?)?.toInt(),
    );
  }

  Future<void> applyAiChecklist({
    required int taskId,
    String? title,
    String? context,
    bool replaceExisting = false,
  }) async {
    final payload = <String, dynamic>{'replace_existing': replaceExisting};

    final trimmedTitle = title?.trim() ?? '';
    if (trimmedTitle.isNotEmpty) {
      payload['title'] = trimmedTitle;
    }

    final trimmedContext = context?.trim() ?? '';
    if (trimmedContext.isNotEmpty) {
      payload['context'] = trimmedContext;
    }

    final response = await apiClient.post(
      '/api/v1/tasks/$taskId/decompose-apply',
      requiresAuth: true,
      body: payload,
    );
    _requireSuccess(response.statusCode, action: 'aplicar checklist com IA');
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

    final remindersRaw = raw['next_reminders'] as List<dynamic>? ?? const [];
    final nextReminders = remindersRaw
        .whereType<Map>()
        .map(
          (item) => TaskReminderPreview(
            id: (item['id'] as num?)?.toInt() ?? 0,
            triggerAt:
                DateTime.tryParse(item['trigger_at']?.toString() ?? '') ??
                DateTime.now().toUtc(),
            offsetMinutes: (item['offset_minutes'] as num?)?.toInt() ?? 0,
            channels: (item['channels'] as List<dynamic>? ?? const [])
                .map((channel) => channel.toString())
                .toList(),
            status: item['status']?.toString() ?? 'scheduled',
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
      nextReminders: nextReminders,
    );
  }

  TaskAiSuggestion _mapAiSuggestion(Map raw) {
    final subtasksRaw = raw['subtasks'] as List<dynamic>? ?? const [];
    return TaskAiSuggestion(
      id: (raw['id'] as num?)?.toInt() ?? 0,
      userId: (raw['user_id'] as num?)?.toInt() ?? 0,
      taskId: (raw['task_id'] as num?)?.toInt(),
      titleInput: raw['title_input']?.toString() ?? '',
      context: raw['context']?.toString() ?? '',
      source: raw['source']?.toString() ?? 'ai-python',
      subtasks: subtasksRaw.map((subtask) => subtask.toString()).toList(),
      applied: raw['applied'] == true,
      replaceExisting: raw['replace_existing'] == true,
      createdAt:
          DateTime.tryParse(raw['created_at']?.toString() ?? '') ??
          DateTime.now().toUtc(),
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
