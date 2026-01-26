import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/auth_provider.dart';
import '../providers/location_provider.dart';
import '../screens/ets_import_screen.dart';

/// Horizontal scrollable navigation bar showing rooms grouped by area.
/// Tapping a room pill switches the view to that room's devices.
/// Includes a settings menu for commissioning tools.
class RoomNavBar extends ConsumerWidget {
  const RoomNavBar({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final locationAsync = ref.watch(locationDataProvider);
    final selectedRoom = ref.watch(selectedRoomProvider);

    return locationAsync.when(
      data: (data) {
        final roomsByArea = data.roomsByArea;
        if (roomsByArea.isEmpty) return const SizedBox.shrink();

        return Container(
          height: 52,
          decoration: BoxDecoration(
            border: Border(
              bottom: BorderSide(
                color: Theme.of(context).dividerColor.withValues(alpha: 0.3),
              ),
            ),
          ),
          child: Row(
            children: [
              // Room pills (scrollable)
              Expanded(
                child: ListView(
                  scrollDirection: Axis.horizontal,
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                  children: [
                    for (final entry in roomsByArea.entries) ...[
                      // Area label
                      Center(
                        child: Padding(
                          padding: const EdgeInsets.only(right: 8),
                          child: Text(
                            entry.key.name,
                            style: Theme.of(context).textTheme.labelSmall?.copyWith(
                                  color: Colors.grey.shade500,
                                  fontWeight: FontWeight.w600,
                                  letterSpacing: 0.5,
                                ),
                          ),
                        ),
                      ),
                      // Room pills
                      for (final room in entry.value)
                        Padding(
                          padding: const EdgeInsets.only(right: 6),
                          child: ChoiceChip(
                            label: Text(room.name),
                            selected: room.id == selectedRoom,
                            onSelected: (_) {
                              ref.read(selectedRoomProvider.notifier).state = room.id;
                            },
                            labelStyle: TextStyle(
                              fontSize: 13,
                              color: room.id == selectedRoom
                                  ? Theme.of(context).colorScheme.onPrimary
                                  : null,
                            ),
                            selectedColor: Theme.of(context).colorScheme.primary,
                            visualDensity: VisualDensity.compact,
                          ),
                        ),
                      // Separator between areas
                      if (entry.key != roomsByArea.keys.last)
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 8),
                          child: Center(
                            child: Container(
                              width: 1,
                              height: 20,
                              color: Theme.of(context).dividerColor.withValues(alpha: 0.3),
                            ),
                          ),
                        ),
                    ],
                  ],
                ),
              ),
              // Settings menu
              _SettingsMenu(ref: ref),
            ],
          ),
        );
      },
      loading: () => const SizedBox(height: 52),
      error: (_, _) => const SizedBox.shrink(),
    );
  }
}

/// Settings/tools menu button with commissioning options.
class _SettingsMenu extends StatelessWidget {
  final WidgetRef ref;

  const _SettingsMenu({required this.ref});

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<String>(
      icon: Icon(
        Icons.settings_outlined,
        color: Theme.of(context).colorScheme.onSurfaceVariant,
      ),
      tooltip: 'Settings',
      onSelected: (value) => _handleMenuSelection(context, value),
      itemBuilder: (context) => [
        const PopupMenuItem(
          value: 'import_ets',
          child: ListTile(
            leading: Icon(Icons.upload_file_outlined),
            title: Text('Import KNX Devices'),
            subtitle: Text('From ETS project'),
            dense: true,
            contentPadding: EdgeInsets.zero,
          ),
        ),
        const PopupMenuDivider(),
        const PopupMenuItem(
          value: 'refresh',
          child: ListTile(
            leading: Icon(Icons.refresh_outlined),
            title: Text('Refresh'),
            dense: true,
            contentPadding: EdgeInsets.zero,
          ),
        ),
        const PopupMenuItem(
          value: 'logout',
          child: ListTile(
            leading: Icon(Icons.logout_outlined),
            title: Text('Logout'),
            dense: true,
            contentPadding: EdgeInsets.zero,
          ),
        ),
      ],
    );
  }

  void _handleMenuSelection(BuildContext context, String value) {
    switch (value) {
      case 'import_ets':
        Navigator.of(context).push(
          MaterialPageRoute(
            builder: (context) => const ETSImportScreen(),
          ),
        );
        break;
      case 'refresh':
        ref.read(locationDataProvider.notifier).load();
        break;
      case 'logout':
        ref.read(authProvider.notifier).logout();
        break;
    }
  }
}
