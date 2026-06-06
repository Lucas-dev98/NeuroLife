// This is a basic Flutter widget test.
//
// To perform an interaction with a widget in your test, use the WidgetTester
// utility in the flutter_test package. For example, you can send tap and scroll
// gestures. You can also use WidgetTester to find child widgets in the widget
// tree, read text, and verify that the values of widget properties are correct.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:mobile_flutter/app/app.dart';
import 'package:mobile_flutter/app/routes/app_routes.dart';

void main() {
  testWidgets('shows login when there is no session', (WidgetTester tester) async {
    SharedPreferences.setMockInitialValues({});
    await tester.pumpWidget(const NeuroLifeApp());
    await tester.pumpAndSettle();

    expect(find.text('Entrar na NeuroLife'), findsOneWidget);
    expect(find.text('Criar conta'), findsOneWidget);
  });

  testWidgets('blocks navigation to home when unauthenticated', (WidgetTester tester) async {
    SharedPreferences.setMockInitialValues({});
    await tester.pumpWidget(const NeuroLifeApp());
    await tester.pumpAndSettle();

    final navigator = tester.state<NavigatorState>(find.byType(Navigator).first);
    navigator.pushNamed(AppRoutes.home);
    await tester.pumpAndSettle();

    expect(find.text('Entrar na NeuroLife'), findsOneWidget);
    expect(find.textContaining('Bom te ver de novo'), findsNothing);
  });
}
