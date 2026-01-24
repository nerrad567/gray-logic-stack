import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'config/theme.dart';
import 'screens/app_shell.dart';

void main() {
  runApp(const ProviderScope(child: GrayLogicPanel()));
}

class GrayLogicPanel extends StatelessWidget {
  const GrayLogicPanel({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Gray Logic Panel',
      debugShowCheckedModeBanner: false,
      theme: wallPanelTheme(),
      home: const AppShell(),
    );
  }
}
