import 'package:flutter/material.dart';

import '../../../../app/routes/app_routes.dart';
import '../../../auth/presentation/state/auth_controller.dart';
import 'agenda_page.dart';
import 'profile_page.dart';
import 'tasks_page.dart';
import '../state/home_controller.dart';
import '../state/task_board_controller.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key, required this.authController});

  final AuthController authController;

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _index = 0;
  late final HomeController _homeController;
  TaskBoardController? _taskBoardController;

  @override
  void initState() {
    super.initState();
    _homeController = HomeController(apiClient: widget.authController.apiClient);
    _taskBoardController = TaskBoardController();
    _taskBoardController!.load();
    _homeController.load();
  }

  @override
  void dispose() {
    _homeController.dispose();
    _taskBoardController?.dispose();
    super.dispose();
  }

  Future<void> _logout() async {
    await widget.authController.logout();
    if (!mounted) return;
    Navigator.pushNamedAndRemoveUntil(context, AppRoutes.login, (_) => false);
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _homeController,
      builder: (context, _) {
        final profile = _homeController.profile;
        final taskBoardController = _taskBoardController ??= TaskBoardController();
        final pages = [
          _DashboardTab(controller: _homeController),
          AgendaPage(controller: _homeController),
          TasksPage(controller: taskBoardController),
          ProfilePage(controller: _homeController),
        ];

        return Scaffold(
          appBar: AppBar(
            title: Text('NeuroLife • ${profile?.name ?? widget.authController.user?.name ?? 'Usuario'}'),
            actions: [
              IconButton(
                tooltip: 'Atualizar',
                onPressed: _homeController.isLoading ? null : _homeController.load,
                icon: const Icon(Icons.refresh_rounded),
              ),
              IconButton(
                tooltip: 'Sair',
                onPressed: _logout,
                icon: const Icon(Icons.logout_rounded),
              ),
            ],
          ),
          body: _HomeBody(
            controller: _homeController,
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 240),
              child: pages[_index],
            ),
          ),
          bottomNavigationBar: NavigationBar(
            selectedIndex: _index,
            onDestinationSelected: (value) => setState(() => _index = value),
            destinations: const [
              NavigationDestination(icon: Icon(Icons.grid_view_rounded), label: 'Visao geral'),
              NavigationDestination(icon: Icon(Icons.event_note_rounded), label: 'Agenda'),
              NavigationDestination(icon: Icon(Icons.checklist_rounded), label: 'Tarefas'),
              NavigationDestination(icon: Icon(Icons.person_rounded), label: 'Perfil'),
            ],
          ),
        );
      },
    );
  }
}

class _HomeBody extends StatelessWidget {
  const _HomeBody({required this.controller, required this.child});

  final HomeController controller;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (controller.isLoading && controller.profile == null && controller.summary == null) {
      return const Center(child: CircularProgressIndicator());
    }

    if (controller.errorMessage != null && controller.profile == null && controller.summary == null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.cloud_off_rounded, size: 36),
              const SizedBox(height: 12),
              Text(
                controller.errorMessage!,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 12),
              ElevatedButton(
                onPressed: controller.load,
                child: const Text('Tentar novamente'),
              ),
            ],
          ),
        ),
      );
    }

    return child;
  }
}

class _DashboardTab extends StatelessWidget {
  const _DashboardTab({required this.controller});

  final HomeController controller;

  @override
  Widget build(BuildContext context) {
    final summary = controller.summary;
    final nextEvents = controller.events;

    return ListView(
      key: const ValueKey('dashboard'),
      padding: const EdgeInsets.all(20),
      children: [
        _HeroCard(
          name: controller.profile?.name ?? 'Usuario',
          summary: summary,
        ),
        const SizedBox(height: 16),
        const _SectionTitle('Resumo do dia'),
        const SizedBox(height: 8),
        _QuickTile(
          icon: Icons.event_available_rounded,
          title: 'Eventos proximos',
          subtitle: nextEvents.isEmpty ? 'Nenhum evento carregado.' : '${nextEvents.length} eventos nos proximos dias.',
        ),
        _QuickTile(
          icon: Icons.local_fire_department_rounded,
          title: 'Sequencia ativa',
          subtitle: '${summary?.currentStreak ?? 0} dias seguidos com consistencia.',
        ),
        _QuickTile(
          icon: Icons.workspace_premium_rounded,
          title: 'Conquistas desbloqueadas',
          subtitle: '${controller.achievements.length} marcos registrados no seu perfil.',
        ),
        if (controller.errorMessage != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Text(
              controller.errorMessage!,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Theme.of(context).colorScheme.error),
            ),
          ),
      ],
    );
  }
}

class _HeroCard extends StatelessWidget {
  const _HeroCard({required this.name, required this.summary});

  final String name;
  final HomeGamificationSummary? summary;

  @override
  Widget build(BuildContext context) {
    final streak = summary?.currentStreak ?? 0;
    final level = summary?.level ?? 1;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF0E5A8A), Color(0xFF4A90C2)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(22),
        boxShadow: const [
          BoxShadow(color: Color(0x280E5A8A), blurRadius: 22, offset: Offset(0, 8)),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Bom te ver de novo', style: TextStyle(color: Colors.white70, fontSize: 14)),
          SizedBox(height: 6),
          Text(name, style: TextStyle(color: Colors.white, fontSize: 24, fontWeight: FontWeight.w800)),
          SizedBox(height: 12),
          Text('Nivel $level • Sequencia atual de $streak dias.', style: TextStyle(color: Colors.white70)),
        ],
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
    );
  }
}

class _QuickTile extends StatelessWidget {
  const _QuickTile({required this.icon, required this.title, required this.subtitle});

  final IconData icon;
  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Icon(icon, color: const Color(0xFF0E5A8A)),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 2),
                Text(subtitle),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
