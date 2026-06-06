import 'package:flutter/material.dart';

import '../state/home_controller.dart';

class AgendaPage extends StatefulWidget {
  const AgendaPage({super.key, required this.controller});

  final HomeController controller;

  @override
  State<AgendaPage> createState() => _AgendaPageState();
}

class _AgendaPageState extends State<AgendaPage> {
  Future<void> _createEvent() async {
    final draft = await _showEventDialog(context);
    if (draft == null) return;

    await _runAction(
      'Evento criado com sucesso.',
      () => widget.controller.createEvent(draft),
    );
  }

  Future<void> _editEvent(HomeEvent event) async {
    final draft = await _showEventDialog(context, existingEvent: event);
    if (draft == null) return;

    await _runAction(
      'Evento atualizado com sucesso.',
      () => widget.controller.updateEvent(event.id, draft),
    );
  }

  Future<void> _deleteEvent(HomeEvent event) async {
    final confirmed = await showDialog<bool>(
          context: context,
          builder: (dialogContext) => AlertDialog(
            title: const Text('Excluir evento'),
            content: Text('Deseja excluir "${event.title}"?'),
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

    if (!confirmed) return;

    await _runAction(
      'Evento excluido com sucesso.',
      () => widget.controller.deleteEvent(event.id),
    );
  }

  Future<void> _completeEvent(HomeEvent event) async {
    await _runAction(
      event.isCompleted ? 'Evento ja estava concluido.' : 'Evento concluido com sucesso.',
      () => widget.controller.completeEvent(event.id),
    );
  }

  Future<void> _runAction(String successMessage, Future<void> Function() action) async {
    try {
      await action();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(successMessage)),
      );
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(error.toString())),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final events = widget.controller.events;

    return ListView(
      key: const ValueKey('agenda'),
      padding: const EdgeInsets.all(20),
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                'Agenda',
                style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
              ),
            ),
            FilledButton.icon(
              onPressed: _createEvent,
              icon: const Icon(Icons.add_rounded),
              label: const Text('Novo'),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Text(
          'Compromissos reais carregados do gateway.',
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 16),
        if (events.isEmpty)
          const _AgendaCard(
            icon: Icons.event_busy_rounded,
            title: 'Nenhum evento proximo',
            subtitle: 'Crie compromissos no gateway para preencher sua agenda.',
          ),
        for (final event in events)
          _AgendaCard(
            icon: event.isCompleted ? Icons.check_circle_rounded : Icons.schedule_rounded,
            title: event.title,
            subtitle: _formatEventSubtitle(event),
            onComplete: event.isCompleted ? null : () => _completeEvent(event),
            onEdit: () => _editEvent(event),
            onDelete: () => _deleteEvent(event),
          ),
      ],
    );
  }
}

class _AgendaCard extends StatelessWidget {
  const _AgendaCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    this.onComplete,
    this.onEdit,
    this.onDelete,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback? onComplete;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        boxShadow: const [
          BoxShadow(color: Color(0x14000000), blurRadius: 16, offset: Offset(0, 6)),
        ],
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: const Color(0xFF0E5A8A)),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 4),
                Text(subtitle),
                if (onComplete != null || onEdit != null || onDelete != null) ...[
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      if (onComplete != null)
                        OutlinedButton.icon(
                          onPressed: onComplete,
                          icon: const Icon(Icons.check_rounded),
                          label: const Text('Concluir'),
                        ),
                      if (onEdit != null)
                        OutlinedButton.icon(
                          onPressed: onEdit,
                          icon: const Icon(Icons.edit_rounded),
                          label: const Text('Editar'),
                        ),
                      if (onDelete != null)
                        OutlinedButton.icon(
                          onPressed: onDelete,
                          icon: const Icon(Icons.delete_outline_rounded),
                          label: const Text('Excluir'),
                        ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

String _formatEventSubtitle(HomeEvent event) {
  final range = event.isAllDay
      ? 'Dia todo'
      : '${_formatDateTime(event.startAt.toLocal())} - ${_formatDateTime(event.endAt.toLocal())}';

  if (event.description.trim().isEmpty) {
    return range;
  }

  return '${event.description} • $range';
}

String _formatDateTime(DateTime value) {
  final day = value.day.toString().padLeft(2, '0');
  final month = value.month.toString().padLeft(2, '0');
  final hour = value.hour.toString().padLeft(2, '0');
  final minute = value.minute.toString().padLeft(2, '0');
  return '$day/$month $hour:$minute';
}

Future<EventDraft?> _showEventDialog(
  BuildContext context, {
  HomeEvent? existingEvent,
}) async {
  final messenger = ScaffoldMessenger.of(context);
  final titleController = TextEditingController(text: existingEvent?.title ?? '');
  final descriptionController = TextEditingController(text: existingEvent?.description ?? '');
  DateTime startAt = existingEvent?.startAt.toLocal() ?? DateTime.now().add(const Duration(hours: 1));
  DateTime endAt = existingEvent?.endAt.toLocal() ?? startAt.add(const Duration(hours: 1));
  bool isAllDay = existingEvent?.isAllDay ?? false;

  Future<DateTime?> pickDateTime(DateTime initialValue) async {
    final pickedDate = await showDatePicker(
      context: context,
      initialDate: initialValue,
      firstDate: DateTime.now().subtract(const Duration(days: 365)),
      lastDate: DateTime.now().add(const Duration(days: 3650)),
    );
    if (pickedDate == null) return null;
    if (!context.mounted) return null;

    final pickedTime = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(initialValue),
    );
    if (pickedTime == null) return null;

    return DateTime(
      pickedDate.year,
      pickedDate.month,
      pickedDate.day,
      pickedTime.hour,
      pickedTime.minute,
    );
  }

  final result = await showDialog<EventDraft>(
    context: context,
    builder: (dialogContext) {
      return StatefulBuilder(
        builder: (context, setState) {
          return AlertDialog(
            title: Text(existingEvent == null ? 'Novo evento' : 'Editar evento'),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  TextField(
                    controller: titleController,
                    decoration: const InputDecoration(labelText: 'Titulo'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: descriptionController,
                    decoration: const InputDecoration(labelText: 'Descricao'),
                    minLines: 2,
                    maxLines: 4,
                  ),
                  const SizedBox(height: 12),
                  SwitchListTile.adaptive(
                    contentPadding: EdgeInsets.zero,
                    value: isAllDay,
                    title: const Text('Dia todo'),
                    onChanged: (value) => setState(() => isAllDay = value),
                  ),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    title: const Text('Inicio'),
                    subtitle: Text(_formatDateTime(startAt)),
                    trailing: const Icon(Icons.calendar_today_rounded),
                    onTap: () async {
                      final picked = await pickDateTime(startAt);
                      if (picked != null) {
                        setState(() => startAt = picked);
                        if (!endAt.isAfter(startAt)) {
                          setState(() => endAt = startAt.add(const Duration(hours: 1)));
                        }
                      }
                    },
                  ),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    title: const Text('Fim'),
                    subtitle: Text(_formatDateTime(endAt)),
                    trailing: const Icon(Icons.event_available_rounded),
                    onTap: () async {
                      final picked = await pickDateTime(endAt);
                      if (picked != null) {
                        setState(() => endAt = picked);
                      }
                    },
                  ),
                ],
              ),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(),
                child: const Text('Cancelar'),
              ),
              FilledButton(
                onPressed: () {
                  final title = titleController.text.trim();
                  final description = descriptionController.text.trim();
                  if (title.length < 3) {
                    messenger.showSnackBar(
                      const SnackBar(content: Text('Titulo deve ter pelo menos 3 caracteres.')),
                    );
                    return;
                  }
                  if (!endAt.isAfter(startAt)) {
                    messenger.showSnackBar(
                      const SnackBar(content: Text('Fim deve ser depois do inicio.')),
                    );
                    return;
                  }
                  Navigator.of(dialogContext).pop(
                    EventDraft(
                      title: title,
                      description: description,
                      startAt: startAt,
                      endAt: endAt,
                      timezone: DateTime.now().timeZoneName,
                      isAllDay: isAllDay,
                    ),
                  );
                },
                child: Text(existingEvent == null ? 'Criar' : 'Salvar'),
              ),
            ],
          );
        },
      );
    },
  );

  titleController.dispose();
  descriptionController.dispose();
  return result;
}
