import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:mobile_flutter/features/home/presentation/state/task_board_controller.dart';

void main() {
  group('TaskBoardController', () {
    test('loads empty state when storage is empty', () async {
      SharedPreferences.setMockInitialValues({});

      final controller = TaskBoardController();
      await controller.load();

      expect(controller.isLoading, isFalse);
      expect(controller.errorMessage, isNull);
      expect(controller.tasks, isEmpty);
    });

    test('adds task and toggles checklist items', () async {
      SharedPreferences.setMockInitialValues({});

      final controller = TaskBoardController();
      await controller.load();
      await controller.addTask(
        const TaskDraft(
          title: 'Planejar semana',
          description: 'Separar blocos de foco',
          category: 'Rotina',
          priority: 'high',
          dueAt: null,
          checklistTitles: ['Definir 3 prioridades', 'Bloquear horario de foco'],
        ),
      );

      expect(controller.tasks, hasLength(1));
      final task = controller.tasks.first;
      expect(task.title, 'Planejar semana');
      expect(task.checklist, hasLength(2));

      await controller.toggleChecklistItem(task.id, task.checklist.first.id);

      final updatedTask = controller.taskById(task.id)!;
      expect(updatedTask.checklist.first.isDone, isTrue);
      expect(updatedTask.progress, closeTo(0.5, 0.01));
    });

    test('persists tasks across reloads', () async {
      SharedPreferences.setMockInitialValues({});

      final firstController = TaskBoardController();
      await firstController.load();
      await firstController.addTask(
        const TaskDraft(
          title: 'Comprar materiais',
          description: '',
          category: 'Casa',
          priority: 'medium',
          dueAt: null,
          checklistTitles: ['Lista', 'Ir ao mercado'],
        ),
      );

      final secondController = TaskBoardController();
      await secondController.load();

      expect(secondController.tasks, hasLength(1));
      expect(secondController.tasks.first.title, 'Comprar materiais');
      expect(secondController.tasks.first.checklist, hasLength(2));
    });
  });
}