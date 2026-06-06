import 'package:flutter/material.dart';

import '../../../../app/routes/app_routes.dart';
import '../../domain/auth_validators.dart';
import '../state/auth_controller.dart';
import '../widgets/auth_scaffold.dart';
import '../widgets/auth_text_field.dart';

class ResetPasswordPage extends StatefulWidget {
  const ResetPasswordPage({
    super.key,
    required this.authController,
    this.initialToken,
  });

  final AuthController authController;
  final String? initialToken;

  @override
  State<ResetPasswordPage> createState() => _ResetPasswordPageState();
}

class _ResetPasswordPageState extends State<ResetPasswordPage> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _tokenController;
  final _passwordController = TextEditingController();
  final _confirmController = TextEditingController();
  bool _isSubmitting = false;

  @override
  void initState() {
    super.initState();
    _tokenController = TextEditingController(text: widget.initialToken ?? '');
  }

  @override
  void dispose() {
    _tokenController.dispose();
    _passwordController.dispose();
    _confirmController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;

    setState(() => _isSubmitting = true);
    try {
      await widget.authController.resetPassword(
        token: _tokenController.text,
        password: _passwordController.text,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Senha redefinida com sucesso.')),
      );
      Navigator.pushNamedAndRemoveUntil(context, AppRoutes.home, (_) => false);
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(error.toString())),
      );
    } finally {
      if (mounted) {
        setState(() => _isSubmitting = false);
      }
    }
  }

  String? _validateToken(String? value) {
    final text = (value ?? '').trim();
    if (text.isEmpty) return 'Informe o token de redefinicao.';
    if (text.length < 12) return 'Token invalido.';
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return AuthScaffold(
      title: 'Redefinir senha',
      subtitle: 'Cole o token recebido e escolha uma nova senha segura.',
      child: Form(
        key: _formKey,
        child: Column(
          children: [
            AuthTextField(
              label: 'Token de redefinicao',
              hint: 'Cole o token gerado pelo backend',
              controller: _tokenController,
              validator: _validateToken,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: 12),
            AuthTextField(
              label: 'Nova senha',
              hint: 'Minimo de 8 caracteres',
              controller: _passwordController,
              validator: AuthValidators.password,
              obscureText: true,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: 12),
            AuthTextField(
              label: 'Confirmar nova senha',
              hint: 'Repita a nova senha',
              controller: _confirmController,
              validator: (value) => AuthValidators.confirmPassword(value, _passwordController.text),
              obscureText: true,
              textInputAction: TextInputAction.done,
            ),
            const SizedBox(height: 20),
            ElevatedButton(
              onPressed: _isSubmitting ? null : _submit,
              child: _isSubmitting
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Redefinir senha'),
            ),
          ],
        ),
      ),
    );
  }
}
