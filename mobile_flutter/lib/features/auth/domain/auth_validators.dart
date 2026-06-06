class AuthValidators {
  static String? name(String? value) {
    final text = (value ?? '').trim();
    if (text.isEmpty) return 'Informe seu nome.';
    if (text.length < 3) return 'Nome deve ter pelo menos 3 caracteres.';
    return null;
  }

  static String? email(String? value) {
    final text = (value ?? '').trim();
    if (text.isEmpty) return 'Informe seu e-mail.';
    final regex = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$');
    if (!regex.hasMatch(text)) return 'E-mail invalido.';
    return null;
  }

  static String? password(String? value) {
    final text = value ?? '';
    if (text.isEmpty) return 'Informe sua senha.';
    if (text.length < 8) return 'Senha deve ter pelo menos 8 caracteres.';
    return null;
  }

  static String? confirmPassword(String? value, String original) {
    if ((value ?? '').isEmpty) return 'Confirme sua senha.';
    if (value != original) return 'As senhas nao coincidem.';
    return null;
  }
}
