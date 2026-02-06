import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/room.dart';
import '../models/scene.dart';
import '../providers/device_provider.dart';
import '../providers/location_provider.dart';
import '../providers/scene_provider.dart';
import '../widgets/connection_indicator.dart';
import '../widgets/device_grid.dart';
import '../widgets/panel_overview_page.dart';
import '../widgets/scene_button.dart';

/// Kiosk-mode shell for wall panels.
///
/// Strips all chrome (no nav bar, no settings, no scene editing) and provides
/// swipe-based room navigation with dot indicators. If the panel has >1 room,
/// the first page is a master overview showing all devices grouped by room.
class PanelShell extends ConsumerStatefulWidget {
  const PanelShell({super.key});

  @override
  ConsumerState<PanelShell> createState() => _PanelShellState();
}

class _PanelShellState extends ConsumerState<PanelShell> {
  final PageController _pageController = PageController();
  int _currentPage = 0;

  @override
  void initState() {
    super.initState();
    // Load all devices and scenes once — individual pages filter from this.
    ref.listenManual(locationDataProvider, (_, __) {});
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(roomDevicesProvider.notifier).loadDevices('__all__');
      ref.read(allScenesProvider.notifier).load();
    });
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final locationAsync = ref.watch(locationDataProvider);
    final rooms = locationAsync.value?.sortedRooms ?? [];

    if (rooms.isEmpty) {
      return Scaffold(
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(Icons.meeting_room_outlined, size: 48, color: Colors.grey.shade600),
              const SizedBox(height: 12),
              Text(
                'No rooms assigned',
                style: TextStyle(fontSize: 16, color: Colors.grey.shade500),
              ),
            ],
          ),
        ),
      );
    }

    final hasOverview = rooms.length > 1;
    final pageCount = (hasOverview ? 1 : 0) + rooms.length;

    return Scaffold(
      body: SafeArea(
        child: Stack(
          children: [
            Column(
              children: [
                Expanded(
                  child: PageView.builder(
                    controller: _pageController,
                    itemCount: pageCount,
                    onPageChanged: (page) => setState(() => _currentPage = page),
                    itemBuilder: (context, index) {
                      if (hasOverview && index == 0) {
                        return PanelOverviewPage(rooms: rooms);
                      }
                      final roomIndex = hasOverview ? index - 1 : index;
                      return _PanelRoomPage(room: rooms[roomIndex]);
                    },
                  ),
                ),
                _PageDots(count: pageCount, current: _currentPage),
              ],
            ),
            const Positioned(
              top: 8,
              right: 8,
              child: ConnectionIndicator(),
            ),
          ],
        ),
      ),
    );
  }
}

/// A single room page showing devices and scene buttons.
class _PanelRoomPage extends ConsumerWidget {
  final Room room;

  const _PanelRoomPage({required this.room});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final devicesAsync = ref.watch(roomDevicesProvider);
    final scenesAsync = ref.watch(allScenesProvider);

    final allDevices = devicesAsync.value ?? [];
    final allScenes = scenesAsync.value ?? [];

    final roomDevices =
        allDevices.where((d) => d.roomId == room.id).toList();
    final roomScenes =
        allScenes.where((s) => s.enabled && s.roomId == room.id).toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Text(
            room.name,
            style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
          ),
        ),
        Expanded(
          child: DeviceGrid(devices: roomDevices),
        ),
        if (roomScenes.isNotEmpty) _PanelSceneBar(scenes: roomScenes),
      ],
    );
  }
}

/// Horizontal scene activation bar for panel mode. No add button, no editing.
class _PanelSceneBar extends ConsumerWidget {
  final List<Scene> scenes;

  const _PanelSceneBar({required this.scenes});

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

/// Dot indicators for PageView navigation.
class _PageDots extends StatelessWidget {
  final int count;
  final int current;

  const _PageDots({required this.count, required this.current});

  @override
  Widget build(BuildContext context) {
    if (count <= 1) return const SizedBox(height: 8);

    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.only(bottom: 8, top: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: List.generate(count, (index) {
          final isActive = index == current;
          return AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            margin: const EdgeInsets.symmetric(horizontal: 3),
            width: isActive ? 12 : 6,
            height: 6,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(3),
              color: isActive
                  ? theme.colorScheme.primary
                  : theme.colorScheme.onSurface.withValues(alpha: 0.3),
            ),
          );
        }),
      ),
    );
  }
}
