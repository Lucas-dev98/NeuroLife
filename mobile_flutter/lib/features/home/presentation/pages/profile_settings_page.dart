import 'package:flutter/material.dart';

import '../../../auth/domain/auth_validators.dart';
import '../state/home_controller.dart';

class ProfileSettingsPage extends StatefulWidget {
  const ProfileSettingsPage({super.key, required this.controller});

  final HomeController controller;

  @override
  State<ProfileSettingsPage> createState() => _ProfileSettingsPageState();
}

class _ProfileSettingsPageState extends State<ProfileSettingsPage> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameController;
  late String _reminderIntensity;
  late bool _pushEnabled;
  late bool _emailEnabled;
  late bool _whatsappEnabled;
  bool _isSaving = false;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.controller.profile?.name ?? '');
    _reminderIntensity = widget.controller.preferences.reminderIntensity;
    _pushEnabled = widget.controller.preferences.pushEnabled;
    _emailEnabled = widget.controller.preferences.emailEnabled;
    _whatsappEnabled = widget.controller.preferences.whatsappEnabled;
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    setState(() => _isSaving = true);
    try {
      await widget.controller.updateProfile(_nameController.text);
      await widget.controller.updatePreferences(
        HomePreferences(
          reminderIntensity: _reminderIntensity,
          pushEnabled: _pushEnabled,
          emailEnabled: _emailEnabled,
          whatsappEnabled: _whatsappEnabled,
        ),
      );
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Perfil e preferencias atualizados.')),
      );
      Navigator.of(context).pop();
    } catch (error) {
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Nao foi possivel salvar: $error')),
      );
    } finally {
      if (mounted) {
        setState(() => _isSaving = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(title: const Text('Editar perfil e preferencias')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Material(
            color: Colors.white,
            elevation: 6,
            shadowColor: const Color(0x14000000),
            borderRadius: BorderRadius.circular(18),
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Form(
                key: _formKey,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Perfil',
                      style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _nameController,
                      enabled: !_isSaving,
                      decoration: const InputDecoration(
                        labelText: 'Nome',
                        prefixIcon: Icon(Icons.person_rounded),
                      ),
                      validator: AuthValidators.name,
                    ),
                    const SizedBox(height: 20),
                    Text(
                      'Preferencias de notificacao',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: _reminderIntensity,
                      onChanged: _isSaving ? null : (value) => setState(() => _reminderIntensity = value ?? 'medium'),
                      decoration: const InputDecoration(
                        labelText: 'Intensidade dos lembretes',
                        prefixIcon: Icon(Icons.alarm_rounded),
                      ),
                      items: const [
                        DropdownMenuItem(value: 'low', child: Text('Baixa')),
                        DropdownMenuItem(value: 'medium', child: Text('Media')),
                        DropdownMenuItem(value: 'high', child: Text('Alta')),
                      ],
                    ),
                    const SizedBox(height: 8),
                    SwitchListTile.adaptive(
                      value: _pushEnabled,
                      onChanged: _isSaving ? null : (value) => setState(() => _pushEnabled = value),
                      title: const Text('Push'),
                      subtitle: const Text('Receber notificacoes no aplicativo.'),
                      activeThumbColor: scheme.primary,
                    ),
                    SwitchListTile.adaptive(
                      value: _emailEnabled,
                      onChanged: _isSaving ? null : (value) => setState(() => _emailEnabled = value),
                      title: const Text('E-mail'),
                      subtitle: const Text('Receber alertas por e-mail.'),
                      activeThumbColor: scheme.primary,
                    ),
                    SwitchListTile.adaptive(
                      value: _whatsappEnabled,
                      onChanged: _isSaving ? null : (value) => setState(() => _whatsappEnabled = value),
                      title: const Text('WhatsApp'),
                      subtitle: const Text('Receber alertas no WhatsApp.'),
                      activeThumbColor: scheme.primary,
                    ),
                    const SizedBox(height: 10),
                    Row(
                      children: [
                        Expanded(
                          child: OutlinedButton(
                            onPressed: _isSaving ? null : () => Navigator.of(context).pop(),
                            child: const Text('Cancelar'),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: FilledButton(
                            onPressed: _isSaving ? null : _save,
                            child: _isSaving
                                ? const SizedBox(
                                    height: 18,
                                    width: 18,
                                    child: CircularProgressIndicator(strokeWidth: 2),
                                  )
                                : const Text('Salvar'),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}