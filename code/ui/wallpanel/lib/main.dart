import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'config/theme.dart';
import 'screens/app_shell.dart';

void main() {
  // Disable the browser's long-press context menu — this is a wall panel app,
  // never needs right-click/long-press browser menus.
  if (kIsWeb) {
    BrowserContextMenu.disableContextMenu();
  }
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
