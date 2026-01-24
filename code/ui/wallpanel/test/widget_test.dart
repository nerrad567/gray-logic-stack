import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:wallpanel/main.dart';

void main() {
  testWidgets('App renders without crashing', (WidgetTester tester) async {
    await tester.pumpWidget(
      const ProviderScope(child: GrayLogicPanel()),
    );

    // App should show a loading indicator while restoring session
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });
}
