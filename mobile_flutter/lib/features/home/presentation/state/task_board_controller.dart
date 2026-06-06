import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

class TaskChecklistItem {
  const TaskChecklistItem({
    required this.id,
    required this.title,
    required this.isDone,
  });

  final String id;
  final String title;
  final bool isDone;

  TaskChecklistItem copyWith({String? title, bool? isDone}) {
    return TaskChecklistItem(
      id: id,
      title: title ?? this.title,
      isDone: isDone ?? this.isDone,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'title': title,
      'is_done': isDone,
    };
  }

  factory TaskChecklistItem.fromJson(Map<String, dynamic> json) {
    return TaskChecklistItem(
      id: json['id']?.toString() ?? '',
      title: json['title']?.toString() ?? '',
      isDone: json['is_done'] == true,
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

  double get progress {
    if (checklist.isEmpty) {
      return 0;
    }
    final doneCount = checklist.where((item) => item.isDone).length;
    return doneCount / checklist.length;
  }

  bool get isCompleted => checklist.isNotEmpty && checklist.every((item) => item.isDone);

  TaskItem copyWith({
    String? title,
    String? description,
    String? category,
    String? priority,
    DateTime? dueAt,
    List<TaskChecklistItem>? checklist,
    DateTime? updatedAt,
  }) {
    return TaskItem(
      id: id,
      title: title ?? this.title,
      description: description ?? this.description,
      category: category ?? this.category,
      priority: priority ?? this.priority,
      dueAt: dueAt ?? this.dueAt,
      checklist: checklist ?? this.checklist,
      createdAt: createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'title': title,
      'description': description,
      'category': category,
      'priority': priority,
      'due_at': dueAt?.toIso8601String(),
      'checklist': checklist.map((item) => item.toJson()).toList(),
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }

  factory TaskItem.fromJson(Map<String, dynamic> json) {
    final checklistItems = json['checklist'] as List<dynamic>? ?? const [];
    return TaskItem(
      id: (json['id'] as num?)?.toInt() ?? 0,
      title: json['title']?.toString() ?? '',
      description: json['description']?.toString() ?? '',
      category: json['category']?.toString() ?? '',
      priority: json['priority']?.toString() ?? 'medium',
      dueAt: DateTime.tryParse(json['due_at']?.toString() ?? ''),
      checklist: checklistItems
          .whereType<Map>()
          .map((item) => TaskChecklistItem.fromJson(item.cast<String, dynamic>()))
          .toList(),
      createdAt: DateTime.tryParse(json['created_at']?.toString() ?? '') ?? DateTime.now().toUtc(),
      updatedAt: DateTime.tryParse(json['updated_at']?.toString() ?? '') ?? DateTime.now().toUtc(),
    );
  }
}

class TaskDraft {
  const TaskDraft({
    required this.title,
    required this.description,
    required this.category,
    required this.priority,
    required this.dueAt,
    required this.checklistTitles,
  });

  final String title;
  final String description;
  final String category;
  final String priority;
  final DateTime? dueAt;
  final List<String> checklistTitles;
}

class TaskBoardController extends ChangeNotifier {
  TaskBoardController({this._prefs});

  static const _storageKey = 'nl_tasks_v1';

  final SharedPreferences? _prefs;

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
      final prefs = _prefs ?? await SharedPreferences.getInstance();
      final raw = prefs.getString(_storageKey);
      if (raw == null || raw.trim().isEmpty) {
        _tasks = const [];
        return;
      }

      final decoded = jsonDecode(raw);
      if (decoded is! List) {
        throw StateError('Formato de tarefas invalido.');
      }

      _tasks = decoded
          .whereType<Map>()
          .map((item) => TaskItem.fromJson(item.cast<String, dynamic>()))
          .toList()
        ..sort((left, right) => _compareTasks(left, right));
    } catch (error) {
      _errorMessage = error.toString();
      _tasks = const [];
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> addTask(TaskDraft draft) async {
    final now = DateTime.now().toUtc();
    final task = TaskItem(
      id: now.microsecondsSinceEpoch,
      title: draft.title.trim(),
      description: draft.description.trim(),
      category: draft.category.trim(),
      priority: draft.priority,
      dueAt: draft.dueAt?.toUtc(),
      checklist: draft.checklistTitles
          .map((title) => title.trim())
          .where((title) => title.isNotEmpty)
          .map(
            (title) => TaskChecklistItem(
              id: '${now.microsecondsSinceEpoch}-${title.hashCode.abs()}',
              title: title,
              isDone: false,
            ),
          )
          .toList(),
      createdAt: now,
      updatedAt: now,
    );

    _tasks = [..._tasks, task]..sort((left, right) => _compareTasks(left, right));
    await _persist();
  }

  Future<void> updateTask(int taskId, TaskDraft draft) async {
    _tasks = _tasks.map((task) {
      if (task.id != taskId) {
        return task;
      }
      return task.copyWith(
        title: draft.title.trim(),
        description: draft.description.trim(),
        category: draft.category.trim(),
        priority: draft.priority,
        dueAt: draft.dueAt?.toUtc(),
        updatedAt: DateTime.now().toUtc(),
      );
    }).toList()
      ..sort((left, right) => _compareTasks(left, right));

    await _persist();
  }

  Future<void> deleteTask(int taskId) async {
    _tasks = _tasks.where((task) => task.id != taskId).toList();
    await _persist();
  }

  Future<void> toggleChecklistItem(int taskId, String checklistItemId) async {
    _tasks = _tasks.map((task) {
      if (task.id != taskId) {
        return task;
      }

      final updatedChecklist = task.checklist
          .map((item) => item.id == checklistItemId ? item.copyWith(isDone: !item.isDone) : item)
          .toList();
      return task.copyWith(updatedAt: DateTime.now().toUtc(), checklist: updatedChecklist);
    }).toList();

    await _persist();
  }

  Future<void> addChecklistItem(int taskId, String title) async {
    final trimmedTitle = title.trim();
    if (trimmedTitle.isEmpty) {
      return;
    }

    final now = DateTime.now().toUtc();
    _tasks = _tasks.map((task) {
      if (task.id != taskId) {
        return task;
      }
      final updatedChecklist = [
        ...task.checklist,
        TaskChecklistItem(
          id: '${now.microsecondsSinceEpoch}-${trimmedTitle.hashCode.abs()}',
          title: trimmedTitle,
          isDone: false,
        ),
      ];
      return task.copyWith(updatedAt: now, checklist: updatedChecklist);
    }).toList();

    await _persist();
  }

  Future<void> deleteChecklistItem(int taskId, String checklistItemId) async {
    _tasks = _tasks.map((task) {
      if (task.id != taskId) {
        return task;
      }
      final updatedChecklist = task.checklist.where((item) => item.id != checklistItemId).toList();
      return task.copyWith(updatedAt: DateTime.now().toUtc(), checklist: updatedChecklist);
    }).toList();

    await _persist();
  }

  TaskItem? taskById(int taskId) {
    for (final task in _tasks) {
      if (task.id == taskId) {
        return task;
      }
    }
    return null;
  }

  int _compareTasks(TaskItem left, TaskItem right) {
    final priorityRank = _priorityWeight(right.priority).compareTo(_priorityWeight(left.priority));
    if (priorityRank != 0) {
      return priorityRank;
    }

    final leftDue = left.dueAt ?? DateTime.fromMillisecondsSinceEpoch(0);
    final rightDue = right.dueAt ?? DateTime.fromMillisecondsSinceEpoch(0);
    return leftDue.compareTo(rightDue);
  }

  int _priorityWeight(String priority) {
    switch (priority.toLowerCase()) {
      case 'urgent':
        return 3;
      case 'high':
        return 2;
      case 'medium':
        return 1;
      default:
        return 0;
    }
  }

  Future<void> _persist() async {
    final prefs = _prefs ?? await SharedPreferences.getInstance();
    final encoded = jsonEncode(_tasks.map((task) => task.toJson()).toList());
    await prefs.setString(_storageKey, encoded);
    notifyListeners();
  }
}