import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/room.dart';
import '../models/scene.dart';
import '../providers/device_provider.dart';
import '../providers/scene_provider.dart';
import 'device_grid.dart';
import 'scene_button.dart';

/// Master overview page for panel mode when multiple rooms are assigned.
/// Shows all devices grouped by room with inline scene buttons.
class PanelOverviewPage extends ConsumerWidget {
  final List<Room> rooms;

  const PanelOverviewPage({super.key, required this.rooms});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final devicesAsync = ref.watch(roomDevicesProvider);
    final scenesAsync = ref.watch(allScenesProvider);

    final devices = devicesAsync.value ?? [];
    final scenes = scenesAsync.value ?? [];

    return ListView.builder(
      padding: const EdgeInsets.only(top: 16, bottom: 24),
      itemCount: rooms.length + 1, // +1 for header
      itemBuilder: (context, index) {
        if (index == 0) {
          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Text(
              'Overview',
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
            ),
          );
        }

        final room = rooms[index - 1];
        final roomDevices =
            devices.where((d) => d.roomId == room.id).toList();
        final roomScenes = scenes
            .where((s) => s.enabled && s.roomId == room.id)
            .toList();

        return Padding(
          padding: const EdgeInsets.only(bottom: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Text(
                  room.name,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                ),
              ),
              if (roomDevices.isNotEmpty)
                DeviceGrid(devices: roomDevices, shrinkWrap: true),
              if (roomScenes.isNotEmpty)
                _OverviewSceneBar(scenes: roomScenes),
            ],
          ),
        );
      },
    );
  }
}

/// Horizontal scene buttons for the overview page. Activation-only, no editing.
class _OverviewSceneBar extends ConsumerWidget {
  final List<Scene> scenes;

  const _OverviewSceneBar({required this.scenes});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return SizedBox(
      height: 56,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        itemCount: scenes.length,
        separatorBuilder: (_, __) => const SizedBox(width: 8),
        itemBuilder: (_, index) {
          return Center(
            child: SceneButton(
              scene: scenes[index],
              onLongPress: null,
            ),
          );
        },
      ),
    );
  }
}
