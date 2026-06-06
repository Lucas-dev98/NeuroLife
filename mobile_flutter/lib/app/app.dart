import 'package:flutter/material.dart';

import 'routes/app_routes.dart';
import 'theme/app_theme.dart';
import '../features/auth/presentation/state/auth_controller.dart';
import '../features/auth/presentation/pages/forgot_password_page.dart';
import '../features/auth/presentation/pages/login_page.dart';
import '../features/auth/presentation/pages/register_page.dart';
import '../features/auth/presentation/pages/reset_password_page.dart';
import '../features/home/presentation/pages/home_page.dart';

class NeuroLifeApp extends StatefulWidget {
  const NeuroLifeApp({super.key, this.authController});

  final AuthController? authController;

  @override
  State<NeuroLifeApp> createState() => _NeuroLifeAppState();
}

class _NeuroLifeAppState extends State<NeuroLifeApp> {
  late final AuthController _authController;
  late final bool _ownsAuthController;

  @override
  void initState() {
    super.initState();
    _ownsAuthController = widget.authController == null;
    _authController = widget.authController ?? AuthController();
    _authController.initialize();
  }

  @override
  void dispose() {
    if (_ownsAuthController) {
      _authController.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _authController,
      builder: (context, _) {
        if (_authController.isInitializing) {
          return MaterialApp(
            debugShowCheckedModeBanner: false,
            theme: AppTheme.light,
            home: const _AppLoadingScreen(),
          );
        }

        return MaterialApp(
          key: ValueKey(_authController.isAuthenticated),
          title: 'NeuroLife',
          debugShowCheckedModeBanner: false,
          theme: AppTheme.light,
          initialRoute: _authController.isAuthenticated ? AppRoutes.home : AppRoutes.login,
          onGenerateRoute: (settings) {
            final routeName = settings.name ?? AppRoutes.login;

            if (routeName == AppRoutes.home && !_authController.isAuthenticated) {
              return MaterialPageRoute<void>(
                builder: (_) => LoginPage(authController: _authController),
                settings: const RouteSettings(name: AppRoutes.login),
              );
            }

            switch (routeName) {
              case AppRoutes.login:
                return MaterialPageRoute<void>(
                  builder: (_) => LoginPage(authController: _authController),
                  settings: const RouteSettings(name: AppRoutes.login),
                );
              case AppRoutes.register:
                return MaterialPageRoute<void>(
                  builder: (_) => RegisterPage(authController: _authController),
                  settings: const RouteSettings(name: AppRoutes.register),
                );
              case AppRoutes.forgotPassword:
                return MaterialPageRoute<void>(
                  builder: (_) => ForgotPasswordPage(authController: _authController),
                  settings: const RouteSettings(name: AppRoutes.forgotPassword),
                );
              case AppRoutes.resetPassword:
                return MaterialPageRoute<void>(
                  builder: (_) => ResetPasswordPage(
                    authController: _authController,
                    initialToken: settings.arguments as String?,
                  ),
                  settings: const RouteSettings(name: AppRoutes.resetPassword),
                );
              case AppRoutes.home:
                return MaterialPageRoute<void>(
                  builder: (_) => HomePage(authController: _authController),
                  settings: const RouteSettings(name: AppRoutes.home),
                );
              default:
                return MaterialPageRoute<void>(
                  builder: (_) => LoginPage(authController: _authController),
                  settings: const RouteSettings(name: AppRoutes.login),
                );
            }
          },
        );
      },
    );
  }
}

class _AppLoadingScreen extends StatelessWidget {
  const _AppLoadingScreen();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(
        child: CircularProgressIndicator(),
      ),
    );
  }
}
