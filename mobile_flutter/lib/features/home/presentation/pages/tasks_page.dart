import 'package:flutter/material.dart';

import '../state/task_board_controller.dart';

class TasksPage extends StatefulWidget {
  const TasksPage({super.key, required this.controller});

  final TaskBoardController controller;

  @override
  State<TasksPage> createState() => _TasksPageState();
}

class _TasksPageState extends State<TasksPage> {
  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: widget.controller,
      builder: (context, _) {
        final tasks = widget.controller.tasks;

        return ListView(
          key: const ValueKey('tasks'),
          padding: const EdgeInsets.all(20),
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    'Tarefas',
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
                FilledButton.icon(
                  onPressed: () => _openTaskDialog(context),
                  icon: const Icon(Icons.add_rounded),
                  label: const Text('Nova'),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              'Tarefas e checklist sincronizados com o backend.',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 16),
            if (tasks.isEmpty)
              const _EmptyTaskCard()
            else
              for (final task in tasks)
                _TaskCard(
                  task: task,
                  onEdit: () => _openTaskDialog(context, existingTask: task),
                  onDelete: () => _deleteTask(context, task),
                  onAddChecklistItem: () => _addChecklistItem(context, task),
                  onToggleChecklistItem: (itemId) =>
                      widget.controller.toggleChecklistItem(task.id, itemId),
                  onDeleteChecklistItem: (itemId) =>
                      widget.controller.deleteChecklistItem(task.id, itemId),
                ),
          ],
        );
      },
    );
  }

  Future<void> _openTaskDialog(
    BuildContext context, {
    TaskItem? existingTask,
  }) async {
    final draft = await showDialog<TaskDraft>(
      context: context,
      builder: (dialogContext) => _TaskDialog(existingTask: existingTask),
    );

    if (draft == null) {
      return;
    }

    if (existingTask == null) {
      await widget.controller.addTask(draft);
    } else {
      await widget.controller.updateTask(existingTask.id, draft);
    }
  }

  Future<void> _deleteTask(BuildContext context, TaskItem task) async {
    final confirmed =
        await showDialog<bool>(
          context: context,
          builder: (dialogContext) => AlertDialog(
            title: const Text('Excluir tarefa'),
            content: Text('Deseja excluir "${task.title}"?'),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(false),
                child: const Text('Cancelar'),
              ),
              FilledButton(
                onPressed: () => Navigator.of(dialogContext).pop(true),
                child: const Text('Excluir'),
              ),
            ],
          ),
        ) ??
        false;

    if (confirmed) {
      await widget.controller.deleteTask(task.id);
    }
  }

  Future<void> _addChecklistItem(BuildContext context, TaskItem task) async {
    final titleController = TextEditingController();
    final result = await showDialog<String>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Nova subtarefa'),
        content: TextField(
          controller: titleController,
          autofocus: true,
          decoration: const InputDecoration(labelText: 'Titulo da subtarefa'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text('Cancelar'),
          ),
          FilledButton(
            onPressed: () =>
                Navigator.of(dialogContext).pop(titleController.text),
            child: const Text('Adicionar'),
          ),
        ],
      ),
    );

    titleController.dispose();
    if (result == null || result.trim().isEmpty) {
      return;
    }

    await widget.controller.addChecklistItem(task.id, result);
  }
}

class _TaskCard extends StatelessWidget {
  const _TaskCard({
    required this.task,
    required this.onEdit,
    required this.onDelete,
    required this.onAddChecklistItem,
    required this.onToggleChecklistItem,
    required this.onDeleteChecklistItem,
  });

  final TaskItem task;
  final VoidCallback onEdit;
  final VoidCallback onDelete;
  final VoidCallback onAddChecklistItem;
  final void Function(int checklistItemId) onToggleChecklistItem;
  final void Function(int checklistItemId) onDeleteChecklistItem;

  @override
  Widget build(BuildContext context) {
    final percent = (task.progress * 100).round();

    return Material(
      color: Colors.white,
      elevation: 6,
      shadowColor: const Color(0x14000000),
      borderRadius: BorderRadius.circular(18),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: const Color(0x110E5A8A),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: const Icon(
                    Icons.checklist_rounded,
                    color: Color(0xFF0E5A8A),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        task.title,
                        style: const TextStyle(
                          fontWeight: FontWeight.w700,
                          fontSize: 16,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(_buildMeta(task)),
                    ],
                  ),
                ),
                PopupMenuButton<String>(
                  onSelected: (value) {
                    switch (value) {
                      case 'edit':
                        onEdit();
                        break;
                      case 'delete':
                        onDelete();
                        break;
                    }
                  },
                  itemBuilder: (context) => const [
                    PopupMenuItem(value: 'edit', child: Text('Editar')),
                    PopupMenuItem(value: 'delete', child: Text('Excluir')),
                  ],
                ),
              ],
            ),
            if (task.description.isNotEmpty) ...[
              const SizedBox(height: 12),
              Text(task.description),
            ],
            const SizedBox(height: 12),
            LinearProgressIndicator(
              value: task.progress == 0 ? 0.02 : task.progress,
            ),
            const SizedBox(height: 6),
            Text('$percent% concluido'),
            const SizedBox(height: 12),
            if (task.checklist.isEmpty)
              TextButton.icon(
                onPressed: onAddChecklistItem,
                icon: const Icon(Icons.playlist_add_rounded),
                label: const Text('Adicionar checklist'),
              )
            else
              Column(
                children: [
                  for (final item in task.checklist)
                    Material(
                      color: Colors.transparent,
                      child: CheckboxListTile(
                        contentPadding: EdgeInsets.zero,
                        value: item.isDone,
                        onChanged: (_) => onToggleChecklistItem(item.id),
                        title: Text(item.title),
                        secondary: IconButton(
                          tooltip: 'Remover subtarefa',
                          onPressed: () => onDeleteChecklistItem(item.id),
                          icon: const Icon(Icons.close_rounded),
                        ),
                      ),
                    ),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: TextButton.icon(
                      onPressed: onAddChecklistItem,
                      icon: const Icon(Icons.playlist_add_rounded),
                      label: const Text('Adicionar item'),
                    ),
                  ),
                ],
              ),
          ],
        ),
      ),
    );
  }

  String _buildMeta(TaskItem task) {
    final due = task.dueAt == null
        ? 'Sem prazo'
        : _formatDate(task.dueAt!.toLocal());
    return '${task.category} • ${task.priority.toUpperCase()} • $due';
  }

  String _formatDate(DateTime value) {
    final day = value.day.toString().padLeft(2, '0');
    final month = value.month.toString().padLeft(2, '0');
    final hour = value.hour.toString().padLeft(2, '0');
    final minute = value.minute.toString().padLeft(2, '0');
    return '$day/$month $hour:$minute';
  }
}

class _EmptyTaskCard extends StatelessWidget {
  const _EmptyTaskCard();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        boxShadow: const [
          BoxShadow(
            color: Color(0x14000000),
            blurRadius: 16,
            offset: Offset(0, 6),
          ),
        ],
      ),
      child: const Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Nenhuma tarefa criada ainda',
            style: TextStyle(fontWeight: FontWeight.w700),
          ),
          SizedBox(height: 4),
          Text(
            'Crie tarefas com checklist manual para organizar a execucao em passos pequenos.',
          ),
        ],
      ),
    );
  }
}

class _TaskDialog extends StatefulWidget {
  const _TaskDialog({this.existingTask});

  final TaskItem? existingTask;

  @override
  State<_TaskDialog> createState() => _TaskDialogState();
}

class _TaskDialogState extends State<_TaskDialog> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _titleController;
  late final TextEditingController _descriptionController;
  late final TextEditingController _categoryController;
  late final TextEditingController _checklistController;
  late String _priority;
  DateTime? _dueAt;
  final List<String> _checklistTitles = [];

  @override
  void initState() {
    super.initState();
    final existing = widget.existingTask;
    _titleController = TextEditingController(text: existing?.title ?? '');
    _descriptionController = TextEditingController(
      text: existing?.description ?? '',
    );
    _categoryController = TextEditingController(
      text: existing?.category ?? 'Geral',
    );
    _checklistController = TextEditingController();
    _priority = existing?.priority ?? 'medium';
    _dueAt = existing?.dueAt?.toLocal();
    if (existing != null) {
      _checklistTitles.addAll(existing.checklist.map((item) => item.title));
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _categoryController.dispose();
    _checklistController.dispose();
    super.dispose();
  }

  void _addChecklistTitle() {
    final text = _checklistController.text.trim();
    if (text.isEmpty) {
      return;
    }
    setState(() {
      _checklistTitles.add(text);
      _checklistController.clear();
    });
  }

  Future<void> _pickDueDate() async {
    final pickedDate = await showDatePicker(
      context: context,
      initialDate: _dueAt ?? DateTime.now(),
      firstDate: DateTime.now().subtract(const Duration(days: 365)),
      lastDate: DateTime.now().add(const Duration(days: 3650)),
    );
    if (pickedDate == null) {
      return;
    }
    if (!mounted) {
      return;
    }

    final pickedTime = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(_dueAt ?? DateTime.now()),
    );
    if (pickedTime == null) {
      return;
    }
    if (!mounted) {
      return;
    }

    setState(() {
      _dueAt = DateTime(
        pickedDate.year,
        pickedDate.month,
        pickedDate.day,
        pickedTime.hour,
        pickedTime.minute,
      );
    });
  }

  void _save() {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    Navigator.of(context).pop(
      TaskDraft(
        title: _titleController.text,
        description: _descriptionController.text,
        category: _categoryController.text,
        priority: _priority,
        dueAt: _dueAt,
        checklistTitles: _checklistTitles,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isEditing = widget.existingTask != null;

    return AlertDialog(
      title: Text(isEditing ? 'Editar tarefa' : 'Nova tarefa'),
      content: SingleChildScrollView(
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: _titleController,
                decoration: const InputDecoration(labelText: 'Titulo'),
                validator: (value) {
                  final text = value?.trim() ?? '';
                  if (text.length < 3) {
                    return 'Informe um titulo com pelo menos 3 caracteres.';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _descriptionController,
                decoration: const InputDecoration(labelText: 'Descricao'),
                minLines: 2,
                maxLines: 4,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _categoryController,
                decoration: const InputDecoration(labelText: 'Categoria'),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _priority,
                decoration: const InputDecoration(labelText: 'Prioridade'),
                items: const [
                  DropdownMenuItem(value: 'low', child: Text('Baixa')),
                  DropdownMenuItem(value: 'medium', child: Text('Media')),
                  DropdownMenuItem(value: 'high', child: Text('Alta')),
                  DropdownMenuItem(value: 'urgent', child: Text('Urgente')),
                ],
                onChanged: (value) =>
                    setState(() => _priority = value ?? 'medium'),
              ),
              const SizedBox(height: 12),
              ListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('Prazo'),
                subtitle: Text(
                  _dueAt == null ? 'Sem prazo' : _formatDate(_dueAt!),
                ),
                trailing: const Icon(Icons.event_rounded),
                onTap: _pickDueDate,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _checklistController,
                decoration: InputDecoration(
                  labelText: 'Subtarefa',
                  suffixIcon: IconButton(
                    onPressed: _addChecklistTitle,
                    icon: const Icon(Icons.add_rounded),
                  ),
                ),
                onFieldSubmitted: (_) => _addChecklistTitle(),
              ),
              const SizedBox(height: 8),
              if (_checklistTitles.isNotEmpty)
                Align(
                  alignment: Alignment.centerLeft,
                  child: Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      for (final item in _checklistTitles)
                        Chip(
                          label: Text(item),
                          onDeleted: () =>
                              setState(() => _checklistTitles.remove(item)),
                        ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancelar'),
        ),
        FilledButton(
          onPressed: _save,
          child: Text(isEditing ? 'Salvar' : 'Criar'),
        ),
      ],
    );
  }

  String _formatDate(DateTime value) {
    final day = value.day.toString().padLeft(2, '0');
    final month = value.month.toString().padLeft(2, '0');
    final year = value.year.toString();
    final hour = value.hour.toString().padLeft(2, '0');
    final minute = value.minute.toString().padLeft(2, '0');
    return '$day/$month/$year $hour:$minute';
  }
}
