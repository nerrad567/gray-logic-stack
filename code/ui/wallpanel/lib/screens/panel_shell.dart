import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/room.dart';
import '../models/scene.dart';
import '../providers/auth_provider.dart';
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

class _PanelShellState extends ConsumerState<PanelShell>
    with SingleTickerProviderStateMixin {
  final PageController _pageController = PageController();
  int _currentPage = 0;

  // Hidden exit gesture state
  static const _exitTapThreshold = 5;
  static const _exitTapWindow = Duration(seconds: 3);
  int _exitTapCount = 0;
  DateTime? _firstExitTap;
  late AnimationController _pulseController;

  @override
  void initState() {
    super.initState();
    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 300),
      lowerBound: 0.0,
      upperBound: 1.0,
    );
    // Load all devices and scenes once — individual pages filter from this.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(roomDevicesProvider.notifier).loadDevices('__all__');
      ref.read(allScenesProvider.notifier).load();
    });
  }

  @override
  void dispose() {
    _pageController.dispose();
    _pulseController.dispose();
    super.dispose();
  }

  /// Hidden exit: 5 taps within 3 seconds triggers logout.
  /// Each tap provides visual feedback via the exit indicator dot.
  void _handleExitTap() {
    final now = DateTime.now();
    if (_firstExitTap == null ||
        now.difference(_firstExitTap!) > _exitTapWindow) {
      _firstExitTap = now;
      _exitTapCount = 1;
    } else {
      _exitTapCount++;
    }

    // Pulse animation on each tap
    _pulseController.forward(from: 0.0);

    setState(() {});

    if (_exitTapCount >= _exitTapThreshold) {
      _exitTapCount = 0;
      _firstExitTap = null;
      _confirmLogout();
    }
  }

  /// Progress towards logout (0.0 to 1.0).
  double get _exitProgress {
    if (_exitTapCount == 0 || _firstExitTap == null) return 0.0;
    final elapsed = DateTime.now().difference(_firstExitTap!);
    if (elapsed > _exitTapWindow) return 0.0;
    return _exitTapCount / _exitTapThreshold;
  }

  Future<void> _confirmLogout() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Exit Panel Mode'),
        content: const Text('Log out and return to the login screen?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Log Out'),
          ),
        ],
      ),
    );
    if (confirmed == true && mounted) {
      ref.read(authProvider.notifier).logout();
    }
    // Reset visual state if cancelled
    if (mounted) setState(() {});
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
              Icon(Icons.meeting_room_outlined, size: 48,
                  color: Colors.grey.shade600),
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
    final progress = _exitProgress;

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
                _BottomBar(
                  pageCount: pageCount,
                  currentPage: _currentPage,
                  exitProgress: progress,
                  pulseAnimation: _pulseController,
                  onTap: _handleExitTap,
                ),
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

/// Bottom bar combining page dots and the hidden exit indicator.
/// The exit dot starts invisible and becomes more visible/excited with each tap.
class _BottomBar extends AnimatedWidget {
  final int pageCount;
  final int currentPage;
  final double exitProgress;
  final VoidCallback onTap;

  const _BottomBar({
    required this.pageCount,
    required this.currentPage,
    required this.exitProgress,
    required Animation<double> pulseAnimation,
    required this.onTap,
  }) : super(listenable: pulseAnimation);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final pulse = (listenable as Animation<double>).value;

    // Exit dot: starts invisible, grows with each tap
    final dotOpacity = exitProgress > 0 ? 0.2 + (exitProgress * 0.8) : 0.0;
    final dotSize = 6.0 + (exitProgress * 10.0) + (pulse * 4.0);
    final dotColor = Color.lerp(
      theme.colorScheme.onSurface.withValues(alpha: 0.3),
      theme.colorScheme.error,
      exitProgress,
    )!;

    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onTap,
      child: SizedBox(
        height: 28,
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // Page dots (if multiple pages)
            if (pageCount > 1)
              ...List.generate(pageCount, (index) {
                final isActive = index == currentPage;
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

            // Spacer before exit dot (only if dots are showing)
            if (pageCount > 1) const SizedBox(width: 12),

            // Exit indicator dot — invisible until tapped
            AnimatedOpacity(
              duration: const Duration(milliseconds: 150),
              opacity: dotOpacity,
              child: Container(
                width: dotSize,
                height: dotSize,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: dotColor,
                ),
              ),
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
