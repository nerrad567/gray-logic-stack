import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/device_provider.dart';
import '../providers/location_provider.dart';
import '../providers/scene_provider.dart';
import '../widgets/connection_indicator.dart';
import '../widgets/device_grid.dart';
import '../widgets/scene_bar.dart';

/// Main panel screen showing devices and scenes for the configured room.
/// Layout: header (room name + connection status) → device grid → scene bar.
class RoomView extends ConsumerStatefulWidget {
  final String roomId;

  const RoomView({super.key, required this.roomId});

  @override
  ConsumerState<RoomView> createState() => _RoomViewState();
}

class _RoomViewState extends ConsumerState<RoomView> {
  @override
  void initState() {
    super.initState();
    // Load devices and scenes when the view mounts
    Future.microtask(() {
      ref.read(roomDevicesProvider.notifier).loadDevices(widget.roomId);
      ref.read(roomScenesProvider.notifier).loadScenes(widget.roomId);
    });
  }

  @override
  Widget build(BuildContext context) {
    final devicesAsync = ref.watch(roomDevicesProvider);
    final scenesAsync = ref.watch(roomScenesProvider);
    final locationData = ref.watch(locationDataProvider).valueOrNull;

    // Resolve room name from location data
    final String roomName;
    if (widget.roomId == '__all__') {
      roomName = 'All Devices';
    } else {
      roomName = locationData?.rooms
              .where((r) => r.id == widget.roomId)
              .map((r) => r.name)
              .firstOrNull ??
          widget.roomId;
    }

    return Column(
      children: [
        // Header: room name + connection indicator
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 12, 24, 8),
          child: Row(
            children: [
              Text(
                roomName,
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const Spacer(),
              const ConnectionIndicator(),
            ],
          ),
        ),

        // Device grid (main content area)
        Expanded(
          child: devicesAsync.when(
            data: (devices) => DeviceGrid(devices: devices),
            loading: () => const Center(
              child: CircularProgressIndicator(),
            ),
            error: (error, _) => Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.error_outline,
                    size: 48,
                    color: Theme.of(context).colorScheme.error,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Failed to load devices',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 8),
                  TextButton.icon(
                    onPressed: () => ref
                        .read(roomDevicesProvider.notifier)
                        .loadDevices(widget.roomId),
                    icon: const Icon(Icons.refresh),
                    label: const Text('Retry'),
                  ),
                ],
              ),
            ),
          ),
        ),

        // Scene bar (bottom)
        if (scenesAsync.valueOrNull?.isNotEmpty ?? false)
          const Padding(
            padding: EdgeInsets.only(bottom: 16),
            child: SceneBar(),
          ),
      ],
    );
  }
}
