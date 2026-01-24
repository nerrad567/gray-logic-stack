import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/location_provider.dart';

/// Horizontal scrollable navigation bar showing rooms grouped by area.
/// Tapping a room pill switches the view to that room's devices.
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
        );
      },
      loading: () => const SizedBox(height: 52),
      error: (_, __) => const SizedBox.shrink(),
    );
  }
}
