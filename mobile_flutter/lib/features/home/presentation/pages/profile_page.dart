import 'package:flutter/material.dart';

import '../state/home_controller.dart';
import 'profile_settings_page.dart';

class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key, required this.controller});

  final HomeController controller;

  @override
  Widget build(BuildContext context) {
    final profile = controller.profile;
    final summary = controller.summary;
    final preferences = controller.preferences;
    final achievements = controller.achievements.take(5).toList();

    return ListView(
      key: const ValueKey('profile'),
      padding: const EdgeInsets.all(20),
      children: [
        Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(18),
            boxShadow: const [
              BoxShadow(color: Color(0x14000000), blurRadius: 16, offset: Offset(0, 6)),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const CircleAvatar(radius: 30, child: Icon(Icons.person_rounded, size: 32)),
              const SizedBox(height: 12),
              Text(
                profile?.name ?? 'Usuario NeuroLife',
                style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 4),
              Text(profile?.email ?? 'Sem e-mail carregado'),
              const SizedBox(height: 8),
              Text('Nivel atual: ${summary?.level ?? 1} • XP: ${summary?.xp ?? 0}'),
              const SizedBox(height: 12),
              FilledButton.icon(
                onPressed: profile == null
                    ? null
                    : () {
                        Navigator.of(context).push(
                          MaterialPageRoute(
                            builder: (_) => ProfileSettingsPage(controller: controller),
                          ),
                        );
                      },
                icon: const Icon(Icons.edit_rounded),
                label: const Text('Editar perfil e preferencias'),
              ),
              if (profile != null) ...[
                const SizedBox(height: 4),
                Text('ID do usuario: ${profile.id}'),
              ],
            ],
          ),
        ),
        const SizedBox(height: 16),
        Text(
          'Preferencias',
          style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
        ),
        const SizedBox(height: 8),
        _PreferenceCard(
          reminderIntensity: preferences.reminderIntensity,
          pushEnabled: preferences.pushEnabled,
          emailEnabled: preferences.emailEnabled,
          whatsappEnabled: preferences.whatsappEnabled,
        ),
        const SizedBox(height: 16),
        Text(
          'Conquistas',
          style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
        ),
        const SizedBox(height: 8),
        if (achievements.isEmpty)
          const _AchievementCard(
            title: 'Sem conquistas ainda',
            description: 'Complete eventos para desbloquear progresso e marcos.',
          ),
        for (final achievement in achievements)
          _AchievementCard(
            title: achievement.title,
            description: achievement.description,
          ),
      ],
    );
  }
}

class _PreferenceCard extends StatelessWidget {
  const _PreferenceCard({
    required this.reminderIntensity,
    required this.pushEnabled,
    required this.emailEnabled,
    required this.whatsappEnabled,
  });

  final String reminderIntensity;
  final bool pushEnabled;
  final bool emailEnabled;
  final bool whatsappEnabled;

  @override
  Widget build(BuildContext context) {
    final intensityLabel = switch (reminderIntensity.toLowerCase()) {
      'low' => 'Baixa',
      'high' => 'Alta',
      _ => 'Media',
    };

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(18),
        boxShadow: const [
          BoxShadow(color: Color(0x14000000), blurRadius: 16, offset: Offset(0, 6)),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Intensidade dos lembretes: $intensityLabel', style: const TextStyle(fontWeight: FontWeight.w700)),
          const SizedBox(height: 10),
          _PreferenceLine(label: 'Push', value: pushEnabled ? 'Ativado' : 'Desativado'),
          _PreferenceLine(label: 'E-mail', value: emailEnabled ? 'Ativado' : 'Desativado'),
          _PreferenceLine(label: 'WhatsApp', value: whatsappEnabled ? 'Ativado' : 'Desativado'),
        ],
      ),
    );
  }
}

class _PreferenceLine extends StatelessWidget {
  const _PreferenceLine({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w700)),
        ],
      ),
    );
  }
}

class _AchievementCard extends StatelessWidget {
  const _AchievementCard({required this.title, required this.description});

  final String title;
  final String description;

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
          const Icon(Icons.workspace_premium_rounded, color: Color(0xFF0E5A8A)),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 4),
                Text(description),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
